//go:build uint256

package precompile

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

type stateDBWithCode struct {
	vm.StateDB
	code []byte
}

func (s stateDBWithCode) GetCode(common.Address) []byte {
	return s.code
}

func TestIsEOACode(t *testing.T) {
	testCases := []struct {
		name string
		code []byte
		exp  bool
	}{
		{
			name: "empty code is EOA",
			code: nil,
			exp:  true,
		},
		{
			name: "contract bytecode is not EOA",
			code: []byte{0x60, 0x00, 0x60, 0x00, 0x56},
			exp:  false,
		},
		{
			name: "delegation designator is EOA-compatible",
			code: gethtypes.AddressToDelegation(common.HexToAddress("0x00000000000000000000000000000000000000a1")),
			exp:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.exp, isEOACode(tc.code))
		})
	}
}

func TestEnsureEOACaller(t *testing.T) {
	p := Precompile{}
	callerContractCode := common.HexToAddress("0x00000000000000000000000000000000000000b1")
	callerDelegationCode := common.HexToAddress("0x00000000000000000000000000000000000000b2")
	callerConstructorCtx := common.HexToAddress("0x00000000000000000000000000000000000000b3")
	originEOA := common.HexToAddress("0x00000000000000000000000000000000000000e1")
	methods := []string{"addRegistry", "addRecord", "updateRecordStatus", "grantRole", "revokeRole"}

	testCases := []struct {
		name      string
		stateDB   vm.StateDB
		contract  *vm.Contract
		txOrigin  common.Address
		wantError bool
	}{
		{
			name:      "nil inputs",
			stateDB:   nil,
			contract:  nil,
			txOrigin:  common.Address{},
			wantError: true,
		},
		{
			name:      "contract code rejected",
			stateDB:   stateDBWithCode{code: []byte{0x60, 0x00, 0x60, 0x00, 0x56}},
			contract:  vm.NewContract(callerContractCode, common.Address{}, nil, 0, nil),
			txOrigin:  callerContractCode,
			wantError: true,
		},
		{
			name:      "delegation code allowed",
			stateDB:   stateDBWithCode{code: gethtypes.AddressToDelegation(common.HexToAddress("0x00000000000000000000000000000000000000c1"))},
			contract:  vm.NewContract(callerDelegationCode, common.Address{}, nil, 0, nil),
			txOrigin:  callerDelegationCode,
			wantError: false,
		},
		{
			name: "empty code rejected when caller != tx origin (constructor bypass blocked)",
			// Constructor execution has no runtime code persisted yet for the caller address.
			stateDB:   stateDBWithCode{code: nil},
			contract:  vm.NewContract(callerConstructorCtx, common.Address{}, nil, 0, nil),
			txOrigin:  originEOA,
			wantError: true,
		},
		{
			name:      "empty code allowed when caller == tx origin",
			stateDB:   stateDBWithCode{code: nil},
			contract:  vm.NewContract(originEOA, common.Address{}, nil, 0, nil),
			txOrigin:  originEOA,
			wantError: false,
		},
	}

	for _, method := range methods {
		for _, tc := range testCases {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				err := p.ensureEOACaller(tc.stateDB, tc.contract, method, tc.txOrigin)
				if tc.wantError {
					require.Error(t, err)
					require.ErrorIs(t, err, core.ErrSenderNoEOA)
					return
				}
				require.NoError(t, err)
			})
		}
	}
}

func TestEnsureNoValue(t *testing.T) {
	p := Precompile{}
	caller := common.HexToAddress("0x00000000000000000000000000000000000000d1")

	testCases := []struct {
		name      string
		contract  *vm.Contract
		wantError bool
	}{
		{
			name:      "nil contract rejected",
			contract:  nil,
			wantError: true,
		},
		{
			name:      "zero value allowed",
			contract:  vm.NewContract(caller, AnchoringPrecompileAddress, uint256.NewInt(0), 0, nil),
			wantError: false,
		},
		{
			name:      "non-zero value rejected",
			contract:  vm.NewContract(caller, AnchoringPrecompileAddress, uint256.NewInt(1), 0, nil),
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := p.ensureNoValue(tc.contract)
			if tc.wantError {
				require.Error(t, err)
				if tc.contract == nil {
					require.Contains(t, err.Error(), "nil contract")
					return
				}
				require.Contains(t, err.Error(), "cannot receive funds")
				return
			}
			require.NoError(t, err)
		})
	}
}
