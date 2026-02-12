package keeper_test

import (
	"testing"

	appparams "github.com/MANTRA-Chain/inveniam/app/params"
	keepertest "github.com/MANTRA-Chain/inveniam/testutil/keeper"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/keeper"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/types"
	"github.com/stretchr/testify/require"
)

func TestMsgRevokeRole_DisallowRevokingLastRegistryAdmin(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	require.NoError(t, k.RegistryCount.Set(ctx, 0))

	admin1 := "inveniam1axznhnm82lah8qqvp9hxdad49yx3s5dcmnx072"
	admin2 := "inveniam15m77x4pe6w9vtpuqm22qxu0ds7vn4ehz80mwh8"

	addRegRes, err := ms.AddRegistry(ctx, &types.MsgAddRegistry{
		Sender:      admin1,
		Name:        "reg-last-admin",
		Description: "",
		Metadata:    "{}",
	})
	require.NoError(t, err)

	// Sole admin cannot revoke their own admin role (would leave 0 admins).
	_, err = ms.RevokeRole(ctx, &types.MsgRevokeRole{
		Admin:      admin1,
		Address:    admin1,
		RegistryId: addRegRes.RegistryId,
		Checksum:   "",
		Role:       keeper.RoleAdmin,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot revoke the last registry admin")

	// Grant a replacement admin, then self-revocation should be allowed.
	_, err = ms.GrantRole(ctx, &types.MsgGrantRole{
		Admin:      admin1,
		Address:    admin2,
		RegistryId: addRegRes.RegistryId,
		Checksum:   "",
		Role:       keeper.RoleAdmin,
	})
	require.NoError(t, err)

	_, err = ms.RevokeRole(ctx, &types.MsgRevokeRole{
		Admin:      admin1,
		Address:    admin1,
		RegistryId: addRegRes.RegistryId,
		Checksum:   "",
		Role:       keeper.RoleAdmin,
	})
	require.NoError(t, err)
}
