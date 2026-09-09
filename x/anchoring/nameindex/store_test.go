package nameindex_test

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/nameindex"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"
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
		got, err := s.Search(nameindex.MatchModeExact, "kyc_registry", 50, 0)
		require.NoError(t, err)
		require.Equal(t, []string{"kyc_registry"}, names(t, got))
	})

	t.Run("prefix", func(t *testing.T) {
		got, err := s.Search(nameindex.MatchModePrefix, "kyc", 50, 0)
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"kyc_registry", "KYC_Extended"}, names(t, got))
	})

	t.Run("suffix", func(t *testing.T) {
		got, err := s.Search(nameindex.MatchModeSuffix, "registry", 50, 0)
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"kyc_registry", "aml_registry"}, names(t, got))
	})

	t.Run("contains", func(t *testing.T) {
		got, err := s.Search(nameindex.MatchModeContains, "_reg", 50, 0)
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"kyc_registry", "aml_registry"}, names(t, got))
	})

	t.Run("no match", func(t *testing.T) {
		got, err := s.Search(nameindex.MatchModeContains, "zzz", 50, 0)
		require.NoError(t, err)
		require.Empty(t, got)
	})
}

func TestStore_LikeMetacharactersAreLiteral(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: "50%_off"}))
	require.NoError(t, s.Upsert(&types.Registry{Id: 2, Name: "50X_off"}))

	// A literal "%" or "_" in the query must not act as a SQL LIKE wildcard.
	got, err := s.Search(nameindex.MatchModeContains, "%_off", 50, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"50%_off"}, names(t, got))
}

func TestStore_ContainsNeedsThreeCharacters(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: "kyc_registry"}))
	require.NoError(t, s.Upsert(&types.Registry{Id: 2, Name: "café"}))

	_, err := s.Search(nameindex.MatchModeContains, "ky", 50, 0)
	require.ErrorIs(t, err, nameindex.ErrContainsTooShort)

	// The floor is counted in characters, not bytes: "afé" is 3 runes.
	got, err := s.Search(nameindex.MatchModeContains, "afé", 50, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"café"}, names(t, got))

	// The other modes have no minimum.
	got, err = s.Search(nameindex.MatchModePrefix, "k", 50, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"kyc_registry"}, names(t, got))
}

func TestStore_ContainsTreatsFTSSyntaxAsText(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: `say "hi" OR NOT`}))
	require.NoError(t, s.Upsert(&types.Registry{Id: 2, Name: "plain"}))

	// Quotes and FTS5 operators in the query are matched as characters.
	got, err := s.Search(nameindex.MatchModeContains, `"hi" OR`, 50, 0)
	require.NoError(t, err)
	require.Equal(t, []string{`say "hi" OR NOT`}, names(t, got))

	got, err = s.Search(nameindex.MatchModeContains, "NOT plain", 50, 0)
	require.NoError(t, err)
	require.Empty(t, got, `"NOT" must not act as an operator`)
}

func TestStore_ContainsFollowsRename(t *testing.T) {
	// Upserting an existing id takes the UPDATE path, whose trigger must drop
	// the old name's trigrams and index the new one.
	s := newStore(t)
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: "alpha-one"}))
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: "beta-two"}))

	got, err := s.Search(nameindex.MatchModeContains, "alpha", 50, 0)
	require.NoError(t, err)
	require.Empty(t, got)
	got, err = s.Search(nameindex.MatchModeContains, "beta", 50, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"beta-two"}, names(t, got))
}

func TestStore_UnicodeSuffix(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: "café-Registry"}))

	got, err := s.Search(nameindex.MatchModeSuffix, "registry", 50, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"café-Registry"}, names(t, got))
}

func TestStore_Pagination(t *testing.T) {
	s := newStore(t)
	for i := uint64(1); i <= 5; i++ {
		require.NoError(t, s.Upsert(&types.Registry{Id: i, Name: "reg"}))
	}

	page1, err := s.Search(nameindex.MatchModeExact, "reg", 2, 0)
	require.NoError(t, err)
	page2, err := s.Search(nameindex.MatchModeExact, "reg", 2, 2)
	require.NoError(t, err)
	page3, err := s.Search(nameindex.MatchModeExact, "reg", 2, 4)
	require.NoError(t, err)

	require.Len(t, page1, 2)
	require.Len(t, page2, 2)
	require.Len(t, page3, 1)
	require.Equal(t, uint64(1), page1[0].Id)
	require.Equal(t, uint64(3), page2[0].Id)
	require.Equal(t, uint64(5), page3[0].Id)
}

func TestStore_BatchIsAtomic(t *testing.T) {
	s := newStore(t)

	// A batch closed without Commit leaves nothing behind.
	batch, err := s.Begin()
	require.NoError(t, err)
	require.NoError(t, batch.Upsert(&types.Registry{Id: 1, Name: "dropped"}))
	batch.Close()
	got, err := s.Search(nameindex.MatchModeExact, "dropped", 50, 0)
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
	got, err = s.Search(nameindex.MatchModeExact, "kept", 50, 0)
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestStore_OffsetPastInt64IsEmpty(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: "reg"}))

	// database/sql refuses uint64 values with the high bit set; the store
	// must clamp rather than surface a driver error for a valid request.
	got, err := s.Search(nameindex.MatchModeExact, "reg", math.MaxUint64, math.MaxUint64)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestStore_UpsertIsIdempotent(t *testing.T) {
	s := newStore(t)
	reg := &types.Registry{Id: 1, Name: "original", Description: "v1"}
	require.NoError(t, s.Upsert(reg))

	reg.Description = "v2"
	require.NoError(t, s.Upsert(reg))

	got, err := s.Search(nameindex.MatchModeExact, "original", 50, 0)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "v2", got[0].Description)
}
