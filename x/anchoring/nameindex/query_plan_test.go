package nameindex

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"
)

// explain returns the "detail" column of EXPLAIN QUERY PLAN for query. It is
// an internal (package nameindex) test so it can reach s.db directly; the
// SQL text below must be kept in sync with the WHERE clauses Search builds.
func explain(t *testing.T, s *Store, where string) []string {
	t.Helper()
	rows, err := s.db.Query(`EXPLAIN QUERY PLAN SELECT data FROM registries WHERE ` + where + ` ORDER BY id LIMIT 5 OFFSET 0`)
	require.NoError(t, err)
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		require.NoError(t, rows.Scan(&id, &parent, &notUsed, &detail))
		lines = append(lines, detail)
	}
	require.NoError(t, rows.Err())
	return lines
}

// TestSearch_UsesIndexes pins that exact/prefix/suffix hit a B-tree index
// instead of scanning every row, and documents that contains cannot: a
// leading '%' rules out any index range scan, B-tree or otherwise, without
// FTS5/trigram support this package does not use.
//
// The precompile bills a registriesByName call only for the rows it
// returns (see chargeIndexReadGas), not the rows SQLite examines internally,
// so losing an index silently would turn a query into unbilled CPU work
// under contention rather than a metered one. That regression would not show
// up in store_test.go, which only asserts on results, not the query plan.
// `case_sensitive_like` (set in Open) is what makes the range scan possible
// at all: SQLite disables the LIKE optimization by default because its
// case-insensitive matching cannot be expressed as a binary-collated index
// range.
func TestSearch_UsesIndexes(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	s, err := Open(filepath.Join(t.TempDir(), "test.db"), cdc)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	testCases := []struct {
		mode      string
		where     string
		wantIndex string
	}{
		{"exact", `name_lower = 'foo'`, "idx_registries_name_lower"},
		{"prefix", `name_lower LIKE 'foo%' ESCAPE '\'`, "idx_registries_name_lower"},
		{"suffix", `name_rev_lower LIKE 'oof%' ESCAPE '\'`, "idx_registries_name_rev_lower"},
	}
	for _, tc := range testCases {
		t.Run(tc.mode, func(t *testing.T) {
			plan := strings.Join(explain(t, s, tc.where), "\n")
			require.Containsf(t, plan, "SEARCH registries USING INDEX "+tc.wantIndex, "plan:\n%s", plan)
			require.NotContains(t, plan, "SCAN registries")
		})
	}

	t.Run("contains cannot use an index", func(t *testing.T) {
		plan := strings.Join(explain(t, s, `name_lower LIKE '%foo%' ESCAPE '\'`), "\n")
		require.Contains(t, plan, "SCAN registries",
			"if this now uses an index, the comment above Search's contains case is stale")
	})
}
