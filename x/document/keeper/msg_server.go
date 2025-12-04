package keeper

import (
	"context"

	"github.com/MANTRA-Chain/inveniam/x/document/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) AddDocument(goCtx context.Context, req *types.MsgAddDocument) (*types.MsgAddDocumentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender, err := k.addressCodec.StringToBytes(req.Sender)
	if err != nil {
		return nil, err
	}

	if err := k.AddDocumentInner(ctx, sender, *req.Document); err != nil {
		return nil, err
	}

	return &types.MsgAddDocumentResponse{}, nil
}

func (k msgServer) RemoveDocument(goCtx context.Context, req *types.MsgRemoveDocument) (*types.MsgRemoveDocumentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	sender, err := k.addressCodec.StringToBytes(req.Sender)
	if err != nil {
		return nil, err
	}
	if err := k.RemoveDocumentInner(ctx, sender, req.Denom, req.Index); err != nil {
		return nil, err
	}
	return &types.MsgRemoveDocumentResponse{}, nil
}
