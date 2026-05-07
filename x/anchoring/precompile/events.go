//go:build uint256

package precompile

import (
	"fmt"

	"cosmossdk.io/collections"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	addRegistryEventSig        = "AddRegistry(address,uint64,string)"
	addRecordEventSig          = "AddRecord(address,uint64,uint64,uint64,string)"
	updateRecordStatusEventSig = "UpdateRecordStatus(address,uint64,uint64,uint64,string)"
	grantRoleEventSig          = "GrantRole(address,uint64,string,address,string)"
	revokeRoleEventSig         = "RevokeRole(address,uint64,string,address,string)"
)

var (
	addRegistryEventID        = crypto.Keccak256Hash([]byte(addRegistryEventSig))
	addRecordEventID          = crypto.Keccak256Hash([]byte(addRecordEventSig))
	updateRecordStatusEventID = crypto.Keccak256Hash([]byte(updateRecordStatusEventSig))
	grantRoleEventID          = crypto.Keccak256Hash([]byte(grantRoleEventSig))
	revokeRoleEventID         = crypto.Keccak256Hash([]byte(revokeRoleEventSig))
)

func topicAddress(addr common.Address) common.Hash {
	return common.BytesToHash(common.LeftPadBytes(addr.Bytes(), 32))
}

func pack(typesAndValues ...any) ([]byte, error) {
	if len(typesAndValues)%2 != 0 {
		return nil, fmt.Errorf("pack: expected even number of args (type,value) pairs")
	}

	args := make(abi.Arguments, 0, len(typesAndValues)/2)
	values := make([]any, 0, len(typesAndValues)/2)

	for i := 0; i < len(typesAndValues); i += 2 {
		typ, ok := typesAndValues[i].(string)
		if !ok {
			return nil, fmt.Errorf("pack: expected type string at index %d", i)
		}
		abiType, err := abi.NewType(typ, "", nil)
		if err != nil {
			return nil, err
		}
		args = append(args, abi.Argument{Type: abiType})
		values = append(values, typesAndValues[i+1])
	}

	return args.Pack(values...)
}

func emitLog(ctx sdk.Context, stateDB vm.StateDB, eventID common.Hash, indexedCaller common.Address, data []byte) {
	if stateDB == nil {
		return
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     AnchoringPrecompileAddress,
		Topics:      []common.Hash{eventID, topicAddress(indexedCaller)},
		Data:        data,
		BlockNumber: uint64(ctx.BlockHeight()),
	})
}

func (p Precompile) emitAddRegistryEvent(ctx sdk.Context, stateDB vm.StateDB, caller common.Address, registryID uint64, name string) error {
	data, err := pack("uint64", registryID, "string", name)
	if err != nil {
		return err
	}
	emitLog(ctx, stateDB, addRegistryEventID, caller, data)
	return nil
}

func (p Precompile) emitAddRecordEvent(ctx sdk.Context, stateDB vm.StateDB, caller common.Address, registryID, recordID uint64, checksum string) error {
	index, err := p.keeper.RecordIndices.Get(ctx, collections.Join(registryID, recordID))
	if err != nil {
		return err
	}

	data, err := pack("uint64", registryID, "uint64", recordID, "uint64", index, "string", checksum)
	if err != nil {
		return err
	}
	emitLog(ctx, stateDB, addRecordEventID, caller, data)
	return nil
}

func (p Precompile) emitUpdateRecordStatusEvent(ctx sdk.Context, stateDB vm.StateDB, caller common.Address, registryID, recordID, index uint64, status string) error {
	data, err := pack("uint64", registryID, "uint64", recordID, "uint64", index, "string", status)
	if err != nil {
		return err
	}
	emitLog(ctx, stateDB, updateRecordStatusEventID, caller, data)
	return nil
}

func (p Precompile) emitGrantRoleEvent(ctx sdk.Context, stateDB vm.StateDB, caller common.Address, registryID uint64, checksum string, account common.Address, role string) error {
	data, err := pack("uint64", registryID, "string", checksum, "address", account, "string", role)
	if err != nil {
		return err
	}
	emitLog(ctx, stateDB, grantRoleEventID, caller, data)
	return nil
}

func (p Precompile) emitRevokeRoleEvent(ctx sdk.Context, stateDB vm.StateDB, caller common.Address, registryID uint64, checksum string, account common.Address, role string) error {
	data, err := pack("uint64", registryID, "string", checksum, "address", account, "string", role)
	if err != nil {
		return err
	}
	emitLog(ctx, stateDB, revokeRoleEventID, caller, data)
	return nil
}
