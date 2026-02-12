package keeper

import (
	"context"

	"github.com/MANTRA-Chain/inveniam/x/anchoring/rbac"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/types"
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

// isRecordRole returns true if the checksum indicates a record-specific role
func isRecordRole(checksum string) bool {
	return checksum != ""
}

func (k msgServer) AddRegistry(goCtx context.Context, req *types.MsgAddRegistry) (*types.MsgAddRegistryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	sender, err := k.addressCodec.StringToBytes(req.Sender)
	if err != nil {
		return nil, err
	}
	registryId, err := k.Keeper.AddRegistry(ctx, sender, req.Name, req.Description, req.Metadata)
	if err != nil {
		return nil, err
	}

	return &types.MsgAddRegistryResponse{RegistryId: registryId}, nil
}

func (k msgServer) AddRecord(goCtx context.Context, req *types.MsgAddRecord) (*types.MsgAddRecordResponse, error) {
	if req == nil {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("empty request")
	}
	if req.Record == nil {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("record cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	sender, err := k.addressCodec.StringToBytes(req.Sender)
	if err != nil {
		return nil, err
	}
	// Only admin or editor can add
	recordId, err := k.Keeper.AddRecord(ctx, sender, *req.Record)
	if err != nil {
		return nil, err
	}
	return &types.MsgAddRecordResponse{RecordId: recordId}, nil
}

func (k msgServer) UpdateRecordStatus(goCtx context.Context, req *types.MsgUpdateRecordStatus) (*types.MsgUpdateRecordStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	sender, err := k.addressCodec.StringToBytes(req.Editor)
	if err != nil {
		return nil, err
	}
	// Only admin or editor can update
	if err := k.Keeper.UpdateRecordStatus(ctx, sender, req.RegistryId, req.RecordId, req.Index, req.Status); err != nil {
		return nil, err
	}
	return &types.MsgUpdateRecordStatusResponse{}, nil
}

// GrantRole allows an admin to assign a role to an address
func (k msgServer) GrantRole(goCtx context.Context, req *types.MsgGrantRole) (*types.MsgGrantRoleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	granter, err := k.addressCodec.StringToBytes(req.Admin)
	if err != nil {
		return nil, err
	}
	grantee, err := k.addressCodec.StringToBytes(req.Address)
	if err != nil {
		return nil, err
	}

	var role rbac.Role
	var adminRole rbac.Role

	if isRecordRole(req.Checksum) {
		role = k.Keeper.RecordRole(req.Checksum, req.Role)
	} else {
		role = k.Keeper.RegistryRole(req.RegistryId, req.Role)
	}
	adminRole = k.Keeper.RegistryRole(req.RegistryId, RoleAdmin)
	if err := k.Keeper.RBAC.SetRoleAdmin(ctx, role, adminRole); err != nil {
		return nil, err
	}

	if err := k.Keeper.RBAC.GrantRole(ctx, role, grantee, granter); err != nil {
		return nil, err
	}

	return &types.MsgGrantRoleResponse{}, nil
}

// RevokeRole allows an admin to remove a role from an address
func (k msgServer) RevokeRole(goCtx context.Context, req *types.MsgRevokeRole) (*types.MsgRevokeRoleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	revoker, err := k.addressCodec.StringToBytes(req.Admin)
	if err != nil {
		return nil, err
	}
	revokee, err := k.addressCodec.StringToBytes(req.Address)
	if err != nil {
		return nil, err
	}

	// revoke common roles (editor, then admin) when no specific role provided
	rolesToTry := []string{req.Role}
	if req.Role == "" {
		rolesToTry = []string{RoleEditor, RoleAdmin}
	}

	var lastErr error
	for _, roleStr := range rolesToTry {
		var role rbac.Role
		if isRecordRole(req.Checksum) {
			role = k.Keeper.RecordRole(req.Checksum, roleStr)
		} else {
			role = k.Keeper.RegistryRole(req.RegistryId, roleStr)
		}
		hasRole, err := k.Keeper.RBAC.HasRole(ctx, role, revokee)
		if err != nil {
			lastErr = err
			continue
		}
		if !hasRole {
			continue
		}

		// Prevent revoking the last registry-level admin, which would permanently lock the registry.
		if !isRecordRole(req.Checksum) && roleStr == RoleAdmin {
			adminRole := k.Keeper.RegistryRole(req.RegistryId, RoleAdmin)
			isSoleAdmin, err := k.Keeper.RBAC.IsSoleAdmin(ctx, adminRole)
			if err != nil {
				lastErr = err
				continue
			}
			if isSoleAdmin {
				return nil, sdkerrors.ErrInvalidRequest.Wrap("cannot revoke the last registry admin")
			}
		}

		if err := k.Keeper.RBAC.RevokeRole(ctx, role, revokee, revoker); err != nil {
			lastErr = err
			continue
		}

		return &types.MsgRevokeRoleResponse{}, nil
	}

	if req.Role == "" && lastErr == nil {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("address has no roles to revoke")
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return &types.MsgRevokeRoleResponse{}, nil
}
