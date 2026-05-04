package keeper

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/core/address"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	_ "github.com/NVNM-Chain/nvnmchain/app"
	"github.com/NVNM-Chain/nvnmchain/x/tax/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/tax/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
)

// MockAuthKeeper is a minimal auth keeper for tests.
type MockAuthKeeper struct {
	ModuleAccounts map[string]sdk.ModuleAccountI
}

func (m *MockAuthKeeper) GetModuleAccount(_ context.Context, name string) sdk.ModuleAccountI {
	if m.ModuleAccounts != nil {
		return m.ModuleAccounts[name]
	}
	return nil
}

// MockBankKeeper is a minimal bank keeper for tests that need BlockedAddr behaviour.
type MockBankKeeper struct {
	BlockedAddrs map[string]bool
	// Balances holds per-address balances returned by GetAllBalances.
	Balances map[string]sdk.Coins
}

func (m *MockBankKeeper) BlockedAddr(addr sdk.AccAddress) bool {
	return m.BlockedAddrs[addr.String()]
}

func (m *MockBankKeeper) GetAllBalances(_ context.Context, addr sdk.AccAddress) sdk.Coins {
	return m.Balances[addr.String()]
}

func (m *MockBankKeeper) GetBalance(_ context.Context, _ sdk.AccAddress, _ string) sdk.Coin {
	return sdk.Coin{}
}

func (m *MockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, _ string, recipientAddr sdk.AccAddress, _ sdk.Coins) error {
	if m.BlockedAddrs[recipientAddr.String()] {
		return fmt.Errorf("%s is not allowed to receive funds", recipientAddr)
	}
	return nil
}

func (m *MockBankKeeper) BurnCoins(_ context.Context, _ string, _ sdk.Coins) error { return nil }

func TaxKeeper(tb testing.TB) (keeper.Keeper, sdk.Context, address.Codec) {
	tb.Helper()
	return TaxKeeperWithBankKeeper(tb, &MockBankKeeper{})
}

func TaxKeeperWithBankKeeper(tb testing.TB, bk types.BankKeeper) (keeper.Keeper, sdk.Context, address.Codec) {
	tb.Helper()
	return TaxKeeperWithAuthAndBankKeeper(tb, nil, bk)
}

func TaxKeeperWithAuthAndBankKeeper(tb testing.TB, ak types.AccountKeeper, bk types.BankKeeper) (keeper.Keeper, sdk.Context, address.Codec) {
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
		log.NewNopLogger(),
		ak,
		bk,
		authtypes.FeeCollectorName,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	if err := k.Params.Set(ctx, types.DefaultParams()); err != nil {
		tb.Fatalf("failed to set params: %v", err)
	}

	return k, ctx, addressCodec
}
