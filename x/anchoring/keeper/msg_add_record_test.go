package keeper_test

import (
	"strings"
	"testing"

	appparams "github.com/MANTRA-Chain/nvnmchain/app/params"
	keepertest "github.com/MANTRA-Chain/nvnmchain/testutil/keeper"
	"github.com/MANTRA-Chain/nvnmchain/x/anchoring/keeper"
	"github.com/MANTRA-Chain/nvnmchain/x/anchoring/types"
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

	_, err := ms.AddRecord(ctx, &types.MsgAddRecord{Sender: keepertest.TestSenderAddr})
	require.Error(t, err)
	require.Contains(t, err.Error(), "record cannot be nil")
}

func TestMsgAddRecord_ValidateBasic_NilRecord(t *testing.T) {
	appparams.SetAddressPrefixes()

	sender := keepertest.TestSenderAddr

	validRecord := func() *types.Record {
		return keepertest.ValidRecord("reg1")
	}

	testCases := []struct {
		name            string
		msg             types.MsgAddRecord
		wantErrContains string
	}{
		{
			name:            "nil record",
			msg:             types.MsgAddRecord{Sender: sender, Record: nil},
			wantErrContains: "record cannot be nil",
		},
		{
			name:            "empty sender",
			msg:             types.MsgAddRecord{Sender: "", Record: validRecord()},
			wantErrContains: "sender address cannot be empty",
		},
		{
			name:            "invalid sender",
			msg:             types.MsgAddRecord{Sender: "not-an-address", Record: validRecord()},
			wantErrContains: "invalid sender address",
		},
		{
			name: "empty checksum",
			msg: func() types.MsgAddRecord {
				record := validRecord()
				record.Checksum = ""
				return types.MsgAddRecord{Sender: sender, Record: record}
			}(),
			wantErrContains: "checksum cannot be empty",
		},
		{
			name: "empty checksum algo",
			msg: func() types.MsgAddRecord {
				record := validRecord()
				record.ChecksumAlgo = ""
				return types.MsgAddRecord{Sender: sender, Record: record}
			}(),
			wantErrContains: "checksum algorithm cannot be empty",
		},
		{
			name: "oversized checksum",
			msg: func() types.MsgAddRecord {
				record := validRecord()
				record.Checksum = strings.Repeat("a", types.MaxRecordChecksumLen+1)
				return types.MsgAddRecord{Sender: sender, Record: record}
			}(),
			wantErrContains: "checksum exceeds max length",
		},
		{
			name: "empty uri",
			msg: func() types.MsgAddRecord {
				record := validRecord()
				record.Uri = ""
				return types.MsgAddRecord{Sender: sender, Record: record}
			}(),
			wantErrContains: "uri cannot be empty",
		},
		{
			name: "empty metadata",
			msg: func() types.MsgAddRecord {
				record := validRecord()
				record.Metadata = ""
				return types.MsgAddRecord{Sender: sender, Record: record}
			}(),
			wantErrContains: "metadata cannot be empty",
		},
		{
			name: "metadata is empty object",
			msg: func() types.MsgAddRecord {
				record := validRecord()
				record.Metadata = "{}"
				return types.MsgAddRecord{Sender: sender, Record: record}
			}(),
			wantErrContains: "metadata cannot be empty",
		},
		{
			name: "oversized metadata",
			msg: func() types.MsgAddRecord {
				record := validRecord()
				record.Metadata = strings.Repeat("a", types.MaxRecordMetadataLen+1)
				return types.MsgAddRecord{Sender: sender, Record: record}
			}(),
			wantErrContains: "metadata exceeds max length",
		},
		{
			name: "empty status",
			msg: func() types.MsgAddRecord {
				record := validRecord()
				record.Status = ""
				return types.MsgAddRecord{Sender: sender, Record: record}
			}(),
			wantErrContains: "status cannot be empty",
		},
		{
			name: "empty registry",
			msg: func() types.MsgAddRecord {
				record := validRecord()
				record.Registry = ""
				return types.MsgAddRecord{Sender: sender, Record: record}
			}(),
			wantErrContains: "registry cannot be empty",
		},
		{
			name: "oversized checksum algo",
			msg: func() types.MsgAddRecord {
				record := validRecord()
				record.ChecksumAlgo = strings.Repeat("a", types.MaxRecordChecksumAlgoLen+1)
				return types.MsgAddRecord{Sender: sender, Record: record}
			}(),
			wantErrContains: "checksum algorithm exceeds max length",
		},
		{
			name: "unsupported checksum algorithm",
			msg: func() types.MsgAddRecord {
				record := validRecord()
				record.ChecksumAlgo = "md5"
				return types.MsgAddRecord{Sender: sender, Record: record}
			}(),
			wantErrContains: "unsupported checksum algorithm",
		},
		{
			name: "non-hex checksum",
			msg: func() types.MsgAddRecord {
				record := validRecord()
				record.Checksum = strings.Repeat("z", 64)
				return types.MsgAddRecord{Sender: sender, Record: record}
			}(),
			wantErrContains: "checksum must be hex-encoded",
		},
		{
			name: "checksum length mismatch for sha256",
			msg: func() types.MsgAddRecord {
				record := validRecord()
				record.Checksum = strings.Repeat("a", 63)
				return types.MsgAddRecord{Sender: sender, Record: record}
			}(),
			wantErrContains: "checksum length",
		},
		{
			name: "checksum length mismatch for sha512",
			msg: func() types.MsgAddRecord {
				record := validRecord()
				record.ChecksumAlgo = "sha512"
				record.Checksum = strings.Repeat("a", 64) // sha512 needs 128
				return types.MsgAddRecord{Sender: sender, Record: record}
			}(),
			wantErrContains: "checksum length",
		},
		{
			name: "valid sha512 checksum accepted",
			msg: func() types.MsgAddRecord {
				record := validRecord()
				record.ChecksumAlgo = "sha512"
				record.Checksum = strings.Repeat("a", 128)
				return types.MsgAddRecord{Sender: sender, Record: record}
			}(),
			wantErrContains: "", // no error expected
		},
		{
			name: "case-insensitive algo SHA256 accepted",
			msg: func() types.MsgAddRecord {
				record := validRecord()
				record.ChecksumAlgo = "SHA256"
				return types.MsgAddRecord{Sender: sender, Record: record}
			}(),
			wantErrContains: "", // no error expected
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErrContains == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrContains)
			}
		})
	}
}

func TestMsgAddRecord_OversizedChecksumAlgo_RejectedByMsgServerPath(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	sender := keepertest.TestSenderAddr

	// sender becomes admin and can add records
	keepertest.MustCreateAnchoringRegistry(t, k, ctx, sender, "reg1")

	tooLongAlgo := strings.Repeat("a", types.MaxRecordChecksumAlgoLen+1)
	_, err := ms.AddRecord(ctx, &types.MsgAddRecord{
		Sender: sender,
		Record: &types.Record{
			Registry:     "reg1",
			Uri:          "ipfs://bafy...",
			Checksum:     "deadbeef",
			ChecksumAlgo: tooLongAlgo,
			Metadata:     "{\"k\":\"v\"}",
			Status:       "active",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "checksum algorithm exceeds max length")
}

func TestMsgAddRecord_OversizedChecksum_RejectedByMsgServerPath(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	sender := keepertest.TestSenderAddr

	// sender becomes admin and can add records
	keepertest.MustCreateAnchoringRegistry(t, k, ctx, sender, "reg1")
	tooLongChecksum := strings.Repeat("a", types.MaxRecordChecksumLen+1)
	_, err := ms.AddRecord(ctx, &types.MsgAddRecord{
		Sender: sender,
		Record: &types.Record{
			Registry:     "reg1",
			Uri:          "ipfs://bafy...",
			Checksum:     tooLongChecksum,
			ChecksumAlgo: "sha256",
			Metadata:     "{\"k\":\"v\"}",
			Status:       "active",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "checksum exceeds max length")
}

func TestMsgAddRecord_OversizedMetadata_RejectedByMsgServerPath(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	sender := keepertest.TestSenderAddr

	keepertest.MustCreateAnchoringRegistry(t, k, ctx, sender, "reg1")

	tooLongMetadata := strings.Repeat("a", types.MaxRecordMetadataLen+1)
	_, err := ms.AddRecord(ctx, &types.MsgAddRecord{
		Sender: sender,
		Record: &types.Record{
			Registry:     "reg1",
			Uri:          "ipfs://bafy...",
			Checksum:     "deadbeef",
			ChecksumAlgo: "sha256",
			Metadata:     tooLongMetadata,
			Status:       "active",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "metadata exceeds max length")
}
