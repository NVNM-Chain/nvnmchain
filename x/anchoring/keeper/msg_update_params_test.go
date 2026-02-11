package keeper_test

import (
	"testing"

	appparams "github.com/MANTRA-Chain/inveniam/app/params"
	keepertest "github.com/MANTRA-Chain/inveniam/testutil/keeper"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/keeper"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/types"
	"github.com/stretchr/testify/require"
)

func TestMsgUpdateParams(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	params := types.DefaultParams()
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
			name: "update mca address",
			input: &types.MsgUpdateParams{
				Authority: "inveniam15m77x4pe6w9vtpuqm22qxu0ds7vn4ehz80mwh8",
				Admin:     keepertest.TestSenderAddr,
			},
			expErr: false,
		},
		{
			name: "old authority address no longer work",
			input: &types.MsgUpdateParams{
				Authority: "inveniam15m77x4pe6w9vtpuqm22qxu0ds7vn4ehz80mwh8",
				Admin:     "",
			},
			expErr:    true,
			expErrMsg: "invalid sender; expected admin",
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
