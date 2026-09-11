package evmlayout_test

import (
	"bufio"
	"bytes"
	"cmp"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/collections"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/evmlayout"
	anchoringkeeper "github.com/NVNM-Chain/nvnmchain/x/anchoring/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// Opt-in; how the Tempo dump is made:
//
//	NVNMCHAIN_EXPORT_DIR=/private/tmp/from-chain \
//	NVNMCHAIN_DUMP_OUT=$HOME/anchoring-state.bin \
//	go test -run TestDumpTheMainnetSeed -v -timeout 4h ./x/anchoring/evmlayout/
//
// It replays mainnet before v1.2.0 (NVNMCHAIN_PRESEED, default testdata/mainnet-preseed.json) and
// then the v1.2.0 seed, so ids, creators and timestamps come out as mainnet holds them.
// NVNMCHAIN_EXPORT_LIMIT stops the seed early, NVNMCHAIN_DUMP_ADDRESS replaces Address, and
// NVNMCHAIN_DUMP_OUT must be absolute or the file lands in the package directory.
const (
	exportDirEnvVar   = "NVNMCHAIN_EXPORT_DIR"
	exportLimitEnvVar = "NVNMCHAIN_EXPORT_LIMIT"
	preseedEnvVar     = "NVNMCHAIN_PRESEED"
	dumpAddressEnvVar = "NVNMCHAIN_DUMP_ADDRESS"
	dumpOutEnvVar     = "NVNMCHAIN_DUMP_OUT"
	defaultPreseed    = "testdata/mainnet-preseed.json"

	// v1_2.MainnetRegistryAdmin, the creator of every seeded registry.
	mainnetRegistryAdmin = "nvnm14a3em3mr9mvta9ccgk80wn0dxgzt5lkt2r8trx"

	goTimeLayout = "2006-01-02 15:04:05.999999999 -0700 MST" // time.Time.String()
)

// Mainnet's anchoring state before the seed, as nvnmchaind queries return it.
type preseed struct {
	Admin      string           `json:"admin"`
	SeedTime   time.Time        `json:"seed_time"`
	Registries []storedRegistry `json:"registries"`
	Records    []struct {
		RegistryID uint64       `json:"registry_id,string"`
		Stored     storedRecord `json:"stored"`
	} `json:"records"`
}

type storedRegistry struct {
	ID          uint64 `json:"id,string"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Creator     string `json:"creator"`
	CreatedAt   string `json:"created_at"`
	Metadata    string `json:"metadata"`
}

func (r storedRegistry) registry() types.Registry {
	return types.Registry{
		Id: r.ID, Name: r.Name, Description: r.Description,
		Creator: r.Creator, CreatedAt: r.CreatedAt, Metadata: r.Metadata,
	}
}

type storedRecord struct {
	Uri          string `json:"uri"`
	Checksum     string `json:"checksum"`
	ChecksumAlgo string `json:"checksum_algo"`
	Metadata     string `json:"metadata"`
	Timestamp    string `json:"timestamp"`
	Status       string `json:"status"`
	RecordID     uint64 `json:"record_id,string"`
	Index        uint64 `json:"index,string"`
	IsLatest     bool   `json:"is_latest"`
	RegistryID   uint64 `json:"registry_id,string"`
}

func (r storedRecord) record() types.Record {
	return types.Record{
		Uri: r.Uri, Checksum: r.Checksum, ChecksumAlgo: r.ChecksumAlgo, Metadata: r.Metadata,
		Timestamp: r.Timestamp, Status: r.Status, RecordId: r.RecordID, Index: r.Index,
		IsLatest: r.IsLatest, RegistryId: r.RegistryID,
	}
}

// The export's own files, as the v1.2 seed read them.
type manifestFile struct {
	Registry string `json:"registry"`
	Records  int    `json:"records"`
	File     string `json:"file"`
	Tranche  int    `json:"tranche"`
	Sha256Gz string `json:"sha256_gz"`
}

type manifest struct {
	Totals struct {
		Registries int `json:"registries"`
		Records    int `json:"records"`
	} `json:"totals"`
	Files []manifestFile `json:"files"`
}

type registryImport struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Metadata    string `json:"metadata"`
}

// export is what is staged in Dir: the manifest, its files by tranche then registry as the v1.2
// seed took them, and the registries.
type export struct {
	manifest
	Dir        string
	Registries []registryImport
}

type recordImport struct {
	Registry     string `json:"registry"`
	Uri          string `json:"uri"`
	Checksum     string `json:"checksum"`
	ChecksumAlgo string `json:"checksumAlgo"`
	Metadata     string `json:"metadata"`
	Status       string `json:"status"`
}

func TestDumpTheMainnetSeed(t *testing.T) {
	dir, out := os.Getenv(exportDirEnvVar), os.Getenv(dumpOutEnvVar)
	if dir == "" || out == "" {
		t.Skipf("set %s and %s to make the dump", exportDirEnvVar, dumpOutEnvVar)
	}
	address := evmlayout.Address
	if raw := os.Getenv(dumpAddressEnvVar); raw != "" {
		require.Truef(t, common.IsHexAddress(raw), "%s=%q", dumpAddressEnvVar, raw)
		address = common.HexToAddress(raw)
	}
	limit, err := strconv.Atoi(cmp.Or(os.Getenv(exportLimitEnvVar), "0"))
	require.NoError(t, err)

	k, ctx := anchoringKeeper(t)
	state := replayPreseed(t, k, ctx)
	seedTheExport(t, k, ctx.WithBlockTime(state.SeedTime), readExport(t, dir), limit)
	count, err := k.RegistryCount.Get(ctx)
	require.NoError(t, err)
	t.Logf("registries %d in all, seeded at %s", count, state.SeedTime)

	f, err := os.Create(out)
	require.NoError(t, err)
	dump := evmlayout.NewDump(f, address)
	began, slots := time.Now(), 0
	require.NoError(t, evmlayout.Migrate(ctx, k, func(w evmlayout.Write) error {
		slots++
		return dump.Put(w)
	}))
	require.NoError(t, dump.Close())
	require.NoError(t, f.Close())
	t.Logf("dump       %d slots to %s in %s", slots, out, time.Since(began).Round(time.Second))
}

// chainTime parses a keeper timestamp and insists it round-trips: the contract returns the string.
func chainTime(t *testing.T, s string) time.Time {
	t.Helper()
	at, err := time.Parse(goTimeLayout, s)
	require.NoError(t, err)
	require.Equal(t, s, at.UTC().String())
	return at
}

// replayPreseed puts back what mainnet held before the seed, from NVNMCHAIN_PRESEED, each row at
// its own block time so the seeded ids carry on from there. Every value is read back, not just
// the ids.
func replayPreseed(t *testing.T, k anchoringkeeper.Keeper, ctx sdk.Context) preseed {
	t.Helper()
	var state preseed
	readJSON(t, cmp.Or(os.Getenv(preseedEnvVar), defaultPreseed), &state)
	require.NoError(t, k.Params.Set(ctx, types.NewParams(state.Admin)))

	for _, r := range state.Registries {
		at := chainTime(t, r.CreatedAt)
		id, err := k.AddRegistry(ctx.WithBlockTime(at), sdk.MustAccAddressFromBech32(r.Creator),
			r.Name, r.Description, r.Metadata)
		require.NoError(t, err)
		require.Equal(t, r.ID, id)
	}
	// The sender is not stored and no role existed yet, so the registry's creator stands in.
	for _, r := range state.Records {
		registry, err := k.Registries.Get(ctx, r.RegistryID)
		require.NoError(t, err)
		id, err := k.AddRecord(ctx.WithBlockTime(chainTime(t, r.Stored.Timestamp)),
			sdk.MustAccAddressFromBech32(registry.Creator), types.Record{
				RegistryId:   r.RegistryID,
				Uri:          r.Stored.Uri,
				Checksum:     r.Stored.Checksum,
				ChecksumAlgo: r.Stored.ChecksumAlgo,
				Metadata:     r.Stored.Metadata,
				Status:       r.Stored.Status,
			})
		require.NoError(t, err)
		require.Equal(t, r.Stored.RecordID, id)
		index, err := k.RecordIndices.Get(ctx, collections.Join(r.RegistryID, id))
		require.NoError(t, err)
		require.Equal(t, r.Stored.Index, index)
	}

	// Every value, not just the ids, has to come out as the chain holds it.
	for _, r := range state.Registries {
		got, err := k.Registries.Get(ctx, r.ID)
		require.NoError(t, err)
		require.Equal(t, r.registry(), got)
	}
	for _, r := range state.Records {
		got, err := k.Records.Get(ctx, collections.Join3(r.RegistryID, r.Stored.RecordID, r.Stored.Index))
		require.NoError(t, err)
		require.Equal(t, r.Stored.record(), got)
	}
	t.Logf("preseed    %d registries, %d record versions", len(state.Registries), len(state.Records))
	return state
}

// seedTheExport adds e as the v1.2 seed did, as MainnetRegistryAdmin at ctx's block time:
// registries in file order, then records by tranche and registry, stopping at a non-zero limit.
func seedTheExport(t *testing.T, k anchoringkeeper.Keeper, ctx sdk.Context, e export, limit int) map[string]uint64 {
	t.Helper()
	creator := sdk.MustAccAddressFromBech32(mainnetRegistryAdmin)

	began := time.Now()
	ids := make(map[string]uint64, len(e.Registries))
	for _, r := range e.Registries {
		id, err := k.AddRegistry(ctx, creator, r.Name, r.Description, r.Metadata)
		require.NoError(t, err)
		ids[r.Name] = id
	}
	t.Logf("registries %d in %s", len(e.Registries), time.Since(began).Round(time.Millisecond))

	began = time.Now()
	loaded := 0
	for _, entry := range e.Files {
		if limit != 0 && loaded >= limit {
			break
		}
		id, ok := ids[entry.Registry]
		require.Truef(t, ok, "manifest names registry %q", entry.Registry)

		for _, rec := range readTranche(t, e.Dir, entry) {
			if limit != 0 && loaded >= limit {
				break
			}
			_, err := k.AddRecord(ctx, creator, types.Record{
				Uri:          rec.Uri,
				Checksum:     rec.Checksum,
				ChecksumAlgo: rec.ChecksumAlgo,
				Metadata:     rec.Metadata,
				Status:       rec.Status,
				RegistryId:   id,
			})
			require.NoErrorf(t, err, "%s: %s", entry.File, rec.Checksum)
			loaded++
		}
	}
	t.Logf("records    %d in %s", loaded, time.Since(began).Round(time.Second))
	return ids
}

func readExport(t *testing.T, dir string) export {
	t.Helper()
	e := export{Dir: dir}
	readJSON(t, filepath.Join(dir, "manifest.json"), &e.manifest)
	readJSON(t, filepath.Join(dir, "registries.json"), &e.Registries)
	slices.SortFunc(e.Files, func(a, b manifestFile) int {
		return cmp.Or(cmp.Compare(a.Tranche, b.Tranche), cmp.Compare(a.Registry, b.Registry))
	})
	return e
}

func readJSON(t *testing.T, path string, into any) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, into))
}

// readTranche reads one tranche file after checking its manifest digest, as the v1.2 seed did.
func readTranche(t *testing.T, dir string, entry manifestFile) []recordImport {
	t.Helper()
	gzBytes, err := os.ReadFile(filepath.Clean(filepath.Join(dir, entry.File)))
	require.NoErrorf(t, err, "%s is not staged", entry.File)
	sum := sha256.Sum256(gzBytes)
	require.Equalf(t, entry.Sha256Gz, hex.EncodeToString(sum[:]), "%s: sha256_gz", entry.File)

	reader, err := gzip.NewReader(bytes.NewReader(gzBytes))
	require.NoError(t, err)
	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())

	out := make([]recordImport, 0, entry.Records)
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var rec recordImport
		require.NoError(t, json.Unmarshal(line, &rec))
		out = append(out, rec)
	}
	require.NoError(t, scanner.Err())
	require.Lenf(t, out, entry.Records, "%s: record count", entry.File)
	return out
}
