package v2

import (
	"testing"

	"cosmossdk.io/collections"
	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// newTestStore builds a bare KVStoreService + codec + context, independent of the
// anchoring keeper, so this migration package stays decoupled from keeper internals.
func newTestStore(t *testing.T) (corestore.KVStoreService, codec.BinaryCodec, sdk.Context) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	return runtime.NewKVStoreService(storeKey), cdc, ctx
}

// legacyRecord builds a record shaped like one written before Record carried a
// registry_id field: RegistryId is left at its zero value, to be backfilled by
// MigrateStore purely from the record's own store key.
func legacyRecord(uri, checksum string) types.Record {
	return types.Record{
		Uri:          uri,
		Checksum:     checksum,
		ChecksumAlgo: "sha256",
		Metadata:     "{}",
		Status:       "active",
		RecordId:     1,
		Index:        1,
		IsLatest:     true,
	}
}

func TestMigrateStore(t *testing.T) {
	storeService, cdc, ctx := newTestStore(t)

	records, err := recordsCollection(storeService, cdc)
	require.NoError(t, err)

	legacyKeyA := collections.Join3(uint64(1), uint64(1), uint64(1))
	require.NoError(t, records.Set(ctx, legacyKeyA, legacyRecord("ipfs://a", "checksum-a")))

	legacyKeyB := collections.Join3(uint64(2), uint64(1), uint64(1))
	require.NoError(t, records.Set(ctx, legacyKeyB, legacyRecord("ipfs://b", "checksum-b")))

	err = MigrateStore(ctx, storeService, cdc)
	require.NoError(t, err)

	gotA, err := records.Get(ctx, legacyKeyA)
	require.NoError(t, err)
	require.Equal(t, uint64(1), gotA.RegistryId)

	gotB, err := records.Get(ctx, legacyKeyB)
	require.NoError(t, err)
	require.Equal(t, uint64(2), gotB.RegistryId)
}

func TestMigrateStore_Empty(t *testing.T) {
	storeService, cdc, ctx := newTestStore(t)

	err := MigrateStore(ctx, storeService, cdc)
	require.NoError(t, err)
}

// TestMigrateStore_ClearsRetiredRegistryIdByNameIndex guards the other half of the
// v1->v2 migration: the retired registry_id_by_name index (no longer maintained by the
// keeper) must be fully purged from existing chain state, not just left orphaned.
func TestMigrateStore_ClearsRetiredRegistryIdByNameIndex(t *testing.T) {
	storeService, cdc, ctx := newTestStore(t)

	registryIdByName, err := registryIdByNameCollection(storeService)
	require.NoError(t, err)
	require.NoError(t, registryIdByName.Set(ctx, "reg-a", 1))
	require.NoError(t, registryIdByName.Set(ctx, "reg-b", 2))

	err = MigrateStore(ctx, storeService, cdc)
	require.NoError(t, err)

	hasA, err := registryIdByName.Has(ctx, "reg-a")
	require.NoError(t, err)
	require.False(t, hasA)

	hasB, err := registryIdByName.Has(ctx, "reg-b")
	require.NoError(t, err)
	require.False(t, hasB)
}
