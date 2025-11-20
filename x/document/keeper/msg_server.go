package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"github.com/MANTRA-Chain/inveniam/x/document/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
	tokenAdmin, err := k.tokenFactoryKeeper.GetAuthorityMetadata(ctx, req.Document.Denom)
	if err != nil {
		return nil, err
	}
	if tokenAdmin.Admin != req.Sender {
		params, err := k.Params.Get(ctx)
		if err != nil {
			return nil, err
		}
		if params.Admin != req.Sender {
			return nil, sdkerrors.ErrUnauthorized
		}
	}

	if err = k.setDocument(ctx, *req.Document); err != nil {
		return nil, err
	}

	return &types.MsgAddDocumentResponse{}, nil
}

func (k msgServer) RemoveDocument(goCtx context.Context, req *types.MsgRemoveDocument) (*types.MsgRemoveDocumentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	tokenAdmin, err := k.tokenFactoryKeeper.GetAuthorityMetadata(ctx, req.Denom)
	if err != nil {
		return nil, err
	}
	if tokenAdmin.Admin != req.Sender {
		params, err := k.Params.Get(ctx)
		if err != nil {
			return nil, err
		}
		if params.Admin != req.Sender {
			return nil, sdkerrors.ErrUnauthorized
		}
	}

	err = k.Documents.Remove(ctx, collections.Join(req.Denom, req.Index))
	if err != nil {
		return nil, err
	}

	return &types.MsgRemoveDocumentResponse{}, nil
}
