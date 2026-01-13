//go:build uint256

package precompile

import (
	"encoding/binary"
	"errors"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/cosmos/evm/x/vm/statedb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/yihuang/go-abi"
)

//go:generate go run github.com/yihuang/go-abi/cmd -var=CommonABI -output common.abi.go -uint256
var CommonABI = []string{
	"struct Coin {string denom; uint256 amount;}",
	"struct DecCoin {string denom; uint256 amount; uint8 precision;}",
	"struct Dec {uint256 value; uint8 precision;}",
	"struct Height { uint64 revisionNumber; uint64 revisionHeight; }",
	"struct PageRequest { bytes key; uint64 offset; uint64 limit; bool countTotal; bool reverse; }",
	"struct PageResponse { bytes nextKey; uint64 total; }",
	"struct ICS20Allocation { string sourcePort; string sourceChannel; Coin[] spendLimit; string[] allowList; string[] allowedPacketData; }",

	// there's no dedicated tyeps for structs in ABI,
	// the dummy function to keep them in the ABI
	"function dummy(Coin a, DecCoin b, Dec c, Height d, PageRequest e, PageResponse f, ICS20Allocation g)",
}

// ErrUnknownMethodID is raised when the methodID is not known.
var ErrUnknownMethodID = "unknown method id: %d"

// NativeAction abstract the native execution logic of the stateful precompile, it's passed to the base `Precompile`
// struct, base `Precompile` struct will handle things the native context setup, gas management, panic recovery etc,
// before and after the execution.
//
// It's usually implemented by the precompile itself.
type NativeAction func(ctx sdk.Context) ([]byte, error)

func RunNativeAction(p cmn.Precompile, evm *vm.EVM, contract *vm.Contract, action NativeAction) (bz []byte, err error) {
	stateDB, ok := evm.StateDB.(*statedb.StateDB)
	if !ok {
		return nil, errors.New(cmn.ErrNotRunInEvm)
	}

	// get the stateDB cache ctx
	ctx, err := stateDB.GetCacheContext()
	if err != nil {
		return nil, err
	}

	// take a snapshot of the current state before any changes
	// to be able to revert the changes
	snapshot := stateDB.MultiStoreSnapshot()
	events := ctx.EventManager().Events()

	// add precompileCall entry on the stateDB journal
	// this allows to revert the changes within an evm tx
	if err := stateDB.AddPrecompileFn(snapshot, events); err != nil {
		return nil, err
	}

	// commit the current changes in the cache ctx
	// to get the updated state for the precompile call
	if err := stateDB.CommitWithCacheCtx(); err != nil {
		return nil, err
	}

	initialGas := ctx.GasMeter().GasConsumed()

	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	// set the default SDK gas configuration to track gas usage
	// we are changing the gas meter type, so it panics gracefully when out of gas
	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(contract.Gas)).
		WithKVGasConfig(p.KvGasConfig).
		WithTransientKVGasConfig(p.TransientKVGasConfig)

	// we need to consume the gas that was already used by the EVM
	ctx.GasMeter().ConsumeGas(initialGas, "creating a new gas meter")

	bz, err = action(ctx)
	if err != nil {
		return bz, err
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas

	if !contract.UseGas(cost, nil, tracing.GasChangeCallPrecompiledContract) {
		return nil, vm.ErrOutOfGas
	}

	return bz, nil
}

// SplitMethodID splits the method id from the input data.
func SplitMethodID(input []byte) (uint32, []byte, error) {
	if len(input) < 4 {
		return 0, nil, errors.New("invalid input length")
	}

	methodID := binary.BigEndian.Uint32(input)
	return methodID, input[4:], nil
}

// ParseMethod splits method id, and check if it's allowed in readOnly mode.
func ParseMethod(input []byte, readOnly bool, isTransaction func(uint32) bool) (uint32, []byte, error) {
	methodID, input, err := SplitMethodID(input)
	if err != nil {
		return 0, nil, err
	}

	if readOnly && isTransaction(methodID) {
		return 0, nil, vm.ErrWriteProtection
	}

	return methodID, input, nil
}

func Run[I any, PI interface {
	*I
	abi.Decode
}, O abi.Encode](
	ctx sdk.Context,
	fn func(sdk.Context, I) (O, error),
	input []byte,
) ([]byte, error) {
	var in I
	if _, err := PI(&in).Decode(input); err != nil {
		return nil, err
	}

	out, err := fn(ctx, in)
	if err != nil {
		return nil, err
	}

	return out.Encode()
}

func RunWithStateDB[I any, PI interface {
	*I
	abi.Decode
}, O abi.Encode](
	ctx sdk.Context,
	fn func(sdk.Context, I, vm.StateDB, *vm.Contract) (O, error),
	input []byte,
	stateDB vm.StateDB,
	contract *vm.Contract,
) ([]byte, error) {
	var in I
	if _, err := PI(&in).Decode(input); err != nil {
		return nil, err
	}

	out, err := fn(ctx, in, stateDB, contract)
	if err != nil {
		return nil, err
	}

	return out.Encode()
}

func (p PageRequest) ToPageRequest() *query.PageRequest {
	return &query.PageRequest{
		Key:        p.Key,
		Offset:     p.Offset,
		Limit:      p.Limit,
		CountTotal: p.CountTotal,
		Reverse:    p.Reverse,
	}
}

func FromPageResponse(pr *query.PageResponse) (p PageResponse) {
	if pr == nil {
		return
	}

	p.NextKey = pr.NextKey
	p.Total = pr.Total
	return
}
