package tax_test

import (
	"testing"

	appparams "github.com/NVNM-Chain/nvnmchain/app/params"
	keepertest "github.com/NVNM-Chain/nvnmchain/testutil/keeper"
	"github.com/NVNM-Chain/nvnmchain/testutil/nullify"
	tax "github.com/NVNM-Chain/nvnmchain/x/tax/module"
	"github.com/NVNM-Chain/nvnmchain/x/tax/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	k, ctx, _ := keepertest.TaxKeeper(t)
	err := tax.InitGenesis(ctx, k, genesisState)
	require.NoError(t, err)
	got, err := tax.ExportGenesis(ctx, k)
	require.NoError(t, err)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}

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
			desc: "valid genesis state",
			genState: &types.GenesisState{
				Params: types.NewParams("0.1", "nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca"),
			},
			valid: true,
		},
		{
			desc: "negative tax is invalid",
			genState: &types.GenesisState{
				Params: types.NewParams("-0.5", types.DefaultTaxAddress),
			},
			valid: false,
		},
		{
			desc: "tax greater than 1 is invalid",
			genState: &types.GenesisState{
				Params: types.NewParams("1.5", types.DefaultTaxAddress),
			},
			valid: false,
		},
		{
			desc: "invalid bech32 address",
			genState: &types.GenesisState{
				Params: types.NewParams(types.DefaultTax, "invalid_address"),
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

func TestParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  types.Params
		wantErr bool
	}{
		{
			name:    "default params",
			params:  types.DefaultParams(),
			wantErr: false,
		},
		{
			name:    "valid params",
			params:  types.NewParams("0.1", "nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca"),
			wantErr: false,
		},
		{
			name:    "invalid tax",
			params:  types.NewParams("-0.1", types.DefaultTaxAddress),
			wantErr: true,
		},
		{
			name:    "invalid tax address",
			params:  types.NewParams(types.DefaultTax, "invalid_address"),
			wantErr: true,
		},
		{
			name:    "tax too high",
			params:  types.NewParams("1.1", "nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateProportion(t *testing.T) {
	tests := []struct {
		name       string
		proportion string
		wantErr    bool
	}{
		{"valid tax", "0.1", false},
		{"zero tax", "0", false},
		{"max tax", "0.3", false},
		{"negative tax", "-0.1", true},
		{"tax greater than 1", "1.1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateTax(tt.proportion)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateTaxAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{"valid address", "nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca", false},
		{"empty address", "", true},
		{"invalid bech32", "invalid_address", true},
		{"wrong prefix", "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateTaxAddress(tt.address)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
