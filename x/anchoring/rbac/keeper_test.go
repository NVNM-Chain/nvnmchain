package rbac_test

import (
	"testing"

	appparams "github.com/MANTRA-Chain/nvnmchain/app/params"
	keepertest "github.com/MANTRA-Chain/nvnmchain/testutil/keeper"
	"github.com/MANTRA-Chain/nvnmchain/testutil/sample"
	"github.com/MANTRA-Chain/nvnmchain/x/anchoring/keeper"
	"github.com/MANTRA-Chain/nvnmchain/x/anchoring/rbac"
	"github.com/MANTRA-Chain/nvnmchain/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestRBACKeeper_RoleAdminNotConfigured(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, addrCodec := keepertest.AnchoringKeeper(t)

	caller, err := addrCodec.StringToBytes(keepertest.TestSenderAddr)
	require.NoError(t, err)
	target, err := sdk.AccAddressFromBech32(sample.AccAddress())
	require.NoError(t, err)

	role := k.RegistryRole(42, keeper.RoleEditor)

	// GetRoleAdmin on an unconfigured role returns DefaultRoleAdmin + wrapped error.
	adminRole, err := k.RBAC.GetRoleAdmin(ctx, role)
	require.ErrorIs(t, err, types.ErrRoleAdminNotConfigured)
	require.Contains(t, err.Error(), common.Hash(role).Hex())
	require.Equal(t, rbac.DefaultRoleAdmin, adminRole)

	// Grant and Revoke surface the same error.
	require.ErrorIs(t, k.RBAC.GrantRole(ctx, role, target, caller), types.ErrRoleAdminNotConfigured)
	require.ErrorIs(t, k.RBAC.RevokeRole(ctx, role, target, caller), types.ErrRoleAdminNotConfigured)

	// After SetRoleAdmin, GetRoleAdmin returns the configured admin with no error.
	configured := k.RegistryRole(42, keeper.RoleAdmin)
	require.NoError(t, k.RBAC.SetRoleAdmin(ctx, role, configured))
	got, err := k.RBAC.GetRoleAdmin(ctx, role)
	require.NoError(t, err)
	require.Equal(t, configured, got)
}
