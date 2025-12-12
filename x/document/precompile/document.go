package precompile

import (
	"fmt"

	cmn "github.com/cosmos/evm/precompiles/common"

	storetypes "cosmossdk.io/store/types"
	invcmn "github.com/MANTRA-Chain/inveniam/precompiles"
	"github.com/MANTRA-Chain/inveniam/x/document/keeper"
	"github.com/MANTRA-Chain/inveniam/x/document/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

//go:generate go run github.com/yihuang/go-abi/cmd -var=HumanABI -output document.abi.go --external-tuples PageRequest=cmn.PageRequest,PageResponse=cmn.PageResponse --imports cmn=github.com/MANTRA-Chain/inveniam/precompiles
var HumanABI = []string{
	"struct Document {string name; string denom; string uri; string checksum; string checksumAlgo; string timestamp; string figi; string individualId}",
	"struct PageRequest { bytes key; uint64 offset; uint64 limit; bool countTotal; bool reverse; }",
	"struct PageResponse { bytes nextKey; uint64 total; }",

	"function addDocument(Document document) returns ()",
	"function removeDocument(string denom, uint64 index) returns ()",
	"function documents(string denom, uint64 index, PageRequest pagination) returns (Document[] documents, PageResponse pagination)",
}

var DocumentPrecompileAddress = common.HexToAddress("0x0000000000000000000000000000000000000a00")

var _ vm.PrecompiledContract = &Precompile{}

// Precompile defines the bank precompile
type Precompile struct {
	cmn.Precompile
	keeper keeper.Keeper
}

// NewPrecompile creates a new bank Precompile instance implementing the
// PrecompiledContract interface.
func NewPrecompile(
	keeper keeper.Keeper,
) (*Precompile, error) {
	// NOTE: we set an empty gas configuration to avoid extra gas costs
	// during the run execution
	p := &Precompile{
		Precompile: cmn.Precompile{
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
		},
		keeper: keeper,
	}

	// SetAddress defines the address of the bank compile contract.
	p.SetAddress(DocumentPrecompileAddress)

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
		return p.Execute(ctx, evm.StateDB, contract, readOnly)
	})
}

func (p Precompile) Execute(ctx sdk.Context, stateDB vm.StateDB, contract *vm.Contract, readOnly bool) ([]byte, error) {
	methodID, input, err := invcmn.ParseMethod(contract.Input, readOnly, p.IsTransactionID)
	if err != nil {
		return nil, err
	}

	switch methodID {
	case AddDocumentID:
		return invcmn.RunWithStateDB(ctx, p.AddDocument, input, stateDB, contract)
	case RemoveDocumentID:
		return invcmn.RunWithStateDB(ctx, p.RemoveDocument, input, stateDB, contract)
	case DocumentsID:
		return invcmn.Run(ctx, p.Documents, input)
	}

	return nil, fmt.Errorf(invcmn.ErrUnknownMethodID, methodID)
}

// IsTransaction checks if the given method name corresponds to a transaction or query.
// It returns false since all bank methods are queries.
func (Precompile) IsTransactionID(methodID uint32) bool {
	return methodID == AddDocumentID || methodID == RemoveDocumentID
}

func (p Precompile) AddDocument(
	ctx sdk.Context,
	input AddDocumentCall,
	_ vm.StateDB,
	contract *vm.Contract,
) (*AddDocumentReturn, error) {
	doc := FromABIDocument(input.Document)
	if err := p.keeper.AddDocumentInner(ctx, contract.Caller().Bytes(), doc); err != nil {
		return nil, err
	}
	return &AddDocumentReturn{}, nil
}

func (p Precompile) RemoveDocument(
	ctx sdk.Context,
	input RemoveDocumentCall,
	_ vm.StateDB,
	contract *vm.Contract,
) (*RemoveDocumentReturn, error) {
	if err := p.keeper.RemoveDocumentInner(ctx, contract.Caller().Bytes(), input.Denom, input.Index); err != nil {
		return nil, err
	}
	return &RemoveDocumentReturn{}, nil
}

func (p Precompile) Documents(
	ctx sdk.Context,
	input DocumentsCall,
) (*DocumentsReturn, error) {
	pgReq := input.Pagination.ToPageRequest()
	querySrv := keeper.NewQueryServerImpl(p.keeper)
	rsp, err := querySrv.Documents(ctx, &types.QueryDocumentsRequest{
		Denom:      input.Denom,
		Index:      input.Index,
		Pagination: pgReq,
	})
	if err != nil {
		return nil, err
	}

	abiDocs := make([]Document, len(rsp.Documents))
	for i, doc := range rsp.Documents {
		abiDocs[i] = ToABIDocument(*doc)
	}

	return &DocumentsReturn{
		Documents:  abiDocs,
		Pagination: invcmn.FromPageResponse(rsp.Pagination),
	}, nil
}
