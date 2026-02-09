package keeper_test

import (
	"testing"

	appparams "github.com/MANTRA-Chain/inveniam/app/params"
	keepertest "github.com/MANTRA-Chain/inveniam/testutil/keeper"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/keeper"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/types"
	"github.com/stretchr/testify/require"
)

func TestMsgAddRecord_NilRequest(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.AddRecord(ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty request")
}

func TestMsgAddRecord_NilRecord(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.AddRecord(ctx, &types.MsgAddRecord{Sender: "inveniam1axznhnm82lah8qqvp9hxdad49yx3s5dcmnx072"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "record cannot be nil")
}

func TestMsgAddRecord_ValidateBasic_NilRecord(t *testing.T) {
	appparams.SetAddressPrefixes()
	msg := types.MsgAddRecord{Sender: "inveniam1axznhnm82lah8qqvp9hxdad49yx3s5dcmnx072", Record: nil}
	err := msg.ValidateBasic()
	require.Error(t, err)
	require.Contains(t, err.Error(), "record cannot be nil")
}
