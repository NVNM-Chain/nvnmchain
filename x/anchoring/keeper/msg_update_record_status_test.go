package keeper_test

import (
	"strings"
	"testing"

	"cosmossdk.io/collections"
	appparams "github.com/MANTRA-Chain/inveniam/app/params"
	keepertest "github.com/MANTRA-Chain/inveniam/testutil/keeper"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/keeper"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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

func TestMsgUpdateRecordStatus_NilRequest(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.UpdateRecordStatus(ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty request")
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

func TestMsgUpdateRecordStatus_RecordRoleDoesNotBleedAcrossRegistries(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	const (
		checksum       = "same-checksum"
		victimRegistry = "victim-reg"
		attackerReg    = "attacker-reg"
	)

	victimAdmin := keepertest.TestSenderAddr
	attacker := types.DefaultParams().Admin
	victimRegistryID := keepertest.MustCreateAnchoringRegistry(t, k, ctx, victimAdmin, victimRegistry)
	victimRecordID := keepertest.MustAddAnchoringRecord(t, k, ctx, victimAdmin, victimRegistry, checksum, "sha256")

	updateVictimStatus := func(status string) error {
		_, err := ms.UpdateRecordStatus(ctx, &types.MsgUpdateRecordStatus{
			Editor:     attacker,
			RegistryId: victimRegistryID,
			RecordId:   victimRecordID,
			Index:      1,
			Status:     status,
		})
		return err
	}

	assertUnauthorized := func(err error) {
		require.Error(t, err)
		require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)
	}

	// Sanity: attacker cannot mutate victim status before any role assignment.
	assertUnauthorized(updateVictimStatus("unauthorized-pre"))

	attackerRegistryID := keepertest.MustCreateAnchoringRegistry(t, k, ctx, attacker, attackerReg)
	keepertest.MustAddAnchoringRecord(t, k, ctx, attacker, attackerReg, checksum, "sha256")

	// Attacker can grant themselves record-level editor in attacker registry.
	_, err := ms.GrantRole(ctx, &types.MsgGrantRole{
		Admin:      attacker,
		Address:    attacker,
		RegistryId: attackerRegistryID,
		Checksum:   checksum,
		Role:       keeper.RoleEditor,
	})
	require.NoError(t, err)

	// Regression check: role from attacker registry must not authorize victim registry mutation.
	assertUnauthorized(updateVictimStatus("tampered-by-attacker"))

	victimRecord, err := k.Records.Get(ctx, collections.Join3(victimRegistryID, victimRecordID, uint64(1)))
	require.NoError(t, err)
	require.Equal(t, "active", victimRecord.Status)
}

func TestRecordRole_NoAliasWhenChecksumAndRoleContainColons(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, _, _ := keepertest.AnchoringKeeper(t)
	registryID := uint64(42)
	roleA := k.RecordRole(registryID, "a:b", "c")
	roleB := k.RecordRole(registryID, "a", "b:c")
	require.NotEqual(t, roleA, roleB)
}
