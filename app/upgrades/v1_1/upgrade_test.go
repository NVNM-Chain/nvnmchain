package v1_1_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/stretchr/testify/require"

	"github.com/NVNM-Chain/nvnmchain/app/upgrades"
	v1_1 "github.com/NVNM-Chain/nvnmchain/app/upgrades/v1_1"
	testkeeper "github.com/NVNM-Chain/nvnmchain/testutil/keeper"
	anchoringtypes "github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
)

func setup(t *testing.T) (sdk.Context, *upgrades.UpgradeKeepers) {
	t.Helper()
	k, ctx, _ := testkeeper.AnchoringKeeper(t)
	return ctx, &upgrades.UpgradeKeepers{AnchoringKeeper: k}
}

func TestResolveAnchoringFee(t *testing.T) {
	// evm denom set via package init in the testkeeper / evmtypes mock; use
	// whatever GetEVMCoinDenom returns at runtime as the substitution value.
	evmDenom := evmtypes.GetEVMCoinDenom()
	require.NotEmpty(t, evmDenom, "evm denom must be registered for substitution to work")

	defaultAmt := anchoringtypes.DefaultAnchoringFeeAmount

	tests := []struct {
		name       string
		stored     sdk.Coin
		wantDenom  string
		wantAmount math.Int
	}{
		{
			name:       "empty denom filled from evm denom, amount preserved",
			stored:     sdk.Coin{Denom: "", Amount: math.NewIntWithDecimal(1, 16)},
			wantDenom:  evmDenom,
			wantAmount: math.NewIntWithDecimal(1, 16),
		},
		{
			name:       "explicit denom and amount both preserved",
			stored:     sdk.NewCoin("ibc/ABC", math.NewIntWithDecimal(1, 16)),
			wantDenom:  "ibc/ABC",
			wantAmount: math.NewIntWithDecimal(1, 16),
		},
		{
			name:       "nil amount healed to default (pre-feature upgrade scenario)",
			stored:     sdk.Coin{Denom: "", Amount: math.Int{}},
			wantDenom:  evmDenom,
			wantAmount: defaultAmt,
		},
		{
			name:       "zero amount healed to default",
			stored:     sdk.Coin{Denom: "ibc/ABC", Amount: math.ZeroInt()},
			wantDenom:  "ibc/ABC",
			wantAmount: defaultAmt,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, keepers := setup(t)
			params, err := keepers.AnchoringKeeper.Params.Get(ctx)
			require.NoError(t, err)
			params.AnchoringFee = tc.stored
			require.NoError(t, keepers.AnchoringKeeper.Params.Set(ctx, params))

			require.NoError(t, v1_1.ResolveAnchoringFee(ctx, keepers))

			got, err := keepers.AnchoringKeeper.Params.Get(ctx)
			require.NoError(t, err)
			require.Equal(t, tc.wantDenom, got.AnchoringFee.Denom)
			require.False(t, got.AnchoringFee.Amount.IsNil(), "amount must never remain nil after upgrade")
			require.True(t, tc.wantAmount.Equal(got.AnchoringFee.Amount),
				"expected amount %s, got %s", tc.wantAmount, got.AnchoringFee.Amount)
		})
	}
}
