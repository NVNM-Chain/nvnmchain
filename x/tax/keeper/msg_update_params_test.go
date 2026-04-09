package keeper_test

import (
	"testing"

	appparams "github.com/MANTRA-Chain/inveniam/app/params"
	keepertest "github.com/MANTRA-Chain/inveniam/testutil/keeper"
	"github.com/MANTRA-Chain/inveniam/x/tax/keeper"
	"github.com/MANTRA-Chain/inveniam/x/tax/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
)

func TestMsgUpdateParams(t *testing.T) {
	appparams.SetAddressPrefixes()

	// a valid inveniam address that is marked as blocked (e.g. a module account address)
	blockedAddr := authtypes.NewModuleAddress("distribution").String()
	bk := &keepertest.MockBankKeeper{
		BlockedAddrs: map[string]bool{blockedAddr: true},
	}
	k, ctx, _ := keepertest.TaxKeeperWithBankKeeper(t, bk)
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
			name: "blocked mca address",
			input: &types.MsgUpdateParams{
				Authority:  "inveniam15m77x4pe6w9vtpuqm22qxu0ds7vn4ehz80mwh8",
				McaTax:     "",
				McaAddress: blockedAddr,
			},
			expErr:    true,
			expErrMsg: "is blocked",
		},
		{
			name: "update mca address",
			input: &types.MsgUpdateParams{
				Authority:  "inveniam15m77x4pe6w9vtpuqm22qxu0ds7vn4ehz80mwh8",
				McaTax:     "",
				McaAddress: keepertest.TestSenderAddr,
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
				Authority:  keepertest.TestSenderAddr,
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
