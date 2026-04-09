package keeper_test

import (
	"testing"

	"cosmossdk.io/core/address"
	appparams "github.com/MANTRA-Chain/nvnmchain/app/params"
	keepertest "github.com/MANTRA-Chain/nvnmchain/testutil/keeper"
	"github.com/MANTRA-Chain/nvnmchain/x/anchoring/keeper"
	"github.com/MANTRA-Chain/nvnmchain/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

var (
	registryAdminAddr = keepertest.TestSenderAddr
	moduleAdminAddr   = types.DefaultParams().Admin
)

func hasRegistryAdminRole(t *testing.T, k keeper.Keeper, ctx sdk.Context, addressCodec address.Codec, registryID uint64, addr string) bool {
	t.Helper()
	addrBz, err := addressCodec.StringToBytes(addr)
	require.NoError(t, err)
	hasRole, err := k.RBAC.HasRole(ctx, k.RegistryRole(registryID, keeper.RoleAdmin), addrBz)
	require.NoError(t, err)
	return hasRole
}

func TestMsgRevokeRole_DisallowRevokingLastRegistryAdmin(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	var err error

	admin1 := registryAdminAddr
	admin2 := moduleAdminAddr

	registryID := keepertest.MustCreateAnchoringRegistry(t, k, ctx, admin1, "reg-last-admin")

	// Sole admin cannot revoke their own admin role (would leave 0 admins).
	_, err = ms.RevokeRole(ctx, &types.MsgRevokeRole{
		Admin:      admin1,
		Address:    admin1,
		RegistryId: registryID,
		Checksum:   "",
		Role:       keeper.RoleAdmin,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot revoke the last registry admin")

	// Grant a replacement admin, then self-revocation should be allowed.
	_, err = ms.GrantRole(ctx, &types.MsgGrantRole{
		Admin:      admin1,
		Address:    admin2,
		RegistryId: registryID,
		Checksum:   "",
		Role:       keeper.RoleAdmin,
	})
	require.NoError(t, err)

	_, err = ms.RevokeRole(ctx, &types.MsgRevokeRole{
		Admin:      admin1,
		Address:    admin1,
		RegistryId: registryID,
		Checksum:   "",
		Role:       keeper.RoleAdmin,
	})
	require.NoError(t, err)
}

func TestMsgRevokeRole_AllowModuleAdminEmergencyLastAdminRevokeAndRecovery(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, addressCodec := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	var err error
	registryID := keepertest.MustCreateAnchoringRegistry(t, k, ctx, registryAdminAddr, "reg-emergency-admin-recovery")
	require.True(t, hasRegistryAdminRole(t, k, ctx, addressCodec, registryID, registryAdminAddr))

	_, err = ms.RevokeRole(ctx, &types.MsgRevokeRole{
		Admin:      moduleAdminAddr,
		Address:    registryAdminAddr,
		RegistryId: registryID,
		Checksum:   "",
		Role:       keeper.RoleAdmin,
	})
	require.NoError(t, err)
	require.False(t, hasRegistryAdminRole(t, k, ctx, addressCodec, registryID, registryAdminAddr))

	_, err = ms.GrantRole(ctx, &types.MsgGrantRole{
		Admin:      moduleAdminAddr,
		Address:    moduleAdminAddr,
		RegistryId: registryID,
		Checksum:   "",
		Role:       keeper.RoleAdmin,
	})
	require.NoError(t, err)
	require.True(t, hasRegistryAdminRole(t, k, ctx, addressCodec, registryID, moduleAdminAddr))
}

func TestMsgRevokeRole_ValidationErrors(t *testing.T) {
	testCases := []struct {
		name            string
		setupReq        func(t *testing.T, k keeper.Keeper, ctx sdk.Context) *types.MsgRevokeRole
		wantErrContains string
	}{
		{
			name: "registry not found",
			setupReq: func(t *testing.T, _ keeper.Keeper, _ sdk.Context) *types.MsgRevokeRole {
				t.Helper()
				return &types.MsgRevokeRole{
					Admin:      registryAdminAddr,
					Address:    registryAdminAddr,
					RegistryId: 999,
					Checksum:   "",
					Role:       keeper.RoleAdmin,
				}
			},
			wantErrContains: "registry 999 does not exist",
		},
		{
			name: "record checksum missing in registry",
			setupReq: func(t *testing.T, k keeper.Keeper, ctx sdk.Context) *types.MsgRevokeRole {
				t.Helper()
				registryID := keepertest.MustCreateAnchoringRegistry(t, k, ctx, registryAdminAddr, "reg-record-check")
				keepertest.MustAddAnchoringRecord(t, k, ctx, registryAdminAddr, "reg-record-check", "present-checksum", "sha256")
				return &types.MsgRevokeRole{
					Admin:      registryAdminAddr,
					Address:    moduleAdminAddr,
					RegistryId: registryID,
					Checksum:   "missing-checksum",
					Role:       keeper.RoleEditor,
				}
			},
			wantErrContains: "record with checksum missing-checksum does not exist in registry",
		},
		{
			name: "registry zero with checksum",
			setupReq: func(t *testing.T, _ keeper.Keeper, _ sdk.Context) *types.MsgRevokeRole {
				t.Helper()
				return &types.MsgRevokeRole{
					Admin:      registryAdminAddr,
					Address:    moduleAdminAddr,
					RegistryId: 0,
					Checksum:   "any-checksum",
					Role:       keeper.RoleEditor,
				}
			},
			wantErrContains: "registry 0 does not exist",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appparams.SetAddressPrefixes()
			k, ctx, _ := keepertest.AnchoringKeeper(t)
			ms := keeper.NewMsgServerImpl(k)

			req := tc.setupReq(t, k, ctx)
			_, err := ms.RevokeRole(ctx, req)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErrContains)
		})
	}
}
