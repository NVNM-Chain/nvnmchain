package types_test

import (
	"testing"

	appparams "github.com/MANTRA-Chain/nvnmchain/app/params"
	"github.com/MANTRA-Chain/nvnmchain/x/anchoring/types"
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
			desc: "invalid bech32 address",
			genState: &types.GenesisState{
				Params: types.NewParams("nvnm1axznhnm82lah8qqvp9hxdad49yx3s5dcmnx073"),
			},
			valid: false,
		},
		{
			desc: "valid custom parameters",
			genState: &types.GenesisState{
				Params: types.NewParams("nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca"),
			},
			valid: true,
		},
		{
			desc: "empty admin address is invalid",
			genState: &types.GenesisState{
				Params: types.NewParams(""),
			},
			valid: false,
		},
		{
			desc: "mca address with wrong prefix is invalid",
			genState: &types.GenesisState{
				Params: types.NewParams("cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzutu9"),
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
