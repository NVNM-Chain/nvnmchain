package nameindex_test

import (
	"fmt"
	"math"
	"path/filepath"
	"testing"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/nameindex"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"
)

const (
	exact    = types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_EXACT
	prefix   = types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_PREFIX
	suffix   = types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_SUFFIX
	contains = types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_CONTAINS
)

func newStore(t *testing.T) *nameindex.Store {
	t.Helper()
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	s, err := nameindex.Open(filepath.Join(t.TempDir(), "test.db"), cdc)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func names(t *testing.T, regs []*types.Registry) []string {
	t.Helper()
	out := make([]string, len(regs))
	for i, r := range regs {
		out[i] = r.Name
	}
	return out
}

func TestStore_MatchModes(t *testing.T) {
	s := newStore(t)

	seed := []*types.Registry{
		{Id: 1, Name: "kyc_registry"},
		{Id: 2, Name: "aml_registry"},
		{Id: 3, Name: "KYC_Extended"},
		{Id: 4, Name: "unrelated"},
	}
	for _, r := range seed {
		require.NoError(t, s.Upsert(r))
	}

	t.Run("exact is case-insensitive", func(t *testing.T) {
		got, _, err := s.Search(exact, "kyc_registry", 0, 50, 0)
		require.NoError(t, err)
		require.Equal(t, []string{"kyc_registry"}, names(t, got))
	})

	t.Run("prefix", func(t *testing.T) {
		got, _, err := s.Search(prefix, "kyc", 0, 50, 0)
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"kyc_registry", "KYC_Extended"}, names(t, got))
	})

	t.Run("suffix", func(t *testing.T) {
		got, _, err := s.Search(suffix, "registry", 0, 50, 0)
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"kyc_registry", "aml_registry"}, names(t, got))
	})

	t.Run("contains", func(t *testing.T) {
		got, _, err := s.Search(contains, "_reg", 0, 50, 0)
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"kyc_registry", "aml_registry"}, names(t, got))
	})

	t.Run("no match", func(t *testing.T) {
		got, _, err := s.Search(contains, "zzz", 0, 50, 0)
		require.NoError(t, err)
		require.Empty(t, got)
	})
}

// TestStore_CaseInsensitiveAcrossModes pins that every mode folds case on
// both sides of the comparison: a mixed-case stored name is found by
// queries in a different case, in every mode, not just prefix (where the
// other MatchModes tests happen to exercise it incidentally via seed data).
func TestStore_CaseInsensitiveAcrossModes(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: "Kyc_Registry"}))

	testCases := []struct {
		mode  types.RegistryNameMatchMode
		query string
	}{
		{exact, "KYC_REGISTRY"},
		{prefix, "KYC"},
		{suffix, "REGISTRY"},
		{contains, "C_REG"},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("mode=%d query=%s", tc.mode, tc.query), func(t *testing.T) {
			got, _, err := s.Search(tc.mode, tc.query, 0, 50, 0)
			require.NoError(t, err)
			require.Equal(t, []string{"Kyc_Registry"}, names(t, got))
		})
	}
}

// TestStore_UnknownModeIsExact pins the default arm: an unspecified mode, which
// is what a caller gets by leaving the field out, and any out-of-range value
// both resolve to exact, the narrowest mode, never to contains.
func TestStore_UnknownModeIsExact(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: "kyc_registry"}))

	for _, mode := range []types.RegistryNameMatchMode{
		types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_UNSPECIFIED,
		types.RegistryNameMatchMode(99),
	} {
		got, _, err := s.Search(mode, "kyc_registry", 0, 50, 0)
		require.NoError(t, err)
		require.Equal(t, []string{"kyc_registry"}, names(t, got), mode.String())

		got, _, err = s.Search(mode, "kyc", 0, 50, 0)
		require.NoError(t, err)
		require.Empty(t, got, "%s must not widen to a substring match", mode)
	}
}

func TestStore_LikeMetacharactersAreLiteral(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: "50%_off"}))
	require.NoError(t, s.Upsert(&types.Registry{Id: 2, Name: "50X_off"}))

	// A literal "%" or "_" in the query must not act as a SQL LIKE wildcard.
	got, _, err := s.Search(contains, "%_off", 0, 50, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"50%_off"}, names(t, got))
}

func TestStore_ContainsNeedsThreeCharacters(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: "kyc_registry"}))
	require.NoError(t, s.Upsert(&types.Registry{Id: 2, Name: "café"}))

	_, _, err := s.Search(contains, "ky", 0, 50, 0)
	require.ErrorIs(t, err, nameindex.ErrContainsTooShort)

	// The floor is counted in characters, not bytes: "afé" is 3 runes.
	got, _, err := s.Search(contains, "afé", 0, 50, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"café"}, names(t, got))

	// The other modes have no minimum.
	got, _, err = s.Search(prefix, "k", 0, 50, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"kyc_registry"}, names(t, got))
}

func TestStore_ContainsTreatsFTSSyntaxAsText(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: `say "hi" OR NOT`}))
	require.NoError(t, s.Upsert(&types.Registry{Id: 2, Name: "plain"}))

	// Quotes and FTS5 operators in the query are matched as characters.
	got, _, err := s.Search(contains, `"hi" OR`, 0, 50, 0)
	require.NoError(t, err)
	require.Equal(t, []string{`say "hi" OR NOT`}, names(t, got))

	got, _, err = s.Search(contains, "NOT plain", 0, 50, 0)
	require.NoError(t, err)
	require.Empty(t, got, `"NOT" must not act as an operator`)
}

func TestStore_ContainsFollowsRename(t *testing.T) {
	// Upserting an existing id takes the UPDATE path, whose trigger must drop
	// the old name's trigrams and index the new one.
	s := newStore(t)
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: "alpha-one"}))
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: "beta-two"}))

	got, _, err := s.Search(contains, "alpha", 0, 50, 0)
	require.NoError(t, err)
	require.Empty(t, got)
	got, _, err = s.Search(contains, "beta", 0, 50, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"beta-two"}, names(t, got))
}

func TestStore_UnicodeSuffix(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: "café-Registry"}))

	got, _, err := s.Search(suffix, "registry", 0, 50, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"café-Registry"}, names(t, got))
}

func TestStore_Pagination(t *testing.T) {
	s := newStore(t)
	for i := uint64(1); i <= 5; i++ {
		require.NoError(t, s.Upsert(&types.Registry{Id: i, Name: "reg"}))
	}

	page1, _, err := s.Search(exact, "reg", 0, 2, 0)
	require.NoError(t, err)
	page2, _, err := s.Search(exact, "reg", 0, 2, 2)
	require.NoError(t, err)
	page3, _, err := s.Search(exact, "reg", 0, 2, 4)
	require.NoError(t, err)

	require.Len(t, page1, 2)
	require.Len(t, page2, 2)
	require.Len(t, page3, 1)
	require.Equal(t, uint64(1), page1[0].Id)
	require.Equal(t, uint64(3), page2[0].Id)
	require.Equal(t, uint64(5), page3[0].Id)
}

// TestStore_StartPaging walks every mode page by page with the next id each
// page returns, and checks the walk sees each match exactly once.
func TestStore_StartPaging(t *testing.T) {
	s := newStore(t)
	var want []uint64
	for i := uint64(1); i <= 7; i++ {
		name := fmt.Sprintf("kyc_%d_fund", i)
		if i%3 == 0 {
			name = "other"
		} else {
			want = append(want, i)
		}
		require.NoError(t, s.Upsert(&types.Registry{Id: i, Name: name}))
	}

	testCases := []struct {
		mode  types.RegistryNameMatchMode
		query string
	}{
		{prefix, "kyc"},
		{suffix, "fund"},
		{contains, "_fu"},
	}
	for _, tc := range testCases {
		var got []uint64
		var start uint64
		pages := 0
		for {
			page, next, err := s.Search(tc.mode, tc.query, start, 2, 0)
			require.NoError(t, err)
			for _, r := range page {
				got = append(got, r.Id)
			}
			pages++
			if next == 0 {
				break
			}
			require.Len(t, page, 2, "mode %d: a page with a next id is full", tc.mode)
			start = next
		}
		require.Equal(t, want, got, "mode %d", tc.mode)
		require.Equal(t, 3, pages, "mode %d", tc.mode)
	}

	t.Run("exact page that ends on the last match has no next", func(t *testing.T) {
		for i := uint64(10); i <= 11; i++ {
			require.NoError(t, s.Upsert(&types.Registry{Id: i, Name: "dup"}))
		}
		page, next, err := s.Search(exact, "dup", 0, 2, 0)
		require.NoError(t, err)
		require.Len(t, page, 2)
		require.Zero(t, next)

		page, next, err = s.Search(exact, "dup", 0, 1, 0)
		require.NoError(t, err)
		require.Len(t, page, 1)
		require.Equal(t, uint64(11), next)
	})

	t.Run("start past every match is empty", func(t *testing.T) {
		page, next, err := s.Search(prefix, "kyc", 100, 2, 0)
		require.NoError(t, err)
		require.Empty(t, page)
		require.Zero(t, next)
	})
}

func TestStore_BatchIsAtomic(t *testing.T) {
	s := newStore(t)

	// A batch closed without Commit leaves nothing behind.
	batch, err := s.Begin()
	require.NoError(t, err)
	require.NoError(t, batch.Upsert(&types.Registry{Id: 1, Name: "dropped"}))
	batch.Close()
	got, _, err := s.Search(exact, "dropped", 0, 50, 0)
	require.NoError(t, err)
	require.Empty(t, got)

	// A committed batch publishes every row at once; Close after Commit is a
	// no-op.
	batch, err = s.Begin()
	require.NoError(t, err)
	require.NoError(t, batch.Upsert(&types.Registry{Id: 1, Name: "kept"}))
	require.NoError(t, batch.Upsert(&types.Registry{Id: 2, Name: "kept"}))
	require.NoError(t, batch.Commit())
	batch.Close()
	got, _, err = s.Search(exact, "kept", 0, 50, 0)
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestStore_OffsetPastInt64IsEmpty(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: "reg"}))

	// database/sql refuses uint64 values with the high bit set; the store
	// must clamp rather than surface a driver error for a valid request.
	got, _, err := s.Search(exact, "reg", 0, math.MaxUint64, math.MaxUint64)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestStore_UpsertIsIdempotent(t *testing.T) {
	s := newStore(t)
	reg := &types.Registry{Id: 1, Name: "original", Description: "v1"}
	require.NoError(t, s.Upsert(reg))

	reg.Description = "v2"
	require.NoError(t, s.Upsert(reg))

	got, _, err := s.Search(exact, "original", 0, 50, 0)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "v2", got[0].Description)
}
