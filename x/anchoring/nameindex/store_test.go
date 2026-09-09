package nameindex_test

import (
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
