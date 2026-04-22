package types_test

import (
	"strings"
	"testing"

	anchoringtest "github.com/MANTRA-Chain/nvnmchain/testutil/anchoring"
	"github.com/MANTRA-Chain/nvnmchain/x/anchoring/types"
	"github.com/stretchr/testify/require"
)

func validRecord() types.Record {
	return anchoringtest.ValidRecord("reg1")
}

func TestValidateRecordForCreate_HappyPaths(t *testing.T) {
	cases := []struct {
		name   string
		record types.Record
	}{
		{
			name:   "valid sha256",
			record: validRecord(),
		},
		{
			name: "valid sha256 uppercase algo",
			record: func() types.Record {
				r := validRecord()
				r.ChecksumAlgo = "SHA256"
				return r
			}(),
		},
		{
			name: "valid sha-256 alias",
			record: func() types.Record {
				r := validRecord()
				r.ChecksumAlgo = "sha-256"
				return r
			}(),
		},
		{
			name: "valid sha512",
			record: func() types.Record {
				r := validRecord()
				r.ChecksumAlgo = "sha512"
				r.Checksum = strings.Repeat("b", 128)
				return r
			}(),
		},
		{
			name: "valid sha-512 alias",
			record: func() types.Record {
				r := validRecord()
				r.ChecksumAlgo = "sha-512"
				r.Checksum = strings.Repeat("b", 128)
				return r
			}(),
		},
		{
			name: "valid sha3-256",
			record: func() types.Record {
				r := validRecord()
				r.ChecksumAlgo = "sha3-256"
				r.Checksum = strings.Repeat("c", 64)
				return r
			}(),
		},
		{
			name: "uppercase hex digits accepted",
			record: func() types.Record {
				r := validRecord()
				r.Checksum = strings.Repeat("A", 64)
				return r
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, types.ValidateRecordForCreate(tc.record))
		})
	}
}

func TestValidateRecordForCreate_ErrorPaths(t *testing.T) {
	cases := []struct {
		name            string
		record          types.Record
		wantErrContains string
	}{
		{
			name:            "empty checksum",
			record:          func() types.Record { r := validRecord(); r.Checksum = ""; return r }(),
			wantErrContains: "checksum cannot be empty",
		},
		{
			name:            "empty checksum algo",
			record:          func() types.Record { r := validRecord(); r.ChecksumAlgo = ""; return r }(),
			wantErrContains: "checksum algorithm cannot be empty",
		},
		{
			name:            "empty uri",
			record:          func() types.Record { r := validRecord(); r.Uri = ""; return r }(),
			wantErrContains: "uri cannot be empty",
		},
		{
			name:            "empty metadata",
			record:          func() types.Record { r := validRecord(); r.Metadata = ""; return r }(),
			wantErrContains: "metadata cannot be empty",
		},
		{
			name:            "metadata is empty object",
			record:          func() types.Record { r := validRecord(); r.Metadata = "{}"; return r }(),
			wantErrContains: "metadata cannot be empty",
		},
		{
			name:            "empty status",
			record:          func() types.Record { r := validRecord(); r.Status = ""; return r }(),
			wantErrContains: "status cannot be empty",
		},
		{
			name:            "empty registry",
			record:          func() types.Record { r := validRecord(); r.Registry = ""; return r }(),
			wantErrContains: "registry cannot be empty",
		},
		{
			name: "checksum exceeds max length",
			record: func() types.Record {
				r := validRecord()
				r.Checksum = strings.Repeat("a", types.MaxRecordChecksumLen+1)
				return r
			}(),
			wantErrContains: "checksum exceeds max length",
		},
		{
			name:            "unsupported algorithm",
			record:          func() types.Record { r := validRecord(); r.ChecksumAlgo = "md5"; return r }(),
			wantErrContains: "unsupported checksum algorithm",
		},
		{
			name:            "non-hex checksum",
			record:          func() types.Record { r := validRecord(); r.Checksum = strings.Repeat("z", 64); return r }(),
			wantErrContains: "checksum must be hex-encoded",
		},
		{
			name:            "sha256 wrong length (63 chars)",
			record:          func() types.Record { r := validRecord(); r.Checksum = strings.Repeat("a", 63); return r }(),
			wantErrContains: "checksum length",
		},
		{
			name: "sha512 wrong length (64 chars instead of 128)",
			record: func() types.Record {
				r := validRecord()
				r.ChecksumAlgo = "sha512"
				r.Checksum = strings.Repeat("a", 64)
				return r
			}(),
			wantErrContains: "checksum length",
		},
		{
			name: "sha3-256 wrong length (63 chars)",
			record: func() types.Record {
				r := validRecord()
				r.ChecksumAlgo = "sha3-256"
				r.Checksum = strings.Repeat("a", 63)
				return r
			}(),
			wantErrContains: "checksum length",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := types.ValidateRecordForCreate(tc.record)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErrContains)
		})
	}
}
