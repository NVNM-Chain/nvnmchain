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
	"math"
	"net/url"
	"path/filepath"
	"strings"
	"unicode/utf8"

	_ "modernc.org/sqlite"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	"github.com/cosmos/cosmos-sdk/codec"
)

// MinContainsQueryLen is the shortest CONTAINS query a trigram index can
// answer. The other modes have no minimum.
const MinContainsQueryLen = 3

// ErrContainsTooShort rejects a shorter one rather than scanning for it.
var ErrContainsTooShort = fmt.Errorf("contains query needs at least %d characters", MinContainsQueryLen)

// Store is a local, opt-in SQLite-backed index of registry names.
type Store struct {
	db  *sql.DB
	cdc codec.BinaryCodec
}

// Open opens (creating if needed) the SQLite database at path and ensures
// its schema exists. cdc is used to decode the raw Registry proto bytes
// captured off the committed changeset.
//
// The database runs in WAL mode so RPC readers never block the writer:
// ListenCommit runs synchronously inside block Commit, and a burst of
// search queries must not delay it. Writers are serial by construction
// (backfill at startup, then one batch per committed block); busy_timeout
// covers the rare overlap with a WAL checkpoint.
//
// case_sensitive_like is on because SQLite serves LIKE from a B-tree index
// only when it is. Both sides are lowercased in Go, so matching stays
// case-insensitive.
func Open(path string, cdc codec.BinaryCodec) (*Store, error) {
	// Absolute so the URI below is always file:///..., never file://name
	// with the file name parsed as a host.
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("nameindex: resolve %s: %w", path, err)
	}
	dsn := url.URL{
		Scheme:   "file",
		Path:     abs,
		RawQuery: "_pragma=busy_timeout(5000)&_pragma=case_sensitive_like(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)",
	}
	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, fmt.Errorf("nameindex: open %s: %w", path, err)
	}

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("nameindex: migrate schema: %w", err)
	}
	return &Store{db: db, cdc: cdc}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// Count returns the number of indexed registries. Ids are dense and add-only,
// so a count equal to the chain's RegistryCount means the index is complete.
func (s *Store) Count() (uint64, error) {
	var n uint64
	if err := s.db.QueryRow(`SELECT count(*) FROM registries`).Scan(&n); err != nil {
		return 0, fmt.Errorf("nameindex: count: %w", err)
	}
	return n, nil
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

-- Trigram index serving CONTAINS. External-content: it stores only the
-- trigram postings and reads name_lower back from registries, and the
-- triggers keep it current inside the same transaction as the row.
CREATE VIRTUAL TABLE IF NOT EXISTS registries_fts USING fts5(
	name_lower,
	content='registries',
	content_rowid='id',
	tokenize='trigram'
);
CREATE TRIGGER IF NOT EXISTS registries_fts_ai AFTER INSERT ON registries BEGIN
	INSERT INTO registries_fts(rowid, name_lower) VALUES (new.id, new.name_lower);
END;
CREATE TRIGGER IF NOT EXISTS registries_fts_au AFTER UPDATE ON registries BEGIN
	INSERT INTO registries_fts(registries_fts, rowid, name_lower) VALUES ('delete', old.id, old.name_lower);
	INSERT INTO registries_fts(rowid, name_lower) VALUES (new.id, new.name_lower);
END;
CREATE TRIGGER IF NOT EXISTS registries_fts_ad AFTER DELETE ON registries BEGIN
	INSERT INTO registries_fts(registries_fts, rowid, name_lower) VALUES ('delete', old.id, old.name_lower);
END;
`

// execer is the subset of *sql.DB and *sql.Tx that upsert needs.
type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func upsert(e execer, cdc codec.BinaryCodec, reg *types.Registry) error {
	data, err := cdc.Marshal(reg)
	if err != nil {
		return fmt.Errorf("nameindex: marshal registry %d: %w", reg.Id, err)
	}
	lower := strings.ToLower(reg.Name)
	_, err = e.Exec(
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

// Upsert indexes (or re-indexes) a single registry in its own transaction.
// Registries are add-only on-chain, but Upsert is idempotent so backfill and
// listener replay converge on the same content.
func (s *Store) Upsert(reg *types.Registry) error {
	return upsert(s.db, s.cdc, reg)
}

// Batch groups upserts into one SQLite transaction, so a backfill or a block
// with several registry writes costs one fsync and lands atomically.
type Batch struct {
	tx  *sql.Tx
	cdc codec.BinaryCodec
}

// Begin starts a write batch. Commit it to publish; defer Close so an early
// return rolls it back.
func (s *Store) Begin() (*Batch, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("nameindex: begin: %w", err)
	}
	return &Batch{tx: tx, cdc: s.cdc}, nil
}

// Upsert indexes (or re-indexes) a registry within the batch.
func (b *Batch) Upsert(reg *types.Registry) error {
	return upsert(b.tx, b.cdc, reg)
}

// Commit publishes every Upsert in the batch at once.
func (b *Batch) Commit() error {
	if err := b.tx.Commit(); err != nil {
		return fmt.Errorf("nameindex: commit: %w", err)
	}
	return nil
}

// Close rolls the batch back unless it was committed. Safe to defer
// unconditionally.
func (b *Batch) Close() {
	_ = b.tx.Rollback() // sql.ErrTxDone after Commit; nothing to undo
}

// searchStmt builds the statement Search runs for mode. The lowercased query
// is folded into the single bound pattern argument; LIMIT and OFFSET are bound
// after it by the caller. It is separate from Search so tests can EXPLAIN the
// exact statement and pin which index serves each mode.
func searchStmt(mode types.RegistryNameMatchMode, lower string) (stmt, arg string) {
	var where string
	switch mode {
	case types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_CONTAINS:
		// FTS5 serves ORDER BY rowid natively, so the page comes straight
		// off the trigram index and each hit is one primary-key lookup.
		return `SELECT r.data FROM registries_fts f JOIN registries r ON r.id = f.rowid ` +
			`WHERE f.name_lower MATCH ? ORDER BY f.rowid LIMIT ? OFFSET ?`, ftsPhrase(lower)
	case types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_PREFIX:
		where, arg = `name_lower LIKE ? ESCAPE '\'`, escapeLike(lower)+"%"
	case types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_SUFFIX:
		where, arg = `name_rev_lower LIKE ? ESCAPE '\'`, escapeLike(reverse(lower))+"%"
	default: // EXACT; also UNSPECIFIED and any out-of-range value
		where, arg = `name_lower = ?`, lower
	}
	return `SELECT data FROM registries WHERE ` + where + ` ORDER BY id LIMIT ? OFFSET ?`, arg
}

// Search returns registries whose name matches query under mode, ordered by
// id, applying a plain offset/limit page. Matching is always
// case-insensitive. A CONTAINS query shorter than MinContainsQueryLen
// characters returns ErrContainsTooShort.
func (s *Store) Search(mode types.RegistryNameMatchMode, query string, limit, offset uint64) ([]*types.Registry, error) {
	if mode == types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_CONTAINS && utf8.RuneCountInString(query) < MinContainsQueryLen {
		return nil, ErrContainsTooShort
	}
	stmt, arg := searchStmt(mode, strings.ToLower(query))

	// database/sql rejects uint64 values above MaxInt64. Anything that large
	// is past the end of any index, so clamp rather than fail the query.
	limit = min(limit, math.MaxInt64)
	offset = min(offset, math.MaxInt64)

	rows, err := s.db.Query(stmt, arg, int64(limit), int64(offset))
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

// ftsPhrase quotes s as one FTS5 string so every character in it, including
// the operators and quotes FTS5 would otherwise parse, is matched literally.
func ftsPhrase(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
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
