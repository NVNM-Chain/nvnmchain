//go:build uint256

package precompile

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
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
	methods := []string{"addRegistry", "addRecord", "updateRecordStatus", "grantRole", "revokeRole"}

	testCases := []struct {
		name      string
		stateDB   vm.StateDB
		contract  *vm.Contract
		wantError bool
	}{
		{
			name:      "nil inputs",
			stateDB:   nil,
			contract:  nil,
			wantError: true,
		},
		{
			name:      "contract code rejected",
			stateDB:   stateDBWithCode{code: []byte{0x60, 0x00, 0x60, 0x00, 0x56}},
			contract:  vm.NewContract(callerContractCode, common.Address{}, nil, 0, nil),
			wantError: true,
		},
		{
			name:      "delegation code allowed",
			stateDB:   stateDBWithCode{code: gethtypes.AddressToDelegation(common.HexToAddress("0x00000000000000000000000000000000000000c1"))},
			contract:  vm.NewContract(callerDelegationCode, common.Address{}, nil, 0, nil),
			wantError: false,
		},
	}

	for _, method := range methods {
		for _, tc := range testCases {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				err := p.ensureEOACaller(tc.stateDB, tc.contract, method)
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
