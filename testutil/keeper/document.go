package keeper

import (
	"testing"

	"cosmossdk.io/core/address"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	_ "github.com/NVNM-Chain/nvnmchain/app"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

const TestSenderAddr = "nvnm1axznhnm82lah8qqvp9hxdad49yx3s5dc6h6p3s"

func AnchoringKeeper(tb testing.TB) (keeper.Keeper, sdk.Context, address.Codec) {
	tb.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(tb, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())

	k := keeper.NewKeeper(
		cdc,
		addressCodec,
		runtime.NewKVStoreService(storeKey),
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	if err := k.Params.Set(ctx, types.DefaultParams()); err != nil {
		tb.Fatalf("failed to set params: %v", err)
	}
	if err := k.RegistryCount.Set(ctx, 0); err != nil {
		tb.Fatalf("failed to set registry count: %v", err)
	}

	return k, ctx, addressCodec
}

func MustCreateAnchoringRegistry(tb testing.TB, k keeper.Keeper, ctx sdk.Context, sender, name string) uint64 {
	tb.Helper()
	ms := keeper.NewMsgServerImpl(k)
	res, err := ms.AddRegistry(ctx, &types.MsgAddRegistry{
		Sender:      sender,
		Name:        name,
		Description: "test",
		Metadata:    "{\"k\":\"v\"}",
	})
	require.NoError(tb, err)
	return res.RegistryId
}

func MustAddAnchoringRecord(tb testing.TB, k keeper.Keeper, ctx sdk.Context, sender string, registryId uint64, checksum, checksumAlgo string) uint64 {
	tb.Helper()
	ms := keeper.NewMsgServerImpl(k)
	res, err := ms.AddRecord(ctx, &types.MsgAddRecord{
		Sender: sender,
		Record: &types.Record{
			RegistryId:   registryId,
			Uri:          "ipfs://bafy...",
			Checksum:     checksum,
			ChecksumAlgo: checksumAlgo,
			Metadata:     "{\"k\":\"v\"}",
			Status:       "active",
		},
	})
	require.NoError(tb, err)
	return res.RecordId
}
