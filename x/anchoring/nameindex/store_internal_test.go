package nameindex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
)

// TestOpen_WALReadersDoNotBlockWriter pins the property ListenCommit relies
// on: it runs inside block Commit, so an RPC reader holding a result set open
// must not make the writer wait. That is only true in WAL mode.
func TestOpen_WALReadersDoNotBlockWriter(t *testing.T) {
	// A path with a space checks the DSN is URI-encoded rather than pasted.
	dir := filepath.Join(t.TempDir(), "with space")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	s, err := Open(filepath.Join(dir, "index.db"), cdc)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	var mode string
	require.NoError(t, s.db.QueryRow(`PRAGMA journal_mode`).Scan(&mode))
	require.Equal(t, "wal", mode)

	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: "a"}))
	rows, err := s.db.Query(`SELECT id FROM registries`)
	require.NoError(t, err)
	defer rows.Close()
	require.True(t, rows.Next(), "reader holds a snapshot open")

	// With the reader still open, a write on another pooled connection
	// succeeds immediately; in rollback-journal mode it would block on the
	// reader until busy_timeout expired and then fail.
	require.NoError(t, s.Upsert(&types.Registry{Id: 2, Name: "b"}))
}
