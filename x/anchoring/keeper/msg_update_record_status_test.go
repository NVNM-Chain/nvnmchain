package keeper_test

import (
	"strings"
	"testing"

	"cosmossdk.io/collections"
	appparams "github.com/MANTRA-Chain/inveniam/app/params"
	keepertest "github.com/MANTRA-Chain/inveniam/testutil/keeper"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/keeper"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/types"
	"github.com/stretchr/testify/require"
)

func TestMsgUpdateRecordStatus_ValidateBasic_OversizedStatus(t *testing.T) {
	appparams.SetAddressPrefixes()
	sender := keepertest.TestSenderAddr
	testCases := []struct {
		name            string
		msg             types.MsgUpdateRecordStatus
		wantErrContains string
	}{
		{
			name: "oversized status",
			msg: types.MsgUpdateRecordStatus{
				Editor:     sender,
				RegistryId: 1,
				RecordId:   1,
				Index:      1,
				Status:     strings.Repeat("s", types.MaxRecordStatusLen+1),
			},
			wantErrContains: "status exceeds max length",
		},
		{
			name: "empty status",
			msg: types.MsgUpdateRecordStatus{
				Editor:     sender,
				RegistryId: 1,
				RecordId:   1,
				Index:      1,
				Status:     "",
			},
			wantErrContains: "status cannot be empty",
		},
		{
			name: "invalid editor",
			msg: types.MsgUpdateRecordStatus{
				Editor:     "not-an-address",
				RegistryId: 1,
				RecordId:   1,
				Index:      1,
				Status:     "redacted",
			},
			wantErrContains: "invalid Editor address",
		},
		{
			name: "missing record id",
			msg: types.MsgUpdateRecordStatus{
				Editor:     sender,
				RegistryId: 1,
				RecordId:   0,
				Index:      1,
				Status:     "redacted",
			},
			wantErrContains: "record ID cannot be zero",
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

func TestMsgUpdateRecordStatus_MsgServer_Cases(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	sender := keepertest.TestSenderAddr
	registryID := keepertest.MustCreateAnchoringRegistry(t, k, ctx, sender, "reg1")
	recordID := keepertest.MustAddAnchoringRecord(t, k, ctx, sender, "reg1", "deadbeef", "sha256")
	testCases := []struct {
		name            string
		req             *types.MsgUpdateRecordStatus
		wantErrContains string
		wantStatus      string
	}{
		{
			name: "reject oversized status",
			req: &types.MsgUpdateRecordStatus{
				Editor:     sender,
				RegistryId: registryID,
				RecordId:   recordID,
				Index:      1,
				Status:     strings.Repeat("s", types.MaxRecordStatusLen+1),
			},
			wantErrContains: "status exceeds max length",
		},
		{
			name: "accept valid status and persist",
			req: &types.MsgUpdateRecordStatus{
				Editor:     sender,
				RegistryId: registryID,
				RecordId:   recordID,
				Index:      1,
				Status:     "redacted",
			},
			wantStatus: "redacted",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.UpdateRecordStatus(ctx, tc.req)
			if tc.wantErrContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrContains)
				return
			}
			require.NoError(t, err)

			record, err := k.Records.Get(ctx, collections.Join3(registryID, recordID, uint64(1)))
			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, record.Status)
		})
	}
}
