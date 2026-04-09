package types_test

import (
	"testing"

	appparams "github.com/MANTRA-Chain/nvnmchain/app/params"
	"github.com/MANTRA-Chain/nvnmchain/x/tax/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	appparams.SetAddressPrefixes()
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "negative proportion is invalid",
			genState: &types.GenesisState{
				Params: types.NewParams("-0.5", types.DefaultTaxAddress),
			},
			valid: false,
		},
		{
			desc: "invalid bech32 address",
			genState: &types.GenesisState{
				Params: types.NewParams(types.DefaultTax, "nvnm1axznhnm82lah8qqvp9hxdad49yx3s5dcmnx073"),
			},
			valid: false,
		},
		{
			desc: "valid custom parameters",
			genState: &types.GenesisState{
				Params: types.NewParams("0.1", "nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca"),
			},
			valid: true,
		},
		{
			desc: "tax greater than 1 is invalid",
			genState: &types.GenesisState{
				Params: types.NewParams("1.5", types.DefaultTaxAddress),
			},
			valid: false,
		},
		{
			desc: "empty tax address is invalid",
			genState: &types.GenesisState{
				Params: types.NewParams(types.DefaultTax, ""),
			},
			valid: false,
		},
		{
			desc: "tax of 0.5 is invalid",
			genState: &types.GenesisState{
				Params: types.NewParams("0.5", types.DefaultTaxAddress),
			},
			valid: false,
		},
		{
			desc: "tax of 0 is valid",
			genState: &types.GenesisState{
				Params: types.NewParams("0", types.DefaultTaxAddress),
			},
			valid: true,
		},
		{
			desc: "tax of 1 is invalid",
			genState: &types.GenesisState{
				Params: types.NewParams("1", types.DefaultTaxAddress),
			},
			valid: false,
		},
		{
			desc: "tax address with wrong prefix is invalid",
			genState: &types.GenesisState{
				Params: types.NewParams(types.DefaultTax, "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzutu9"),
			},
			valid: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
