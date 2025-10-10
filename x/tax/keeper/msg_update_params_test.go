package keeper_test

import (
	"testing"

	appparams "github.com/MANTRA-Chain/inveniam/app/params"
	keepertest "github.com/MANTRA-Chain/inveniam/testutil/keeper"
	"github.com/MANTRA-Chain/inveniam/x/tax/keeper"
	"github.com/MANTRA-Chain/inveniam/x/tax/types"
	"github.com/stretchr/testify/require"
)

func TestMsgUpdateParams(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.TaxKeeper(t)
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
				Authority:  "invalid",
				McaTax:     "",
				McaAddress: "",
			},
			expErr:    true,
			expErrMsg: "invalid authority",
		},
		{
			name: "update mca tax",
			input: &types.MsgUpdateParams{
				Authority:  "inveniam15m77x4pe6w9vtpuqm22qxu0ds7vn4ehz80mwh8",
				McaTax:     "0.200000000000000000",
				McaAddress: "",
			},
			expErr: false,
		},
		{
			name: "update mca address",
			input: &types.MsgUpdateParams{
				Authority:  "inveniam15m77x4pe6w9vtpuqm22qxu0ds7vn4ehz80mwh8",
				McaTax:     "",
				McaAddress: "inveniam1axznhnm82lah8qqvp9hxdad49yx3s5dcmnx072",
			},
			expErr: false,
		},
		{
			name: "old authority address no longer work",
			input: &types.MsgUpdateParams{
				Authority:  "inveniam15m77x4pe6w9vtpuqm22qxu0ds7vn4ehz80mwh8",
				McaTax:     "",
				McaAddress: "",
			},
			expErr:    true,
			expErrMsg: "invalid sender; expected mcaAddress",
		},
		{
			name: "update both",
			input: &types.MsgUpdateParams{
				Authority:  "inveniam1axznhnm82lah8qqvp9hxdad49yx3s5dcmnx072",
				McaTax:     "0.200000000000000000",
				McaAddress: "inveniam15m77x4pe6w9vtpuqm22qxu0ds7vn4ehz80mwh8",
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
				if tc.input.McaTax != "" {
					require.Equal(t, tc.input.McaTax, params.McaTax.String())
				}
				if tc.input.McaAddress != "" {
					require.Equal(t, tc.input.McaAddress, params.McaAddress)
				}
			}
		})
	}
}
