package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/NVNM-Chain/nvnmchain/app"
	appparams "github.com/NVNM-Chain/nvnmchain/app/params"
	keepertest "github.com/NVNM-Chain/nvnmchain/testutil/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestMsgUpdateParams(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Reflect post-upgrade state: denom resolved by v1.1.0 / InitGenesis.
	params := types.DefaultParams()
	params.AnchoringFee.Denom = app.FutureStakingDenom
	require.NoError(t, k.Params.Set(ctx, params))

	// default params
	testCases := []struct {
		name      string
		input     *types.MsgUpdateParams
		expErr    bool
		expErrMsg string
	}{
		{
			name: "invalid authority",
			input: &types.MsgUpdateParams{
				Authority: "invalid",
				Admin:     "",
			},
			expErr:    true,
			expErrMsg: "invalid authority",
		},
		{
			name: "update tax address",
			input: &types.MsgUpdateParams{
				Authority: "nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca",
				Admin:     keepertest.TestSenderAddr,
			},
			expErr: false,
		},
		{
			name: "old authority address no longer work",
			input: &types.MsgUpdateParams{
				Authority: "nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca",
				Admin:     "",
			},
			expErr:    true,
			expErrMsg: "invalid sender; expected admin",
		},
		{
			name: "empty denom rejected at runtime",
			input: &types.MsgUpdateParams{
				Authority:    keepertest.TestSenderAddr,
				AnchoringFee: &sdk.Coin{Denom: "", Amount: math.NewIntWithDecimal(1, 16)},
			},
			expErr:    true,
			expErrMsg: "anchoring fee denom cannot be empty",
		},
		{
			name: "non-empty denom accepted",
			input: &types.MsgUpdateParams{
				Authority:    keepertest.TestSenderAddr,
				AnchoringFee: &sdk.Coin{Denom: "updated", Amount: math.NewIntWithDecimal(2, 16)},
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.UpdateParams(ctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				params, err := k.Params.Get(ctx)
				require.NoError(t, err)
				if tc.input.Admin != "" {
					require.Equal(t, tc.input.Admin, params.Admin)
				}
			}
		})
	}
}

// Admin-only update on the InitGenesis/upgrade sentinel auto-heals the denom
// from the EVM coin instead of failing validation.
func TestMsgUpdateParams_AdminOnlyAutoHealsWhenStoredDenomEmpty(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	params := types.DefaultParams() // AnchoringFee.Denom is "" by default
	require.NoError(t, k.Params.Set(ctx, params))

	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: types.DefaultAdminAddress,
		Admin:     keepertest.TestSenderAddr,
	})
	require.NoError(t, err)
	got, err := k.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, keepertest.TestSenderAddr, got.Admin)
	require.Equal(t, app.FutureStakingDenom, got.AnchoringFee.Denom)
}

// Admin-only update must not clobber a denom already set by the operator.
func TestMsgUpdateParams_AdminOnlyPreservesExistingDenom(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	const operatorDenom = "ibc/CAFEBABE"
	params := types.DefaultParams()
	params.AnchoringFee = sdk.NewCoin(operatorDenom, math.NewIntWithDecimal(1, 16))
	require.NoError(t, k.Params.Set(ctx, params))

	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: types.DefaultAdminAddress,
		Admin:     keepertest.TestSenderAddr,
	})
	require.NoError(t, err)
	got, err := k.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, operatorDenom, got.AnchoringFee.Denom, "operator-set denom must survive an admin-only update")
}
