//go:build uint256

package precompile

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

type stateDBWithCode struct {
	vm.StateDB
	codes map[common.Address][]byte
}

func (s stateDBWithCode) GetCode(addr common.Address) []byte {
	if code, ok := s.codes[addr]; ok {
		return code
	}
	return nil
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
			stateDB:   stateDBWithCode{codes: map[common.Address][]byte{callerContractCode: {0x60, 0x00, 0x60, 0x00, 0x56}}},
			contract:  vm.NewContract(callerContractCode, common.Address{}, nil, 0, nil),
			txOrigin:  callerContractCode,
			wantError: true,
		},
		{
			name:      "delegation code allowed",
			stateDB:   stateDBWithCode{codes: map[common.Address][]byte{callerDelegationCode: gethtypes.AddressToDelegation(common.HexToAddress("0x00000000000000000000000000000000000000c1"))}},
			contract:  vm.NewContract(callerDelegationCode, common.Address{}, nil, 0, nil),
			txOrigin:  callerDelegationCode,
			wantError: false,
		},
		{
			name: "empty code rejected when caller != tx origin (constructor bypass blocked)",
			// Constructor execution has no runtime code persisted yet for the caller address.
			stateDB:   stateDBWithCode{codes: map[common.Address][]byte{originEOA: nil, callerConstructorCtx: nil}},
			contract:  vm.NewContract(callerConstructorCtx, common.Address{}, nil, 0, nil),
			txOrigin:  originEOA,
			wantError: true,
		},
		{
			name:      "empty code allowed when caller == tx origin",
			stateDB:   stateDBWithCode{codes: map[common.Address][]byte{originEOA: nil}},
			contract:  vm.NewContract(originEOA, common.Address{}, nil, 0, nil),
			txOrigin:  originEOA,
			wantError: false,
		},
		{
			name:      "origin code check rejects non-EOA origin",
			stateDB:   stateDBWithCode{codes: map[common.Address][]byte{originEOA: {0x60, 0x00, 0x60, 0x00, 0x56}}},
			contract:  vm.NewContract(originEOA, common.Address{}, nil, 0, nil),
			txOrigin:  originEOA,
			wantError: true,
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

func TestRevokeRoleRejectsEmptyRole(t *testing.T) {
	p := Precompile{}
	_, err := p.RevokeRole(sdk.Context{}, RevokeRoleCall{Role: ""}, nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "role cannot be empty")
}

func TestEnsureEOACaller_BlocksConstructorBypass(t *testing.T) {
	callerInConstructor := common.HexToAddress("0x00000000000000000000000000000000000000b3")
	originEOA := common.HexToAddress("0x00000000000000000000000000000000000000e1")

	stateDB := stateDBWithCode{
		codes: map[common.Address][]byte{
			callerInConstructor: nil, // constructor code not persisted yet
			originEOA:           nil,
		},
	}

	contract := vm.NewContract(callerInConstructor, AnchoringPrecompileAddress, nil, 1000000, nil)
	p := Precompile{}

	err := p.ensureEOACaller(stateDB, contract, "addRegistry", originEOA)
	require.Error(t, err)
	require.ErrorIs(t, err, core.ErrSenderNoEOA)
}

// TestRegistriesByNameIsQueryOnly pins the ABI surface of the name lookup:
// both selectors are consensus-visible, and adding the filter to registries
// instead would have changed that method's selector.
func TestRegistriesByNameIsQueryOnly(t *testing.T) {
	require.Equal(t, [4]byte{0x42, 0x4d, 0xa2, 0x19}, RegistriesByNameSelector)
	require.Equal(t, [4]byte{0x17, 0xbd, 0x3e, 0x65}, RegistriesSelector,
		"adding registriesByName must not disturb the existing registries selector")

	var p Precompile
	require.False(t, p.IsTransactionID(RegistriesByNameID),
		"registriesByName is a view; a transaction would demand an EOA caller")
	require.Empty(t, p.methodName(RegistriesByNameID))
}
