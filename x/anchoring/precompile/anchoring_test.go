//go:build uint256

package precompile

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/nameindex"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
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

// queryCtx is the context shape eth_call and eth_estimateGas arrive in:
// baseapp's CreateQueryContext passes isCheckTx=true.
func queryCtx() sdk.Context { return sdk.Context{}.WithIsCheckTx(true) }

// blockCtx is the context shape a transaction arrives in: baseapp builds
// finalizeBlockState with isCheckTx=false.
func blockCtx() sdk.Context { return sdk.Context{}.WithIsCheckTx(false) }

// indexedPrecompile is a node that opted into the name index; unindexedPrecompile
// is one that did not. The gate only ever tests NameIndex for nil, so an empty
// Store stands in for an open one.
func indexedPrecompile() Precompile {
	return Precompile{keeper: keeper.Keeper{NameIndex: &nameindex.Store{}}}
}

func unindexedPrecompile() Precompile { return Precompile{} }

func eoaCall(addr common.Address) (vm.StateDB, *vm.Contract) {
	return stateDBWithCode{codes: map[common.Address][]byte{addr: nil}},
		vm.NewContract(addr, AnchoringPrecompileAddress, nil, 1_000_000, nil)
}

func TestRegistriesByNameGate(t *testing.T) {
	eoa := common.HexToAddress("0x00000000000000000000000000000000000000e1")
	contractCaller := common.HexToAddress("0x00000000000000000000000000000000000000b1")

	eoaDB, eoaContract := eoaCall(eoa)
	contractDB := stateDBWithCode{codes: map[common.Address][]byte{
		contractCaller: {0x60, 0x00, 0x60, 0x00, 0x56},
		eoa:            nil,
	}}
	contractInvoked := vm.NewContract(contractCaller, AnchoringPrecompileAddress, nil, 1_000_000, nil)

	testCases := []struct {
		name     string
		p        Precompile
		ctx      sdk.Context
		stateDB  vm.StateDB
		contract *vm.Contract
		txOrigin common.Address
		wantErr  error
	}{
		{
			name:     "eth_call from an EOA on an indexed node is served",
			p:        indexedPrecompile(),
			ctx:      queryCtx(),
			stateDB:  eoaDB,
			contract: eoaContract,
			txOrigin: eoa,
		},
		{
			// The case EOA-gating alone would wave through: a plain transaction
			// sent straight to the precompile has msg.sender == tx.origin and no
			// code at origin, yet it executes in every validator's block.
			name:     "direct EOA transaction is rejected even though it is an EOA",
			p:        indexedPrecompile(),
			ctx:      blockCtx(),
			stateDB:  eoaDB,
			contract: eoaContract,
			txOrigin: eoa,
			wantErr:  ErrRegistriesByNameInTx,
		},
		{
			name:     "contract caller is rejected in a query context",
			p:        indexedPrecompile(),
			ctx:      queryCtx(),
			stateDB:  contractDB,
			contract: contractInvoked,
			txOrigin: eoa,
			wantErr:  core.ErrSenderNoEOA,
		},
		{
			name:     "unindexed node reports the index is off, not a bad request",
			p:        unindexedPrecompile(),
			ctx:      queryCtx(),
			stateDB:  eoaDB,
			contract: eoaContract,
			txOrigin: eoa,
			wantErr:  ErrRegistriesByNameIndexOff,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.p.registriesByNameGate(tc.ctx, tc.stateDB, tc.contract, tc.txOrigin)
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

// TestRegistriesByNameGate_BlockExecutionIsNodeIndependent pins the property the
// whole design rests on. Inside block execution an indexed node and an unindexed
// node must reject the same call with the same error: the consensus gate has to
// be decided before NameIndex is ever consulted. If these two diverged, the two
// nodes would consume different gas and produce different results for the same
// transaction, which is an AppHash divergence — the exact failure the node-local
// index would otherwise introduce.
func TestRegistriesByNameGate_BlockExecutionIsNodeIndependent(t *testing.T) {
	eoa := common.HexToAddress("0x00000000000000000000000000000000000000e1")
	stateDB, contract := eoaCall(eoa)

	indexed := indexedPrecompile().registriesByNameGate(blockCtx(), stateDB, contract, eoa)
	unindexed := unindexedPrecompile().registriesByNameGate(blockCtx(), stateDB, contract, eoa)

	require.ErrorIs(t, indexed, ErrRegistriesByNameInTx)
	require.ErrorIs(t, unindexed, ErrRegistriesByNameInTx,
		"an unindexed node must reject a transaction for the same reason an indexed one does")
}

// TestRegistriesByNameIsQueryOnly pins the ABI surface: registriesByName must
// stay a view, and adding it must not have disturbed the existing selectors.
func TestRegistriesByNameIsQueryOnly(t *testing.T) {
	require.Equal(t, [4]byte{0x55, 0x22, 0xe6, 0xc6}, RegistriesByNameSelector)
	require.Equal(t, [4]byte{0x17, 0xbd, 0x3e, 0x65}, RegistriesSelector,
		"adding registriesByName must not disturb the existing registries selector")
	require.Equal(t, [4]byte{0x64, 0xd2, 0x52, 0x95}, AddRecordSelector)
	require.Equal(t, [4]byte{0xc7, 0xbe, 0x5e, 0x37}, RecordsSelector)

	var p Precompile
	require.False(t, p.IsTransactionID(RegistriesByNameID),
		"registriesByName is a view: the transaction path would apply the wrong gate")
	require.Empty(t, p.methodName(RegistriesByNameID),
		"methodName only names transaction methods")
}

func TestMatchModeFromABI(t *testing.T) {
	testCases := []struct {
		mode uint8
		want types.RegistryNameMatchMode
	}{
		{0, types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_UNSPECIFIED},
		{1, types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_EXACT},
		{2, types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_PREFIX},
		{3, types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_SUFFIX},
		{4, types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_CONTAINS},
	}
	for _, tc := range testCases {
		got, err := matchModeFromABI(tc.mode)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}

	// An unknown mode is rejected rather than silently treated as exact, so a
	// caller asking for a match mode this node does not have never reads the
	// empty result as "no such registry".
	for _, bad := range []uint8{5, 200, 255} {
		_, err := matchModeFromABI(bad)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid matchMode")
	}
}

// TestChargeIndexReadGas checks the index read is metered at all: the SQLite
// work happens outside the SDK store, so without this a caller's gas cap would
// place no bound on how much scanning one eth_call can buy.
func TestChargeIndexReadGas(t *testing.T) {
	p := Precompile{}
	p.KvGasConfig = storetypes.KVGasConfig()

	ctx := queryCtx().WithGasMeter(storetypes.NewGasMeter(10_000_000))
	p.chargeIndexReadGas(ctx, nil)
	require.Zero(t, ctx.GasMeter().GasConsumed(), "no results, no charge")

	one := []Registry{{Id: 1, Name: "fund-documents"}}
	p.chargeIndexReadGas(ctx, one)
	afterOne := ctx.GasMeter().GasConsumed()
	require.Positive(t, afterOne)

	p.chargeIndexReadGas(ctx, []Registry{one[0], one[0], one[0]})
	require.Equal(t, 4*afterOne, ctx.GasMeter().GasConsumed(),
		"cost scales with the number of rows returned")
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
