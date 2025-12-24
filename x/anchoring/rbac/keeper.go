package rbac

import (
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

var (
	DefaultRoleAdmin = Role{}

	ErrMissingRole = errorsmod.Register("rbac", 100, "missing required role")
)

// rbac keeper implements a generic role-based access control,
// * role -> adminRole
// * role -> (account -> struct{})
type RBACKeeper struct {
	RoleAdmins  collections.Map[Role, []byte]
	RoleMembers collections.Map[collections.Pair[Role, sdk.AccAddress], []byte]
}

func NewRBACKeeper(sb *collections.SchemaBuilder, roleAdminPrefix collections.Prefix, roleMembersPrefix collections.Prefix) RBACKeeper {
	return RBACKeeper{
		RoleAdmins: collections.NewMap(
			sb,
			roleAdminPrefix,
			"role_admins",
			RoleCodec{},
			collections.BytesValue,
		),
		RoleMembers: collections.NewMap(
			sb,
			roleMembersPrefix,
			"role_members",
			collections.PairKeyCodec(
				RoleCodec{},
				sdk.AccAddressKey,
			),
			collections.BytesValue,
		),
	}
}

func (k *RBACKeeper) HasRole(ctx sdk.Context, role Role, addr sdk.AccAddress) (bool, error) {
	return k.RoleMembers.Has(ctx, collections.Join(role, addr))
}

func (k *RBACKeeper) GetRoleAdmin(ctx sdk.Context, role Role) (Role, error) {
	bz, err := k.RoleAdmins.Get(ctx, role)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return DefaultRoleAdmin, nil
		}
		return Role{}, err
	}
	return NewRole(bz), nil
}

func (k *RBACKeeper) SetRoleAdmin(ctx sdk.Context, role Role, adminRole Role) error {
	return k.RoleAdmins.Set(ctx, role, adminRole.Bytes())
}

func (k *RBACKeeper) GrantRole(ctx sdk.Context, role Role, addr sdk.AccAddress, granter sdk.AccAddress) error {
	adminRole, err := k.GetRoleAdmin(ctx, role)
	if err != nil {
		return err
	}
	hasAdminRole, err := k.HasRole(ctx, adminRole, granter)
	if err != nil {
		return err
	}
	if !hasAdminRole {
		return ErrMissingRole.Wrapf("account %s is missing role %s", granter.String(), common.Hash(adminRole).Hex())
	}
	return k.RoleMembers.Set(ctx, collections.Join(role, addr), []byte{})
}

func (k *RBACKeeper) RevokeRole(ctx sdk.Context, role Role, addr sdk.AccAddress, revoker sdk.AccAddress) error {
	adminRole, err := k.GetRoleAdmin(ctx, role)
	if err != nil {
		return err
	}
	hasAdminRole, err := k.HasRole(ctx, adminRole, revoker)
	if err != nil {
		return err
	}
	if !hasAdminRole {
		return ErrMissingRole.Wrapf("account %s is missing role %s", revoker.String(), common.Hash(adminRole).Hex())
	}
	return k.RoleMembers.Remove(ctx, collections.Join(role, addr))
}
