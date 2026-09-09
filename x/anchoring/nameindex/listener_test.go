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

// TestListener_ListenCommit_BlockIsAtomic checks that a block is indexed all
// or nothing: a write that cannot be decoded must not leave the block's
// earlier registries behind, since the backfill on restart is what repairs a
// failed block and it should start from a clean slate for it.
func TestListener_ListenCommit_BlockIsAtomic(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	s := newStore(t)
	l := nameindex.NewListener(s)

	registryKey := func(id byte) []byte {
		return append(append([]byte{}, types.RegistriesKeyPrefix...), 0, 0, 0, 0, 0, 0, 0, id)
	}
	changeSet := []*storetypes.StoreKVPair{
		{
			StoreKey: types.StoreKey,
			Key:      registryKey(1),
			Value:    mustMarshal(t, cdc, &types.Registry{Id: 1, Name: "first"}),
		},
		{
			StoreKey: types.StoreKey,
			Key:      registryKey(2),
			Value:    []byte{0xff, 0xff},
		},
	}

	require.Error(t, l.ListenCommit(context.Background(), abci.ResponseCommit{}, changeSet))

	got, err := s.Search(nameindex.MatchModeExact, "first", 50, 0)
	require.NoError(t, err)
	require.Empty(t, got, "a block that fails to decode must not be half-indexed")
}

// TestListener_ListenCommit_NoRegistryWrites covers the common case: almost
// every block touches no registry at all, and those must not open a SQLite
// transaction. An empty or wholly-irrelevant changeset returns before Begin.
func TestListener_ListenCommit_NoRegistryWrites(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	s := newStore(t)
	l := nameindex.NewListener(s)

	// Seed one registry so an accidental write would be visible as a change.
	registryKey := append(append([]byte{}, types.RegistriesKeyPrefix...), 0, 0, 0, 0, 0, 0, 0, 1)
	require.NoError(t, l.ListenCommit(context.Background(), abci.ResponseCommit{},
		[]*storetypes.StoreKVPair{{
			StoreKey: types.StoreKey,
			Key:      registryKey,
			Value:    mustMarshal(t, cdc, &types.Registry{Id: 1, Name: "seeded"}),
		}},
	))

	otherKey := append(append([]byte{}, types.RecordsKeyPrefix...), 1)
	for _, changeSet := range [][]*storetypes.StoreKVPair{
		nil,
		{},
		{{StoreKey: types.StoreKey, Key: otherKey, Value: []byte("record")}},
		{{StoreKey: "other-module", Key: registryKey, Value: []byte("whatever")}},
	} {
		require.NoError(t, l.ListenCommit(context.Background(), abci.ResponseCommit{}, changeSet))
	}

	got, err := s.Search(nameindex.MatchModeExact, "seeded", 50, 0)
	require.NoError(t, err)
	require.Len(t, got, 1, "blocks without registry writes must not disturb the index")
}
