//go:build uint256

package precompile

import (
	"cosmossdk.io/collections"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/yihuang/go-abi"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// emitEvent encodes ev with the generated ABI encoders and appends it to the
// EVM logs of the current transaction, making it visible in the transaction
// receipt. Encoding via the generated code keeps the emitted topics and data in
// lockstep with the event declarations in HumanABI.
func emitEvent(ctx sdk.Context, stateDB vm.StateDB, ev abi.Event) error {
	if stateDB == nil {
		return nil
	}

	topics, data, err := abi.EncodeEvent(ev)
	if err != nil {
		return err
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     AnchoringPrecompileAddress,
		Topics:      topics,
		Data:        data,
		BlockNumber: uint64(ctx.BlockHeight()),
	})
	return nil
}

func (p Precompile) emitAddRegistryEvent(ctx sdk.Context, stateDB vm.StateDB, caller common.Address, registryID uint64, name string) error {
	return emitEvent(ctx, stateDB, NewAddRegistryEvent(caller, registryID, name))
}

func (p Precompile) emitAddRecordEvent(ctx sdk.Context, stateDB vm.StateDB, caller common.Address, registryID, recordID uint64, checksum string) error {
	index, err := p.keeper.RecordIndices.Get(ctx, collections.Join(registryID, recordID))
	if err != nil {
		return err
	}

	return emitEvent(ctx, stateDB, NewAddRecordEvent(caller, registryID, recordID, index, checksum))
}

func (p Precompile) emitUpdateRecordStatusEvent(ctx sdk.Context, stateDB vm.StateDB, caller common.Address, registryID, recordID, index uint64, status string) error {
	return emitEvent(ctx, stateDB, NewUpdateRecordStatusEvent(caller, registryID, recordID, index, status))
}

func (p Precompile) emitGrantRoleEvent(ctx sdk.Context, stateDB vm.StateDB, caller common.Address, registryID uint64, checksum string, account common.Address, role string) error {
	return emitEvent(ctx, stateDB, NewGrantRoleEvent(caller, registryID, checksum, account, role))
}

func (p Precompile) emitRevokeRoleEvent(ctx sdk.Context, stateDB vm.StateDB, caller common.Address, registryID uint64, checksum string, account common.Address, role string) error {
	return emitEvent(ctx, stateDB, NewRevokeRoleEvent(caller, registryID, checksum, account, role))
}
