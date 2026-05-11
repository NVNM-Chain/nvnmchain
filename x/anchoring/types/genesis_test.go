package types_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	appparams "github.com/NVNM-Chain/nvnmchain/app/params"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	"github.com/stretchr/testify/require"
)

func defaultAnchoringFee() sdk.Coin {
	return sdk.NewCoin(types.DefaultAnchoringFeeDenom, types.DefaultAnchoringFeeAmount)
}

func TestGenesisState_ValidateParams(t *testing.T) {
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
				Params: types.NewParams("nvnm1axznhnm82lah8qqvp9hxdad49yx3s5dcmnx073", defaultAnchoringFee()),
			},
			valid: false,
		},
		{
			desc: "valid custom parameters",
			genState: &types.GenesisState{
				Params: types.NewParams("nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca", defaultAnchoringFee()),
			},
			valid: true,
		},
		{
			desc: "empty admin address is invalid",
			genState: &types.GenesisState{
				Params: types.NewParams("", defaultAnchoringFee()),
			},
			valid: false,
		},
		{
			desc: "tax address with wrong prefix is invalid",
			genState: &types.GenesisState{
				Params: types.NewParams("cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzutu9", defaultAnchoringFee()),
			},
			valid: false,
		},
		{
			desc: "zero anchoring fee amount is invalid",
			genState: &types.GenesisState{
				Params: types.NewParams(
					"nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca",
					sdk.NewInt64Coin(types.DefaultAnchoringFeeDenom, 0),
				),
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

func TestGenesisState_ValidateRegistriesAndRecords(t *testing.T) {
	appparams.SetAddressPrefixes()

	validParams := types.NewParams("nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca", defaultAnchoringFee())

	validRegistry := types.Registry{
		Id:      1,
		Name:    "kyc_registry",
		Creator: "nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca",
	}

	validRecord := types.Record{
		Registry:     "kyc_registry",
		RecordId:     1,
		Index:        1,
		IsLatest:     true,
		Checksum:     "abc123",
		ChecksumAlgo: "SHA-256",
		Uri:          "ipfs://QmExample",
		Metadata:     `{"key":"value"}`,
		Status:       "active",
	}

	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc: "valid registries and records",
			genState: &types.GenesisState{
				Params:     validParams,
				Registries: map[uint64]types.Registry{1: validRegistry},
				Records:    []types.Record{validRecord},
			},
			valid: true,
		},
		{
			desc: "no records is valid",
			genState: &types.GenesisState{
				Params:     validParams,
				Registries: map[uint64]types.Registry{1: validRegistry},
				Records:    []types.Record{},
			},
			valid: true,
		},
		{
			desc: "registry id zero is invalid",
			genState: &types.GenesisState{
				Params: validParams,
				Registries: map[uint64]types.Registry{
					0: {Id: 0, Name: "kyc_registry", Creator: "nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca"},
				},
			},
			valid: false,
		},
		{
			desc: "registry map key mismatch is invalid",
			genState: &types.GenesisState{
				Params: validParams,
				Registries: map[uint64]types.Registry{
					2: validRegistry, // validRegistry.Id == 1
				},
			},
			valid: false,
		},
		{
			desc: "empty registry name is invalid",
			genState: &types.GenesisState{
				Params: validParams,
				Registries: map[uint64]types.Registry{
					1: {Id: 1, Name: "", Creator: "nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca"},
				},
			},
			valid: false,
		},
		{
			desc: "registry name exceeding max length is invalid",
			genState: &types.GenesisState{
				Params: validParams,
				Registries: map[uint64]types.Registry{
					1: {Id: 1, Name: strings.Repeat("a", types.MaxRegistryNameLen+1), Creator: "nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca"},
				},
			},
			valid: false,
		},
		{
			desc: "empty registry creator is invalid",
			genState: &types.GenesisState{
				Params: validParams,
				Registries: map[uint64]types.Registry{
					1: {Id: 1, Name: "kyc_registry", Creator: ""},
				},
			},
			valid: false,
		},
		{
			desc: "duplicate registry name is invalid",
			genState: &types.GenesisState{
				Params: validParams,
				Registries: map[uint64]types.Registry{
					1: {Id: 1, Name: "kyc_registry", Creator: "nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca"},
					2: {Id: 2, Name: "kyc_registry", Creator: "nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca"},
				},
			},
			valid: false,
		},
		{
			desc: "record referencing unknown registry is invalid",
			genState: &types.GenesisState{
				Params:     validParams,
				Registries: map[uint64]types.Registry{1: validRegistry},
				Records: []types.Record{
					{
						Registry:     "nonexistent_registry",
						RecordId:     1,
						Index:        1,
						Checksum:     "abc123",
						ChecksumAlgo: "SHA-256",
						Uri:          "ipfs://QmExample",
						Metadata:     `{"key":"value"}`,
						Status:       "active",
					},
				},
			},
			valid: false,
		},
		{
			desc: "record with zero record_id is invalid",
			genState: &types.GenesisState{
				Params:     validParams,
				Registries: map[uint64]types.Registry{1: validRegistry},
				Records: []types.Record{
					{
						Registry:     "kyc_registry",
						RecordId:     0,
						Index:        1,
						Checksum:     "abc123",
						ChecksumAlgo: "SHA-256",
						Uri:          "ipfs://QmExample",
						Metadata:     `{"key":"value"}`,
						Status:       "active",
					},
				},
			},
			valid: false,
		},
		{
			desc: "record with empty checksum is invalid",
			genState: &types.GenesisState{
				Params:     validParams,
				Registries: map[uint64]types.Registry{1: validRegistry},
				Records: []types.Record{
					{
						Registry:     "kyc_registry",
						RecordId:     1,
						Index:        1,
						Checksum:     "",
						ChecksumAlgo: "SHA-256",
						Uri:          "ipfs://QmExample",
						Metadata:     `{"key":"value"}`,
						Status:       "active",
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicate record key is invalid",
			genState: &types.GenesisState{
				Params:     validParams,
				Registries: map[uint64]types.Registry{1: validRegistry},
				Records: []types.Record{
					validRecord,
					validRecord, // same registry, record_id, index
				},
			},
			valid: false,
		},
		{
			desc: "same record_id different index with correct is_latest is valid",
			genState: &types.GenesisState{
				Params:     validParams,
				Registries: map[uint64]types.Registry{1: validRegistry},
				Records: []types.Record{
					{
						Registry:     "kyc_registry",
						RecordId:     1,
						Index:        1,
						IsLatest:     false,
						Checksum:     "abc123",
						ChecksumAlgo: "SHA-256",
						Uri:          "ipfs://QmExample",
						Metadata:     `{"key":"value"}`,
						Status:       "active",
					},
					{
						Registry:     "kyc_registry",
						RecordId:     1,
						Index:        2,
						IsLatest:     true,
						Checksum:     "def456",
						ChecksumAlgo: "SHA-256",
						Uri:          "ipfs://QmExample2",
						Metadata:     `{"key":"value2"}`,
						Status:       "active",
					},
				},
			},
			valid: true,
		},
		{
			desc: "is_latest on non-max index is invalid",
			genState: &types.GenesisState{
				Params:     validParams,
				Registries: map[uint64]types.Registry{1: validRegistry},
				Records: []types.Record{
					{
						Registry:     "kyc_registry",
						RecordId:     1,
						Index:        1,
						IsLatest:     true, // wrong: not the highest index
						Checksum:     "abc123",
						ChecksumAlgo: "SHA-256",
						Uri:          "ipfs://QmExample",
						Metadata:     `{"key":"value"}`,
						Status:       "active",
					},
					{
						Registry:     "kyc_registry",
						RecordId:     1,
						Index:        2,
						IsLatest:     false,
						Checksum:     "def456",
						ChecksumAlgo: "SHA-256",
						Uri:          "ipfs://QmExample2",
						Metadata:     `{"key":"value2"}`,
						Status:       "active",
					},
				},
			},
			valid: false,
		},
		{
			desc: "multiple is_latest in same group is invalid",
			genState: &types.GenesisState{
				Params:     validParams,
				Registries: map[uint64]types.Registry{1: validRegistry},
				Records: []types.Record{
					{
						Registry:     "kyc_registry",
						RecordId:     1,
						Index:        1,
						IsLatest:     true,
						Checksum:     "abc123",
						ChecksumAlgo: "SHA-256",
						Uri:          "ipfs://QmExample",
						Metadata:     `{"key":"value"}`,
						Status:       "active",
					},
					{
						Registry:     "kyc_registry",
						RecordId:     1,
						Index:        2,
						IsLatest:     true,
						Checksum:     "def456",
						ChecksumAlgo: "SHA-256",
						Uri:          "ipfs://QmExample2",
						Metadata:     `{"key":"value2"}`,
						Status:       "active",
					},
				},
			},
			valid: false,
		},
		{
			desc: "no is_latest in group is invalid",
			genState: &types.GenesisState{
				Params:     validParams,
				Registries: map[uint64]types.Registry{1: validRegistry},
				Records: []types.Record{
					{
						Registry:     "kyc_registry",
						RecordId:     1,
						Index:        1,
						IsLatest:     false,
						Checksum:     "abc123",
						ChecksumAlgo: "SHA-256",
						Uri:          "ipfs://QmExample",
						Metadata:     `{"key":"value"}`,
						Status:       "active",
					},
				},
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

func TestGenesisState_ValidateRBAC(t *testing.T) {
	appparams.SetAddressPrefixes()

	validParams := types.NewParams("nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca", defaultAnchoringFee())
	validKey := "\x01\x02\x03" // non-empty binary-encoded key

	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc: "valid RBAC maps",
			genState: &types.GenesisState{
				Params:      validParams,
				RoleAdmins:  map[string][]byte{validKey: {0x01}},
				RoleMembers: map[string][]byte{validKey: {}},
			},
			valid: true,
		},
		{
			desc: "empty RBAC maps are valid",
			genState: &types.GenesisState{
				Params:      validParams,
				RoleAdmins:  map[string][]byte{},
				RoleMembers: map[string][]byte{},
			},
			valid: true,
		},
		{
			desc: "empty role_admins key is invalid",
			genState: &types.GenesisState{
				Params:     validParams,
				RoleAdmins: map[string][]byte{"": {0x01}},
			},
			valid: false,
		},
		{
			desc: "empty role_admins value is invalid",
			genState: &types.GenesisState{
				Params:     validParams,
				RoleAdmins: map[string][]byte{validKey: {}},
			},
			valid: false,
		},
		{
			desc: "empty role_members key is invalid",
			genState: &types.GenesisState{
				Params:      validParams,
				RoleMembers: map[string][]byte{"": {}},
			},
			valid: false,
		},
		{
			desc: "role_members with nil marker value is valid",
			genState: &types.GenesisState{
				Params:      validParams,
				RoleMembers: map[string][]byte{validKey: nil},
			},
			valid: true,
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
