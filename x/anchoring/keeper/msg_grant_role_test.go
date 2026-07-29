package keeper_test

import (
	"testing"

	appparams "github.com/NVNM-Chain/nvnmchain/app/params"
	keepertest "github.com/NVNM-Chain/nvnmchain/testutil/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	"github.com/stretchr/testify/require"
)

// TestMsgGrantRole_ChangeRegistryAdmin exercises the only supported way to "change" a
// registry's admin: grant the admin role to the new address, then revoke it from the old one.
func TestMsgGrantRole_ChangeRegistryAdmin(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, addressCodec := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	oldAdmin := registryAdminAddr
	newAdmin := moduleAdminAddr

	registryID := keepertest.MustCreateAnchoringRegistry(t, k, ctx, oldAdmin, "reg-change-admin")
	require.True(t, hasRegistryAdminRole(t, k, ctx, addressCodec, registryID, oldAdmin))
	require.False(t, hasRegistryAdminRole(t, k, ctx, addressCodec, registryID, newAdmin))

	_, err := ms.GrantRole(ctx, &types.MsgGrantRole{
		Admin:      oldAdmin,
		Address:    newAdmin,
		RegistryId: registryID,
		Checksum:   "",
		Role:       keeper.RoleAdmin,
	})
	require.NoError(t, err)
	require.True(t, hasRegistryAdminRole(t, k, ctx, addressCodec, registryID, newAdmin))

	_, err = ms.RevokeRole(ctx, &types.MsgRevokeRole{
		Admin:      newAdmin,
		Address:    oldAdmin,
		RegistryId: registryID,
		Checksum:   "",
		Role:       keeper.RoleAdmin,
	})
	require.NoError(t, err)
	require.False(t, hasRegistryAdminRole(t, k, ctx, addressCodec, registryID, oldAdmin))
	require.True(t, hasRegistryAdminRole(t, k, ctx, addressCodec, registryID, newAdmin))
}

func TestMsgGrantRole_NonAdminCannotGrantRole(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, addressCodec := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	registryID := keepertest.MustCreateAnchoringRegistry(t, k, ctx, registryAdminAddr, "reg-non-admin-grant")

	// moduleAdminAddr holds no role on this registry, so it cannot grant one — module-admin
	// break-glass only applies to granting the registry-admin role, not editor.
	_, err := ms.GrantRole(ctx, &types.MsgGrantRole{
		Admin:      moduleAdminAddr,
		Address:    moduleAdminAddr,
		RegistryId: registryID,
		Checksum:   "",
		Role:       keeper.RoleEditor,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "is missing role")
	require.False(t, hasRegistryEditorRole(t, k, ctx, addressCodec, registryID, moduleAdminAddr))
}

func TestMsgGrantRole_ModuleAdminBreakGlassGrantsRegistryAdmin(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, addressCodec := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	registryID := keepertest.MustCreateAnchoringRegistry(t, k, ctx, registryAdminAddr, "reg-break-glass-grant")

	// Module admin holds no role on the registry but can still grant themselves registry-admin
	// directly, bypassing the normal admin-role check.
	_, err := ms.GrantRole(ctx, &types.MsgGrantRole{
		Admin:      moduleAdminAddr,
		Address:    moduleAdminAddr,
		RegistryId: registryID,
		Checksum:   "",
		Role:       keeper.RoleAdmin,
	})
	require.NoError(t, err)
	require.True(t, hasRegistryAdminRole(t, k, ctx, addressCodec, registryID, moduleAdminAddr))
	require.True(t, hasRegistryAdminRole(t, k, ctx, addressCodec, registryID, registryAdminAddr))
}

func TestMsgGrantRole_RecordScopedRoleRequiresRegistryAdmin(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, addressCodec := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	registryID := keepertest.MustCreateAnchoringRegistry(t, k, ctx, registryAdminAddr, "reg-record-scoped")
	keepertest.MustAddAnchoringRecord(t, k, ctx, registryAdminAddr, registryID, "checksum-1", "sha256")

	_, err := ms.GrantRole(ctx, &types.MsgGrantRole{
		Admin:      registryAdminAddr,
		Address:    moduleAdminAddr,
		RegistryId: registryID,
		Checksum:   "checksum-1",
		Role:       keeper.RoleEditor,
	})
	require.NoError(t, err)

	granteeBz, err := addressCodec.StringToBytes(moduleAdminAddr)
	require.NoError(t, err)
	hasRole, err := k.RBAC.HasRole(ctx, k.RecordRole(registryID, "checksum-1", keeper.RoleEditor), granteeBz)
	require.NoError(t, err)
	require.True(t, hasRole)
}

func TestMsgGrantRole_ValidationErrors(t *testing.T) {
	testCases := []struct {
		name            string
		req             *types.MsgGrantRole
		wantErrContains string
	}{
		{
			name: "admin address empty",
			req: &types.MsgGrantRole{
				Admin:      "",
				Address:    moduleAdminAddr,
				RegistryId: 1,
				Role:       keeper.RoleAdmin,
			},
			wantErrContains: "admin address cannot be empty",
		},
		{
			name: "grantee address empty",
			req: &types.MsgGrantRole{
				Admin:      registryAdminAddr,
				Address:    "",
				RegistryId: 1,
				Role:       keeper.RoleAdmin,
			},
			wantErrContains: "grantee address cannot be empty",
		},
		{
			name: "registry id cannot be zero",
			req: &types.MsgGrantRole{
				Admin:      registryAdminAddr,
				Address:    moduleAdminAddr,
				RegistryId: 0,
				Role:       keeper.RoleAdmin,
			},
			wantErrContains: "registry ID cannot be zero",
		},
		{
			name: "role cannot be empty",
			req: &types.MsgGrantRole{
				Admin:      registryAdminAddr,
				Address:    moduleAdminAddr,
				RegistryId: 1,
				Role:       "",
			},
			wantErrContains: "role cannot be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appparams.SetAddressPrefixes()
			k, ctx, _ := keepertest.AnchoringKeeper(t)
			ms := keeper.NewMsgServerImpl(k)

			_, err := ms.GrantRole(ctx, tc.req)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErrContains)
		})
	}
}

func TestMsgGrantRole_RegistryNotFound(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.GrantRole(ctx, &types.MsgGrantRole{
		Admin:      registryAdminAddr,
		Address:    moduleAdminAddr,
		RegistryId: 999,
		Role:       keeper.RoleAdmin,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "registry 999 does not exist")
}
