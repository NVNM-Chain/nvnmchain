//go:build uint256

package precompile

import (
	"fmt"

	cmn "github.com/cosmos/evm/precompiles/common"

	storetypes "cosmossdk.io/store/types"
	invcmn "github.com/NVNM-Chain/nvnmchain/precompiles"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/evm/precompiles/erc20"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

//go:generate go run github.com/yihuang/go-abi/cmd -var=HumanABI -output anchoring.abi.go --external-tuples PageRequest=cmn.PageRequest,PageResponse=cmn.PageResponse --imports cmn=github.com/NVNM-Chain/nvnmchain/precompiles -uint256
var HumanABI = []string{
	"struct Record {string registry; string uri; string checksum; string checksumAlgo; string metadata; string timestamp; string status; uint64 recordId; uint64 index; bool isLatest}",
	"struct Registry {uint64 id; string name; string description; string creator; string createdAt; string metadata}",
	"struct PageRequest { bytes key; uint64 offset; uint64 limit; bool countTotal; bool reverse; }",
	"struct PageResponse { bytes nextKey; uint64 total; }",

	"function addRegistry(string name, string description, string metadata) returns (uint64 registryId)",
	"function addRecord(Record record) returns (uint64 recordId)",
	"function updateRecordStatus(uint64 registryId, uint64 recordId, uint64 index, string status) returns ()",
	"function records(string registry, string checksum, uint64 recordId, uint64 index, PageRequest pagination) returns (Record[] records, PageResponse pagination)",
	"function registries(uint64 registryId, string name, PageRequest pagination) returns (Registry[] registries, PageResponse pagination)",

	"function grantRole(uint64 registryId, string checksum, address account, string role) returns ()",
	"function revokeRole(uint64 registryId, string checksum, address account, string role) returns ()",
}

var AnchoringPrecompileAddress = common.HexToAddress("0x0000000000000000000000000000000000000a00")

var _ vm.PrecompiledContract = &Precompile{}

// Precompile defines the bank precompile
type Precompile struct {
	cmn.Precompile
	keeper    keeper.Keeper
	msgServer types.MsgServer
}

// NewPrecompile creates a new bank Precompile instance implementing the
// PrecompiledContract interface.
func NewPrecompile(
	k keeper.Keeper,
) (*Precompile, error) {
	// NOTE: we set an empty gas configuration to avoid extra gas costs
	// during the run execution
	p := &Precompile{
		Precompile: cmn.Precompile{
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
		},
		keeper:    k,
		msgServer: keeper.NewMsgServerImpl(k),
	}

	// SetAddress defines the address of the bank compile contract.
	p.SetAddress(AnchoringPrecompileAddress)

	return p, nil
}

func (p Precompile) RequiredGas(input []byte) uint64 {
	methodID, input, err := invcmn.SplitMethodID(input)
	if err != nil {
		return 0
	}
	return p.Precompile.RequiredGas(input, p.IsTransactionID(methodID))
}

func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	return invcmn.RunNativeAction(p.Precompile, evm, contract, func(ctx sdk.Context) ([]byte, error) {
		return p.Execute(ctx, evm.StateDB, contract, readOnly, evm.TxContext.Origin)
	})
}

func (p Precompile) Execute(ctx sdk.Context, stateDB vm.StateDB, contract *vm.Contract, readOnly bool, txOrigin common.Address) ([]byte, error) {
	if contract == nil {
		return nil, fmt.Errorf("contract is nil")
	}

	methodID, input, err := invcmn.ParseMethod(contract.Input, readOnly, p.IsTransactionID)
	if err != nil {
		return nil, err
	}

	if err := p.ensureNoValue(contract); err != nil {
		return encodeRevertReason(err.Error()), vm.ErrExecutionReverted
	}

	if p.IsTransactionID(methodID) {
		if err := p.ensureEOACaller(stateDB, contract, p.methodName(methodID), txOrigin); err != nil {
			return encodeRevertReason(core.ErrSenderNoEOA.Error()), vm.ErrExecutionReverted
		}
	}

	switch methodID {
	case AddRegistryID:
		return invcmn.RunWithStateDB(ctx, p.AddRegistry, input, stateDB, contract)
	case AddRecordID:
		return invcmn.RunWithStateDB(ctx, p.AddRecord, input, stateDB, contract)
	case UpdateRecordStatusID:
		return invcmn.RunWithStateDB(ctx, p.UpdateRecordStatus, input, stateDB, contract)
	case RecordsID:
		return invcmn.Run(ctx, p.Records, input)
	case RegistriesID:
		return invcmn.Run(ctx, p.Registries, input)
	case GrantRoleID:
		return invcmn.RunWithStateDB(ctx, p.GrantRole, input, stateDB, contract)
	case RevokeRoleID:
		return invcmn.RunWithStateDB(ctx, p.RevokeRole, input, stateDB, contract)
	}

	return nil, fmt.Errorf(invcmn.ErrUnknownMethodID, methodID)
}

// IsTransaction checks if the given method name corresponds to a transaction or query.
// It returns true for state-modifying methods.
func (Precompile) IsTransactionID(methodID uint32) bool {
	return methodID == AddRegistryID || methodID == AddRecordID || methodID == UpdateRecordStatusID || methodID == GrantRoleID || methodID == RevokeRoleID
}

func isEOACode(code []byte) bool {
	if len(code) == 0 {
		return true
	}
	_, delegated := gethtypes.ParseDelegation(code)
	return delegated
}

func (Precompile) methodName(methodID uint32) string {
	switch methodID {
	case AddRegistryID:
		return "addRegistry"
	case AddRecordID:
		return "addRecord"
	case UpdateRecordStatusID:
		return "updateRecordStatus"
	case GrantRoleID:
		return "grantRole"
	case RevokeRoleID:
		return "revokeRole"
	default:
		return ""
	}
}

func encodeRevertReason(reason string) []byte {
	bz, err := evmtypes.RevertReasonBytes(reason)
	if err != nil {
		return nil
	}
	return bz
}

func (Precompile) ensureEOACaller(stateDB vm.StateDB, contract *vm.Contract, method string, txOrigin common.Address) error {
	if stateDB == nil || contract == nil {
		return fmt.Errorf("%w: method %s", core.ErrSenderNoEOA, method)
	}
	if contract.Caller() != txOrigin {
		return fmt.Errorf("%w: caller %v != tx.origin %v, method: %s", core.ErrSenderNoEOA, contract.Caller().Hex(), txOrigin.Hex(), method)
	}
	code := stateDB.GetCode(txOrigin)
	if !isEOACode(code) {
		return fmt.Errorf("%w: tx.origin %v has non-EOA code len(code): %d, method: %s", core.ErrSenderNoEOA, txOrigin.Hex(), len(code), method)
	}
	return nil
}

func (Precompile) ensureNoValue(contract *vm.Contract) error {
	if contract == nil {
		return fmt.Errorf("nil contract")
	}
	if value := contract.Value(); value != nil && value.Sign() == 1 {
		return fmt.Errorf(erc20.ErrCannotReceiveFunds, value.String())
	}
	return nil
}

func (p Precompile) AddRegistry(
	ctx sdk.Context,
	input AddRegistryCall,
	stateDB vm.StateDB,
	contract *vm.Contract,
) (*AddRegistryReturn, error) {
	senderStr, err := p.keeper.BytesToString(contract.Caller().Bytes())
	if err != nil {
		return nil, err
	}
	msg := &types.MsgAddRegistry{
		Name:        input.Name,
		Description: input.Description,
		Sender:      senderStr,
		Metadata:    input.Metadata,
	}
	res, err := p.msgServer.AddRegistry(ctx, msg)
	if err != nil {
		return nil, err
	}
	if err := p.emitAddRegistryEvent(ctx, stateDB, contract.Caller(), res.RegistryId, input.Name); err != nil {
		return nil, err
	}
	return &AddRegistryReturn{RegistryId: res.RegistryId}, nil
}

func (p Precompile) AddRecord(
	ctx sdk.Context,
	input AddRecordCall,
	stateDB vm.StateDB,
	contract *vm.Contract,
) (*AddRecordReturn, error) {
	senderStr, err := p.keeper.BytesToString(contract.Caller().Bytes())
	if err != nil {
		return nil, err
	}

	doc := FromABIRecord(input.Record)
	msg := &types.MsgAddRecord{
		Sender: senderStr,
		Record: &doc,
	}

	res, err := p.msgServer.AddRecord(ctx, msg)
	if err != nil {
		return nil, err
	}
	registryID, err := p.keeper.RegistryIdByName.Get(ctx, doc.Registry)
	if err != nil {
		return nil, err
	}
	if err := p.emitAddRecordEvent(ctx, stateDB, contract.Caller(), registryID, res.RecordId, doc.Checksum); err != nil {
		return nil, err
	}
	return &AddRecordReturn{RecordId: res.RecordId}, nil
}

func (p Precompile) UpdateRecordStatus(
	ctx sdk.Context,
	input UpdateRecordStatusCall,
	stateDB vm.StateDB,
	contract *vm.Contract,
) (*UpdateRecordStatusReturn, error) {
	sender, err := p.keeper.BytesToString(contract.Caller().Bytes())
	if err != nil {
		return nil, err
	}

	msg := &types.MsgUpdateRecordStatus{
		Editor:     sender,
		RegistryId: input.RegistryId,
		RecordId:   input.RecordId,
		Index:      input.Index,
		Status:     input.Status,
	}

	if _, err := p.msgServer.UpdateRecordStatus(ctx, msg); err != nil {
		return nil, err
	}
	if err := p.emitUpdateRecordStatusEvent(ctx, stateDB, contract.Caller(), input.RegistryId, input.RecordId, input.Index, input.Status); err != nil {
		return nil, err
	}
	return &UpdateRecordStatusReturn{}, nil
}

func (p Precompile) Records(
	ctx sdk.Context,
	input RecordsCall,
) (*RecordsReturn, error) {
	pgReq := input.Pagination.ToPageRequest()
	querySrv := keeper.NewQueryServerImpl(p.keeper)

	var registryId uint64
	var err error
	if input.Registry != "" {
		registryId, err = p.keeper.RegistryIdByName.Get(ctx, input.Registry)
		if err != nil {
			return nil, err
		}
	}

	rsp, err := querySrv.Records(ctx, &types.QueryRecordsRequest{
		Checksum:   input.Checksum,
		RegistryId: registryId,
		RecordId:   input.RecordId,
		Index:      input.Index,
		Pagination: pgReq,
	})
	if err != nil {
		return nil, err
	}

	abiRecords := make([]Record, len(rsp.Records))
	for i, rec := range rsp.Records {
		abiRecords[i] = ToABIRecord(*rec)
	}

	return &RecordsReturn{
		Records:    abiRecords,
		Pagination: invcmn.FromPageResponse(rsp.Pagination),
	}, nil
}

func (p Precompile) Registries(
	ctx sdk.Context,
	input RegistriesCall,
) (*RegistriesReturn, error) {
	pgReq := input.Pagination.ToPageRequest()
	querySrv := keeper.NewQueryServerImpl(p.keeper)

	rsp, err := querySrv.Registries(ctx, &types.QueryRegistriesRequest{
		RegistryId: input.RegistryId,
		Name:       input.Name,
		Pagination: pgReq,
	})
	if err != nil {
		return nil, err
	}

	abiRegistries := make([]Registry, len(rsp.Registries))
	for i, reg := range rsp.Registries {
		abiRegistries[i] = ToABIRegistry(*reg)
	}

	return &RegistriesReturn{
		Registries: abiRegistries,
		Pagination: invcmn.FromPageResponse(rsp.Pagination),
	}, nil
}

// GrantRole grants a role to an address for a registry or document
func (p Precompile) GrantRole(
	ctx sdk.Context,
	input GrantRoleCall,
	stateDB vm.StateDB,
	contract *vm.Contract,
) (*GrantRoleReturn, error) {
	admin, err := p.keeper.BytesToString(contract.Caller().Bytes())
	if err != nil {
		return nil, err
	}

	address, err := p.keeper.BytesToString(input.Account.Bytes())
	if err != nil {
		return nil, err
	}

	msg := &types.MsgGrantRole{
		Admin:      admin,
		RegistryId: input.RegistryId,
		Checksum:   input.Checksum,
		Address:    address,
		Role:       input.Role,
	}

	if _, err := p.msgServer.GrantRole(ctx, msg); err != nil {
		return nil, err
	}
	if err := p.emitGrantRoleEvent(ctx, stateDB, contract.Caller(), input.RegistryId, input.Checksum, input.Account, input.Role); err != nil {
		return nil, err
	}
	return &GrantRoleReturn{}, nil
}

// RevokeRole revokes a role from an address for a registry or document
func (p Precompile) RevokeRole(
	ctx sdk.Context,
	input RevokeRoleCall,
	stateDB vm.StateDB,
	contract *vm.Contract,
) (*RevokeRoleReturn, error) {
	if input.Role == "" {
		return nil, fmt.Errorf("role cannot be empty")
	}

	admin, err := p.keeper.BytesToString(contract.Caller().Bytes())
	if err != nil {
		return nil, err
	}

	address, err := p.keeper.BytesToString(input.Account.Bytes())
	if err != nil {
		return nil, err
	}

	msg := &types.MsgRevokeRole{
		Admin:      admin,
		RegistryId: input.RegistryId,
		Checksum:   input.Checksum,
		Address:    address,
		Role:       input.Role,
	}

	if _, err := p.msgServer.RevokeRole(ctx, msg); err != nil {
		return nil, err
	}
	if err := p.emitRevokeRoleEvent(ctx, stateDB, contract.Caller(), input.RegistryId, input.Checksum, input.Account, input.Role); err != nil {
		return nil, err
	}
	return &RevokeRoleReturn{}, nil
}
