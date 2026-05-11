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
)

func setup(t *testing.T) (sdk.Context, *upgrades.UpgradeKeepers) {
	t.Helper()
	k, ctx, _ := testkeeper.AnchoringKeeper(t)
	return ctx, &upgrades.UpgradeKeepers{AnchoringKeeper: k}
}

func TestResolveAnchoringFeeDenom(t *testing.T) {
	// evm denom set via package init in the testkeeper / evmtypes mock; use
	// whatever GetEVMCoinDenom returns at runtime as the substitution value.
	evmDenom := evmtypes.GetEVMCoinDenom()
	require.NotEmpty(t, evmDenom, "evm denom must be registered for substitution to work")

	tests := []struct {
		name      string
		stored    sdk.Coin
		wantDenom string
	}{
		{
			name:      "empty denom is filled from evm denom",
			stored:    sdk.Coin{Denom: "", Amount: math.NewIntWithDecimal(1, 16)},
			wantDenom: evmDenom,
		},
		{
			name:      "explicit denom is preserved",
			stored:    sdk.NewCoin("ibc/ABC", math.NewIntWithDecimal(1, 16)),
			wantDenom: "ibc/ABC",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, keepers := setup(t)
			params, err := keepers.AnchoringKeeper.Params.Get(ctx)
			require.NoError(t, err)
			params.AnchoringFee = tc.stored
			require.NoError(t, keepers.AnchoringKeeper.Params.Set(ctx, params))

			require.NoError(t, v1_1.ResolveAnchoringFeeDenom(ctx, keepers))

			got, err := keepers.AnchoringKeeper.Params.Get(ctx)
			require.NoError(t, err)
			require.Equal(t, tc.wantDenom, got.AnchoringFee.Denom)
			require.Equal(t, tc.stored.Amount, got.AnchoringFee.Amount, "amount must be preserved")
		})
	}
}
