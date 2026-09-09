// Package nameindex implements an opt-in, per-node, off-chain index over
// anchoring Registry names. It exists because the on-chain KV store only
// supports id lookups and ordered-prefix scans: it cannot answer "does any
// registry's name contain X" or "which registries end in X" without a full
// table scan inside consensus-critical code. This package keeps that lookup
// out of the state machine entirely, backed by a local SQLite file that a
// node operator opts into via app.toml.
//
// The index is not part of consensus: nodes are not required to run it, two
// nodes may answer a search differently while one is still backfilling, and
// its content is derived, never authoritative. The authoritative registry
// data always lives in the anchoring module's own collections.
package nameindex

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	"github.com/cosmos/cosmos-sdk/codec"
)

// MatchMode mirrors types.RegistryNameMatchMode, kept as its own type so this
// package does not need to import the query-request wrapper.
type MatchMode int32

const (
	MatchModeExact MatchMode = iota
	MatchModePrefix
	MatchModeSuffix
	MatchModeContains
)

// FromProto maps the generated enum onto MatchMode, defaulting the
// unspecified value to an exact match.
func FromProto(m types.RegistryNameMatchMode) MatchMode {
	switch m {
	case types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_PREFIX:
		return MatchModePrefix
	case types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_SUFFIX:
		return MatchModeSuffix
	case types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_CONTAINS:
		return MatchModeContains
	default:
		return MatchModeExact
	}
}

// Store is a local, opt-in SQLite-backed index of registry names.
type Store struct {
	db  *sql.DB
	cdc codec.BinaryCodec
}

// Open opens (creating if needed) the SQLite database at path and ensures
// its schema exists. cdc is used to decode the raw Registry proto bytes
// captured off the committed changeset.
func Open(path string, cdc codec.BinaryCodec) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("nameindex: open %s: %w", path, err)
	}
	// The index is only ever written from a single ABCIListener goroutine
	// (block commit) plus occasional backfill; a single connection avoids
	// SQLITE_BUSY without needing WAL/busy-timeout tuning.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("nameindex: migrate schema: %w", err)
	}

	return &Store{db: db, cdc: cdc}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

const schema = `
CREATE TABLE IF NOT EXISTS registries (
	id             INTEGER PRIMARY KEY,
	name_lower     TEXT NOT NULL,
	name_rev_lower TEXT NOT NULL,
	data           BLOB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_registries_name_lower ON registries(name_lower);
CREATE INDEX IF NOT EXISTS idx_registries_name_rev_lower ON registries(name_rev_lower);
`

// Upsert indexes (or re-indexes) a single registry. Registries are add-only
// on-chain, but Upsert is idempotent so backfill and listener replay can
// never desync the index.
func (s *Store) Upsert(reg *types.Registry) error {
	data, err := s.cdc.Marshal(reg)
	if err != nil {
		return fmt.Errorf("nameindex: marshal registry %d: %w", reg.Id, err)
	}
	lower := strings.ToLower(reg.Name)
	_, err = s.db.Exec(
		`INSERT INTO registries (id, name_lower, name_rev_lower, data) VALUES (?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET name_lower = excluded.name_lower,
		                                name_rev_lower = excluded.name_rev_lower,
		                                data = excluded.data`,
		reg.Id, lower, reverse(lower), data,
	)
	if err != nil {
		return fmt.Errorf("nameindex: upsert registry %d: %w", reg.Id, err)
	}
	return nil
}

// Search returns registries whose name matches query under mode, ordered by
// id, applying a plain offset/limit page. Matching is always
// case-insensitive.
func (s *Store) Search(mode MatchMode, query string, limit, offset uint64) ([]*types.Registry, error) {
	lower := strings.ToLower(query)

	var column, pattern string
	switch mode {
	case MatchModePrefix:
		column, pattern = "name_lower", escapeLike(lower)+"%"
	case MatchModeSuffix:
		column, pattern = "name_rev_lower", escapeLike(reverse(lower))+"%"
	case MatchModeContains:
		column, pattern = "name_lower", "%"+escapeLike(lower)+"%"
	default: // MatchModeExact
		column, pattern = "name_lower", ""
	}

	var (
		rows *sql.Rows
		err  error
	)
	if mode == MatchModeExact {
		rows, err = s.db.Query(
			`SELECT data FROM registries WHERE name_lower = ? ORDER BY id LIMIT ? OFFSET ?`,
			lower, limit, offset,
		)
	} else {
		rows, err = s.db.Query(
			fmt.Sprintf(`SELECT data FROM registries WHERE %s LIKE ? ESCAPE '\' ORDER BY id LIMIT ? OFFSET ?`, column),
			pattern, limit, offset,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("nameindex: search: %w", err)
	}
	defer rows.Close()

	var results []*types.Registry
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("nameindex: scan: %w", err)
		}
		reg := &types.Registry{}
		if err := s.cdc.Unmarshal(data, reg); err != nil {
			return nil, fmt.Errorf("nameindex: decode registry: %w", err)
		}
		results = append(results, reg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("nameindex: search: %w", err)
	}
	return results, nil
}

// escapeLike escapes the LIKE metacharacters in a literal so the query only
// ever means "this literal substring", never a user-supplied wildcard.
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// reverse reverses s by rune so multi-byte UTF-8 registry names still match
// correctly under suffix search.
func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
