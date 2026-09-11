package evmlayout_test

import (
	"fmt"
	"math/big"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	appparams "github.com/NVNM-Chain/nvnmchain/app/params"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/evmlayout"
	anchoringkeeper "github.com/NVNM-Chain/nvnmchain/x/anchoring/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdkruntime "github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// A checksum's registries come out ascending, however they were written, four ids to a word.
func TestTheChecksumIndexArrivesInOrder(t *testing.T) {
	k, ctx := anchoringKeeper(t)

	for _, name := range []string{"us-a", "us-b", "us-c", "us-d", "us-e"} {
		_, err := k.AddRegistry(ctx, alice.Bytes(), name, "d", `{"a":1}`)
		require.NoError(t, err)
	}
	for _, id := range []uint64{3, 1, 5, 2, 4} {
		_, err := k.AddRecord(ctx, alice.Bytes(), record(id, "shared", "https://ex.test/o"))
		require.NoError(t, err)
	}

	got := migrated(t, k, ctx)
	lengthSlot := evmlayout.MapString("shared", evmlayout.Word(evmlayout.SlotRegistriesByChecksum))
	require.Equal(t, evmlayout.Word(5), got[lengthSlot])
	// The first id in the low 8 bytes of the first word; the fifth starts the next.
	data := evmlayout.ArrayData(lengthSlot)
	want := common.HexToHash("0x0000000000000004" + "0000000000000003" + "0000000000000002" + "0000000000000001")
	require.Equal(t, want, got[data])
	require.Equal(t, evmlayout.Word(5), got[common.BigToHash(new(big.Int).Add(data.Big(), big.NewInt(1)))])
}

// The contract pages registries by arithmetic, so a gap has to be refused here.
func TestAGapInTheRegistryIdsIsRefused(t *testing.T) {
	k, ctx := anchoringKeeper(t)

	for i := 0; i < 3; i++ {
		_, err := k.AddRegistry(ctx, alice.Bytes(), "us-a", "d", `{"a":1}`)
		require.NoError(t, err)
	}
	require.NoError(t, k.Registries.Remove(ctx, 2))

	err := evmlayout.Migrate(ctx, k, func(evmlayout.Write) error { return nil })
	require.ErrorContains(t, err, "ids must run from 1 with no gaps, got 3 after 1")
}

// The header's count is where the contract's paging ends, so it has to be the last id.
func TestARegistryCountPastTheLastIdIsRefused(t *testing.T) {
	k, ctx := anchoringKeeper(t)

	_, err := k.AddRegistry(ctx, alice.Bytes(), "us-a", "d", `{"a":1}`)
	require.NoError(t, err)
	require.NoError(t, k.RegistryCount.Set(ctx, 2))

	err = evmlayout.Migrate(ctx, k, func(evmlayout.Write) error { return nil })
	require.ErrorContains(t, err, "RegistryCount is 2 but the ids end at 1")
}

// An empty module migrates to just the module admin; a zero registry count is not written.
func TestMigratingAnEmptyModule(t *testing.T) {
	k, ctx := anchoringKeeper(t)
	require.Equal(t, map[common.Hash]common.Hash{
		evmlayout.Word(evmlayout.SlotHeader): common.BytesToHash(moduleAdmin.Bytes()),
	}, migrated(t, k, ctx))
}

// migrated is every slot Migrate writes for k's state, refusing one written twice.
func migrated(t *testing.T, k anchoringkeeper.Keeper, ctx sdk.Context) map[common.Hash]common.Hash {
	t.Helper()
	out := map[common.Hash]common.Hash{}
	require.NoError(t, evmlayout.Migrate(ctx, k, func(w evmlayout.Write) error {
		if _, seen := out[w.Slot]; seen {
			return fmt.Errorf("slot %s written twice", w.Slot)
		}
		out[w.Slot] = w.Value
		return nil
	}))
	return out
}

// Built here rather than taken from testutil/keeper, which would bring the app package and its
// uint256 build tag.
func anchoringKeeper(t *testing.T) (anchoringkeeper.Keeper, sdk.Context) {
	t.Helper()
	appparams.SetAddressPrefixes()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	// Plain keys, no IAVL: nothing here asks for a proof or a root, and the tree is what made
	// the mainnet seed expensive. Dropping it costs nothing and saves both time and memory --
	// the full seed of 11.9M records goes from ~72 GB and ~12 minutes to 26 GB and 4 minutes.
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeDB, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	k := anchoringkeeper.NewKeeper(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		sdkruntime.NewKVStoreService(storeKey),
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger()).
		WithBlockTime(blockTime)
	// The fixture contract's admin and clock.
	require.NoError(t, k.Params.Set(ctx, types.NewParams(sdk.AccAddress(moduleAdmin.Bytes()).String())))
	require.NoError(t, k.RegistryCount.Set(ctx, 0))
	return k, ctx
}
