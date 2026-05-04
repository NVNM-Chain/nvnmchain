package keeper_test

import (
	"strings"
	"testing"

	appparams "github.com/NVNM-Chain/nvnmchain/app/params"
	keepertest "github.com/NVNM-Chain/nvnmchain/testutil/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	"github.com/stretchr/testify/require"
)

func TestMsgAddRegistry_ValidateBasic(t *testing.T) {
	appparams.SetAddressPrefixes()
	sender := keepertest.TestSenderAddr
	testCases := []struct {
		name            string
		msg             types.MsgAddRegistry
		wantErrContains string
	}{
		{
			name:            "empty sender",
			msg:             types.MsgAddRegistry{Sender: "", Name: "reg1", Description: "test", Metadata: "{}"},
			wantErrContains: "sender address cannot be empty",
		},
		{
			name:            "invalid sender",
			msg:             types.MsgAddRegistry{Sender: "not-an-address", Name: "reg1", Description: "test", Metadata: "{}"},
			wantErrContains: "invalid sender address",
		},
		{
			name:            "empty name",
			msg:             types.MsgAddRegistry{Sender: sender, Name: "", Description: "test", Metadata: "{}"},
			wantErrContains: "name cannot be empty",
		},
		{
			name: "oversized name",
			msg: types.MsgAddRegistry{
				Sender:      sender,
				Name:        strings.Repeat("a", types.MaxRegistryNameLen+1),
				Description: "test",
				Metadata:    "{}",
			},
			wantErrContains: "name exceeds max length",
		},
		{
			name: "oversized description",
			msg: types.MsgAddRegistry{
				Sender:      sender,
				Name:        "reg1",
				Description: strings.Repeat("a", types.MaxRegistryDescriptionLen+1),
				Metadata:    "{}",
			},
			wantErrContains: "description exceeds max length",
		},
		{
			name: "oversized metadata",
			msg: types.MsgAddRegistry{
				Sender:      sender,
				Name:        "reg1",
				Description: "test",
				Metadata:    strings.Repeat("a", types.MaxRegistryMetadataLen+1),
			},
			wantErrContains: "metadata exceeds max length",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErrContains)
		})
	}
}

func TestMsgAddRegistry_OversizedName_RejectedByMsgServerPath(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	sender := keepertest.TestSenderAddr
	_, err := ms.AddRegistry(ctx, &types.MsgAddRegistry{
		Sender:      sender,
		Name:        strings.Repeat("a", types.MaxRegistryNameLen+1),
		Description: "test",
		Metadata:    "{\"k\":\"v\"}",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "name exceeds max length")
}
