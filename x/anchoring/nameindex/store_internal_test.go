package nameindex

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
)

func openStore(t *testing.T) *Store {
	t.Helper()
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	s, err := Open(filepath.Join(t.TempDir(), "index.db"), cdc)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// queryPlan returns the EXPLAIN QUERY PLAN detail lines for the statement
// Search would run for mode and query.
func queryPlan(t *testing.T, s *Store, mode types.RegistryNameMatchMode, query string) []string {
	t.Helper()
	stmt, arg := searchStmt(mode, query)
	rows, err := s.db.Query("EXPLAIN QUERY PLAN "+stmt, arg, 50, 0)
	require.NoError(t, err)
	defer rows.Close()

	var details []string
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		require.NoError(t, rows.Scan(&id, &parent, &notUsed, &detail))
		details = append(details, detail)
	}
	require.NoError(t, rows.Err())
	return details
}

// TestSearchPlansUseIndexes pins which index serves each mode. SQLite only
// serves LIKE from a B-tree index with case_sensitive_like on; without it
// even a prefix pattern is a full table scan, silently.
func TestSearchPlansUseIndexes(t *testing.T) {
	s := openStore(t)
	for i := 1; i <= 20; i++ {
		require.NoError(t, s.Upsert(&types.Registry{Id: uint64(i), Name: fmt.Sprintf("reg-%02d-fund", i)}))
	}

	testCases := []struct {
		mode  types.RegistryNameMatchMode
		query string
		want  string
		// sorts is whether the plan sorts its matches into id order. Range
		// scans over the name indexes yield name order and must sort, which
		// is O(matches); the trigram index and an equality lookup already
		// come back in rowid order.
		sorts bool
	}{
		{types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_EXACT, "reg-01-fund", "USING INDEX idx_registries_name_lower", false},
		{types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_PREFIX, "reg", "USING INDEX idx_registries_name_lower", true},
		{types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_SUFFIX, "fund", "USING INDEX idx_registries_name_rev_lower", true},
		{types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_CONTAINS, "fund", "SCAN f VIRTUAL TABLE INDEX", false},
	}
	for _, tc := range testCases {
		plan := queryPlan(t, s, tc.mode, tc.query)
		joined := strings.Join(plan, "\n")
		require.NotContains(t, plan, "SCAN registries", "mode %d: %v", tc.mode, plan)
		require.Contains(t, joined, tc.want, "mode %d", tc.mode)
		require.Equal(t, tc.sorts, strings.Contains(joined, "TEMP B-TREE"), "mode %d: %v", tc.mode, plan)
	}
}

// TestOpen_BuildsTrigramIndexForExistingRows covers an index file written
// before the trigram index existed: its rows never passed through the
// triggers, so the first open with the new schema must index them.
func TestOpen_BuildsTrigramIndexForExistingRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.db")
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	s, err := Open(path, cdc)
	require.NoError(t, err)

	// Roll the file back to the pre-trigram schema, then write through it.
	_, err = s.db.Exec(`DROP TRIGGER registries_fts_ai; DROP TRIGGER registries_fts_au;
		DROP TRIGGER registries_fts_ad; DROP TABLE registries_fts;`)
	require.NoError(t, err)
	require.NoError(t, s.Upsert(&types.Registry{Id: 1, Name: "legacy-fund"}))
	require.NoError(t, s.Close())

	s, err = Open(path, cdc)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	got, err := s.Search(types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_CONTAINS, "fund", 50, 0)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "legacy-fund", got[0].Name)
}

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
