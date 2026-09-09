package nameindex_test

import (
	"context"
	"testing"

	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/nameindex"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
)

func mustMarshal(t *testing.T, cdc codec.BinaryCodec, reg *types.Registry) []byte {
	t.Helper()
	bz, err := cdc.Marshal(reg)
	require.NoError(t, err)
	return bz
}

func TestListener_ListenCommit(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	s := newStore(t)
	l := nameindex.NewListener(s)

	registryKey := append(append([]byte{}, types.RegistriesKeyPrefix...), 0, 0, 0, 0, 0, 0, 0, 1)
	otherModuleKey := append(append([]byte{}, types.RegistriesKeyPrefix...), 0, 0, 0, 0, 0, 0, 0, 2)
	nonRegistryKey := append(append([]byte{}, types.RecordsKeyPrefix...), 1)
	deletedKey := append(append([]byte{}, types.RegistriesKeyPrefix...), 0, 0, 0, 0, 0, 0, 0, 3)

	changeSet := []*storetypes.StoreKVPair{
		{
			StoreKey: types.StoreKey,
			Key:      registryKey,
			Value:    mustMarshal(t, cdc, &types.Registry{Id: 1, Name: "indexed_registry"}),
		},
		{
			// Wrong module's store: must be ignored even though the key bytes
			// happen to match the Registries prefix.
			StoreKey: "some-other-module",
			Key:      otherModuleKey,
			Value:    mustMarshal(t, cdc, &types.Registry{Id: 2, Name: "should_not_index"}),
		},
		{
			// Right module, wrong collection prefix: must be ignored.
			StoreKey: types.StoreKey,
			Key:      nonRegistryKey,
			Value:    []byte("not-a-registry"),
		},
		{
			// Delete entries must be ignored; registries are add-only.
			StoreKey: types.StoreKey,
			Key:      deletedKey,
			Value:    mustMarshal(t, cdc, &types.Registry{Id: 3, Name: "deleted_registry"}),
			Delete:   true,
		},
	}

	require.NoError(t, l.ListenCommit(context.Background(), abci.ResponseCommit{}, changeSet))

	got, err := s.Search(nameindex.MatchModeExact, "indexed_registry", 50, 0)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, uint64(1), got[0].Id)

	for _, name := range []string{"should_not_index", "deleted_registry"} {
		got, err := s.Search(nameindex.MatchModeExact, name, 50, 0)
		require.NoError(t, err)
		require.Emptyf(t, got, "expected %q to not be indexed", name)
	}

	require.NoError(t, l.ListenFinalizeBlock(context.Background(), abci.RequestFinalizeBlock{}, abci.ResponseFinalizeBlock{}))
}
