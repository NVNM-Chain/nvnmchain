package tax_test

import (
	"testing"

	"cosmossdk.io/math"
	appparams "github.com/NVNM-Chain/nvnmchain/app/params"
	keepertest "github.com/NVNM-Chain/nvnmchain/testutil/keeper"
	tax "github.com/NVNM-Chain/nvnmchain/x/tax/module"
	"github.com/NVNM-Chain/nvnmchain/x/tax/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
)

func TestEndBlocker_InvalidTaxAddress(t *testing.T) {
	appparams.SetAddressPrefixes()

	// feeCollectorAccount is used by AllocateTax to look up the fee collector module account.
	blockedAddr := authtypes.NewModuleAddress("distribution").String()
	feeCollectorAccount := authtypes.NewEmptyModuleAccount(authtypes.FeeCollectorName)
	feeCollectorCoins := sdk.NewCoins(sdk.NewInt64Coin("uom", 1_000_000))
	ak := &keepertest.MockAuthKeeper{
		ModuleAccounts: map[string]sdk.ModuleAccountI{
			authtypes.FeeCollectorName: feeCollectorAccount,
		},
	}
	bk := &keepertest.MockBankKeeper{
		BlockedAddrs: map[string]bool{blockedAddr: true},
		Balances:     map[string]sdk.Coins{feeCollectorAccount.GetAddress().String(): feeCollectorCoins},
	}

	tests := []struct {
		name        string
		taxAddress  string
		tax         string
		blockHeight int64
		expErr      bool
	}{
		{
			name:        "invalid bech32 tax address returns error",
			taxAddress:  "not_a_valid_address",
			tax:         "0.1",
			blockHeight: 2,
			expErr:      true,
		},
		{
			name:        "empty tax address returns error",
			taxAddress:  "",
			tax:         "0.1",
			blockHeight: 2,
			expErr:      true,
		},
		{
			name:        "invalid address skipped when tax is zero",
			taxAddress:  "not_a_valid_address",
			tax:         "0",
			blockHeight: 2,
			expErr:      false,
		},
		{
			name:        "invalid address skipped at block height 1",
			taxAddress:  "not_a_valid_address",
			tax:         "0.1",
			blockHeight: 1,
			expErr:      false,
		},
		{
			name:        "blocked tax address returns error",
			taxAddress:  blockedAddr,
			tax:         "0.1",
			blockHeight: 2,
			expErr:      true,
		},
		{
			name:        "valid address succeeds",
			taxAddress:  keepertest.TestSenderAddr,
			tax:         "0.1",
			blockHeight: 2,
			expErr:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			k, ctx, _ := keepertest.TaxKeeperWithAuthAndBankKeeper(t, ak, bk)
			ctx = ctx.WithBlockHeader(cmtproto.Header{Height: tc.blockHeight})

			params := types.Params{
				TaxAddress: tc.taxAddress,
				Tax:        math.LegacyMustNewDecFromStr(tc.tax),
			}
			require.NoError(t, k.Params.Set(ctx, params))

			err := tax.EndBlocker(ctx, k)
			if tc.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEndBlocker_DustBalanceNoEvent(t *testing.T) {
	appparams.SetAddressPrefixes()
	feeCollector := authtypes.NewEmptyModuleAccount(authtypes.FeeCollectorName)
	ak := &keepertest.MockAuthKeeper{
		ModuleAccounts: map[string]sdk.ModuleAccountI{authtypes.FeeCollectorName: feeCollector},
	}
	bk := &keepertest.MockBankKeeper{
		Balances: map[string]sdk.Coins{
			feeCollector.GetAddress().String(): sdk.NewCoins(sdk.NewInt64Coin("uom", 1)),
		},
	}
	k, ctx, _ := keepertest.TaxKeeperWithAuthAndBankKeeper(t, ak, bk)
	ctx = ctx.WithBlockHeight(2)

	require.NoError(t, k.Params.Set(ctx, types.Params{
		TaxAddress: keepertest.TestSenderAddr,
		Tax:        math.LegacyMustNewDecFromStr("0.1"),
	}))

	require.NoError(t, tax.EndBlocker(ctx, k))

	for _, ev := range ctx.EventManager().Events() {
		require.NotEqual(t, types.EventTypeTaxAmount, ev.Type)
	}
}
