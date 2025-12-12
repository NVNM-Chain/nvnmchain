package keeper

import (
	"context"

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
	sender, err := k.addressCodec.StringToBytes(req.Sender)
	if err != nil {
		return nil, err
	}
	// Only admin or editor can add
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
	// Only admin or editor can remove
	if err := k.RemoveDocumentInner(ctx, sender, req.Denom, req.Index); err != nil {
		return nil, err
	}
	return &types.MsgRemoveDocumentResponse{}, nil
}

// GrantRole allows an admin to assign a role to an address
func (k msgServer) GrantRole(goCtx context.Context, req *types.MsgGrantRole) (*types.MsgGrantRoleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	// Only admin can grant roles
	if req.Admin != params.Admin {
		return nil, sdkerrors.ErrUnauthorized
	}
	err = k.Roles.Set(ctx, req.Address, req.Role)
	if err != nil {
		return nil, err
	}
	return &types.MsgGrantRoleResponse{}, nil
}

// RevokeRole allows an admin to remove a role from an address
func (k msgServer) RevokeRole(goCtx context.Context, req *types.MsgRevokeRole) (*types.MsgRevokeRoleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	// Only admin can revoke roles
	if req.Admin != params.Admin {
		return nil, sdkerrors.ErrUnauthorized
	}
	role, err := k.Roles.Get(ctx, req.Address)
	if err != nil {
		return nil, err
	}
	if role != req.Role {
		return nil, sdkerrors.ErrUnauthorized
	}
	err = k.Roles.Remove(ctx, req.Address)
	if err != nil {
		return nil, err
	}
	return &types.MsgRevokeRoleResponse{}, nil
}
