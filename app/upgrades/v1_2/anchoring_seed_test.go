package v1_2

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cosmossdk.io/collections"
	appparams "github.com/NVNM-Chain/nvnmchain/app/params"
	"github.com/NVNM-Chain/nvnmchain/app/upgrades"
	"github.com/stretchr/testify/require"
)

// fixtureRecord is the minimal shape written to a synthetic tranche jsonl fixture.
type fixtureRecord struct {
	Registry     string
	Checksum     string
	ChecksumAlgo string
	Uri          string
	Metadata     string
	Status       string
}

// writeFixtureTranche gzips recs as a jsonl file under dir/relPath and returns the
// ManifestFileEntry describing it (Registry/Records/File/Tranche/sha256 digests), so tests can
// build a manifest.json pointing at real, hash-verifiable fixture data.
func writeFixtureTranche(t *testing.T, dir, relPath string, tranche int, recs []fixtureRecord) ManifestFileEntry {
	t.Helper()
	var buf bytes.Buffer
	for _, r := range recs {
		line, err := json.Marshal(map[string]string{
			"registry":     r.Registry,
			"checksum":     r.Checksum,
			"checksumAlgo": r.ChecksumAlgo,
			"uri":          r.Uri,
			"metadata":     r.Metadata,
			"status":       r.Status,
		})
		require.NoError(t, err)
		buf.Write(line)
		buf.WriteByte('\n')
	}
	uncompressed := buf.Bytes()

	var gzBuf bytes.Buffer
	gzWriter := gzip.NewWriter(&gzBuf)
	_, err := gzWriter.Write(uncompressed)
	require.NoError(t, err)
	require.NoError(t, gzWriter.Close())
	gzBytes := gzBuf.Bytes()

	fullPath := filepath.Join(dir, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, gzBytes, 0o644))

	uncompressedSum := sha256.Sum256(uncompressed)
	gzSum := sha256.Sum256(gzBytes)

	return ManifestFileEntry{
		Registry:           recs[0].Registry,
		Records:            len(recs),
		File:               relPath,
		Tranche:            tranche,
		Sha256Gz:           hex.EncodeToString(gzSum[:]),
		Sha256Uncompressed: hex.EncodeToString(uncompressedSum[:]),
	}
}

func fixtureRegistriesJSON(t *testing.T, names ...string) []byte {
	t.Helper()
	regs := make([]RegistryImport, 0, len(names))
	for _, name := range names {
		regs = append(regs, RegistryImport{
			Name:        name,
			Description: "fixture registry " + name,
			Metadata:    `{"source":"test-fixture"}`,
			Tranche:     1,
		})
	}
	b, err := json.Marshal(regs)
	require.NoError(t, err)
	return b
}

// TestSeedAnchoringData_RecordsEndToEnd exercises the full pipeline (registry creation, tranche
// file discovery, hash verification, decompression, record writes, count reconciliation)
// against small synthetic fixtures — standing in for a real mainnet-scale export.
func TestSeedAnchoringData_RecordsEndToEnd(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx := newTestAnchoringKeeper(t)
	blockTime := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(blockTime)

	exportDir := t.TempDir()
	fileA := writeFixtureTranche(t, exportDir, "tranche-1/reg-a.jsonl.gz", 1, []fixtureRecord{
		{Registry: "reg-a", Checksum: "1 A 1", ChecksumAlgo: "cite-canonical-v1", Uri: "https://example.com/a1", Metadata: `{"case":"fixture"}`, Status: "Active"},
		{Registry: "reg-a", Checksum: "1 A 2", ChecksumAlgo: "cite-canonical-v1", Uri: "https://example.com/a2", Metadata: `{"case":"fixture"}`, Status: "Active"},
	})
	fileB := writeFixtureTranche(t, exportDir, "tranche-2/reg-b.jsonl.gz", 2, []fixtureRecord{
		{Registry: "reg-b", Checksum: "1 B 1", ChecksumAlgo: "cite-canonical-v1", Uri: "https://example.com/b1", Metadata: `{"case":"fixture"}`, Status: "Active"},
	})

	manifest := MigrationManifest{Files: []ManifestFileEntry{fileB, fileA}} // deliberately out of tranche order
	manifest.Totals.Registries = 2
	manifest.Totals.Records = 3
	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)

	err = SeedAnchoringData(ctx, &upgrades.UpgradeKeepers{AnchoringKeeper: k}, exportDir, fixtureRegistriesJSON(t, "reg-a", "reg-b"), manifestJSON)
	require.NoError(t, err)

	regAId := findRegistryIdByName(t, k, ctx, "reg-a")
	regBId := findRegistryIdByName(t, k, ctx, "reg-b")

	countA, err := k.RecordsCountByRegistry.Get(ctx, regAId)
	require.NoError(t, err)
	require.Equal(t, uint64(2), countA)

	countB, err := k.RecordsCountByRegistry.Get(ctx, regBId)
	require.NoError(t, err)
	require.Equal(t, uint64(1), countB)

	rec, err := k.Records.Get(ctx, collections.Join3(regAId, uint64(1), uint64(1)))
	require.NoError(t, err)
	require.Equal(t, blockTime.String(), rec.Timestamp)
	require.True(t, rec.IsLatest)
}

func TestSeedAnchoringData_RecordsCountMismatchRejected(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx := newTestAnchoringKeeper(t)

	exportDir := t.TempDir()
	file := writeFixtureTranche(t, exportDir, "tranche-1/reg-a.jsonl.gz", 1, []fixtureRecord{
		{Registry: "reg-a", Checksum: "1 A 1", ChecksumAlgo: "cite-canonical-v1", Uri: "https://example.com/a1", Metadata: `{"case":"fixture"}`, Status: "Active"},
	})
	file.Records = 2 // manifest claims 2 records, file actually has 1

	manifest := MigrationManifest{Files: []ManifestFileEntry{file}}
	manifest.Totals.Registries = 1
	manifest.Totals.Records = 2
	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)

	err = SeedAnchoringData(ctx, &upgrades.UpgradeKeepers{AnchoringKeeper: k}, exportDir, fixtureRegistriesJSON(t, "reg-a"), manifestJSON)
	require.Error(t, err)
	require.Contains(t, err.Error(), "manifest expects 2")
}

func TestSeedAnchoringData_GzHashMismatchRejected(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx := newTestAnchoringKeeper(t)

	exportDir := t.TempDir()
	file := writeFixtureTranche(t, exportDir, "tranche-1/reg-a.jsonl.gz", 1, []fixtureRecord{
		{Registry: "reg-a", Checksum: "1 A 1", ChecksumAlgo: "cite-canonical-v1", Uri: "https://example.com/a1", Metadata: `{"case":"fixture"}`, Status: "Active"},
	})
	file.Sha256Gz = "0000000000000000000000000000000000000000000000000000000000000000"

	manifest := MigrationManifest{Files: []ManifestFileEntry{file}}
	manifest.Totals.Registries = 1
	manifest.Totals.Records = 1
	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)

	err = SeedAnchoringData(ctx, &upgrades.UpgradeKeepers{AnchoringKeeper: k}, exportDir, fixtureRegistriesJSON(t, "reg-a"), manifestJSON)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sha256_gz mismatch")
}

func TestSeedAnchoringData_UncompressedHashMismatchRejected(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx := newTestAnchoringKeeper(t)

	exportDir := t.TempDir()
	file := writeFixtureTranche(t, exportDir, "tranche-1/reg-a.jsonl.gz", 1, []fixtureRecord{
		{Registry: "reg-a", Checksum: "1 A 1", ChecksumAlgo: "cite-canonical-v1", Uri: "https://example.com/a1", Metadata: `{"case":"fixture"}`, Status: "Active"},
	})
	file.Sha256Uncompressed = "0000000000000000000000000000000000000000000000000000000000000000"

	manifest := MigrationManifest{Files: []ManifestFileEntry{file}}
	manifest.Totals.Registries = 1
	manifest.Totals.Records = 1
	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)

	err = SeedAnchoringData(ctx, &upgrades.UpgradeKeepers{AnchoringKeeper: k}, exportDir, fixtureRegistriesJSON(t, "reg-a"), manifestJSON)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sha256_uncompressed mismatch")
}

func TestSeedAnchoringData_UnknownRegistryInManifestRejected(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx := newTestAnchoringKeeper(t)

	exportDir := t.TempDir()
	file := writeFixtureTranche(t, exportDir, "tranche-1/reg-a.jsonl.gz", 1, []fixtureRecord{
		{Registry: "reg-a", Checksum: "1 A 1", ChecksumAlgo: "cite-canonical-v1", Uri: "https://example.com/a1", Metadata: `{"case":"fixture"}`, Status: "Active"},
	})

	manifest := MigrationManifest{Files: []ManifestFileEntry{file}}
	manifest.Totals.Registries = 0 // no registries at all, but the file references "reg-a"
	manifest.Totals.Records = 1
	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)

	err = SeedAnchoringData(ctx, &upgrades.UpgradeKeepers{AnchoringKeeper: k}, exportDir, fixtureRegistriesJSON(t), manifestJSON)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown registry")
}

func TestSeedAnchoringData_DuplicateRegistryNameRejected(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx := newTestAnchoringKeeper(t)

	manifest := MigrationManifest{}
	manifest.Totals.Registries = 2
	manifest.Totals.Records = 0
	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)

	err = SeedAnchoringData(ctx, &upgrades.UpgradeKeepers{AnchoringKeeper: k}, t.TempDir(), fixtureRegistriesJSON(t, "reg-a", "reg-a"), manifestJSON)
	require.Error(t, err)
	require.Contains(t, err.Error(), `duplicate registry name "reg-a"`)
}

func TestSeedAnchoringData_MissingFileOnDiskRejected(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx := newTestAnchoringKeeper(t)

	manifest := MigrationManifest{Files: []ManifestFileEntry{{
		Registry: "reg-a",
		Records:  1,
		File:     "tranche-1/does-not-exist.jsonl.gz",
		Tranche:  1,
	}}}
	manifest.Totals.Registries = 1
	manifest.Totals.Records = 1
	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)

	err = SeedAnchoringData(ctx, &upgrades.UpgradeKeepers{AnchoringKeeper: k}, t.TempDir(), fixtureRegistriesJSON(t, "reg-a"), manifestJSON)
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected staged at")
}

func TestResolveExportDir(t *testing.T) {
	t.Run("default path is derived from home dir", func(t *testing.T) {
		t.Setenv("SOME_EXPORT_DIR_ENV", "")
		got := ResolveExportDir("/home/validator/.nvnmchain", "v1_2", "SOME_EXPORT_DIR_ENV")
		require.Equal(t, filepath.Join("/home/validator/.nvnmchain", "upgrades", "v1_2", "mainnet-full-export"), got)
	})

	t.Run("env var overrides default path", func(t *testing.T) {
		t.Setenv("SOME_EXPORT_DIR_ENV", "/mnt/export-data")
		got := ResolveExportDir("/home/validator/.nvnmchain", "v1_2", "SOME_EXPORT_DIR_ENV")
		require.Equal(t, "/mnt/export-data", got)
	})
}
