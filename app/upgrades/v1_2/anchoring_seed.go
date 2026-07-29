package v1_2

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/NVNM-Chain/nvnmchain/app/upgrades"
	anchoringtypes "github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// maxRecordLineBytes bounds a single record line, guarding against a bufio.Scanner
// "token too long" error if a future export contains an unusually large metadata blob.
const maxRecordLineBytes = 1 << 20 // 1 MiB

// MainnetRegistryAdmin is the Inveniam-designated mainnet address that becomes the permanent
// creator/admin of every registry loaded by a mainnet-full-export migration. Registry creator
// is set once and never updated by any handler (see AddRegistry) — getting this wrong
// permanently freezes every registry loaded with it, so it is deliberately hardcoded here
// rather than sourced from the writer-supplied export data (registries.json has no creator
// field at all, precisely to prevent that).
const MainnetRegistryAdmin = "nvnm14a3em3mr9mvta9ccgk80wn0dxgzt5lkt2r8trx"

// ResolveExportDir resolves the local directory that must contain a mainnet-full-export
// tranche subset's tranche-*/*.jsonl.gz files at upgrade time. These files are large and
// gitignored — never embedded in the binary — so every validator's operator must independently
// stage the export (verified against an embedded manifest.json by SeedAnchoringData) at this
// path before the upgrade height is reached. envVar, if set on the node process, overrides the
// default path of <homeDir>/upgrades/<upgradeSubdir>/mainnet-full-export.
func ResolveExportDir(homeDir, upgradeSubdir, envVar string) string {
	if override := os.Getenv(envVar); override != "" {
		return override
	}
	return filepath.Join(homeDir, "upgrades", upgradeSubdir, "mainnet-full-export")
}

// RegistryImport mirrors the writer-supplied fields of a mainnet-full-export registries.json.
// Id, CreatedAt and Creator are intentionally absent: the registry id continues from the
// chain's own RegistryCount, CreatedAt is stamped with the upgrade's block time (both via
// AddRegistry), and Creator is always MainnetRegistryAdmin (see its doc comment).
type RegistryImport struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Metadata    string `json:"metadata"`
	Tranche     int    `json:"tranche"`
}

// recordImport mirrors the writer-supplied fields of a mainnet-full-export tranche jsonl line.
// Timestamp, RecordId, Index and IsLatest are chain-assigned and are omitted here: AddRecord
// recomputes all four (Timestamp from the upgrade's block time) regardless of any submitted value.
type recordImport struct {
	Registry     string `json:"registry"`
	Uri          string `json:"uri"`
	Checksum     string `json:"checksum"`
	ChecksumAlgo string `json:"checksumAlgo"`
	Metadata     string `json:"metadata"`
	Status       string `json:"status"`
}

// MigrationManifest mirrors a mainnet-full-export manifest.json: per-tranche-file record
// counts and sha256 digests used to verify the disk-provided export before any of it is
// written to state.
type MigrationManifest struct {
	Totals struct {
		Registries int `json:"registries"`
		Records    int `json:"records"`
	} `json:"totals"`
	Files []ManifestFileEntry `json:"files"`
}

type ManifestFileEntry struct {
	Registry           string `json:"registry"`
	Records            int    `json:"records"`
	File               string `json:"file"`
	Tranche            int    `json:"tranche"`
	Sha256Gz           string `json:"sha256_gz"`
	Sha256Uncompressed string `json:"sha256_uncompressed"`
}

// SeedAnchoringData creates every registriesJSON registry (admin: MainnetRegistryAdmin) and
// then, for each tranche file listed in manifestJSON, reads the corresponding .gz from
// exportDir, verifies it against the manifest's sha256 digests, and writes its records through
// the anchoring keeper — so registry ids continue from the chain's current RegistryCount and
// created_at/timestamp are stamped with the upgrade block time. registriesJSON and manifestJSON
// are passed in (rather than read directly from an embed) so tests can exercise this against
// small fixtures instead of a real mainnet-scale export.
func SeedAnchoringData(ctx sdk.Context, keepers *upgrades.UpgradeKeepers, exportDir string, registriesJSON, manifestJSON []byte) error {
	var registries []RegistryImport
	if err := json.Unmarshal(registriesJSON, &registries); err != nil {
		return fmt.Errorf("failed to unmarshal registries.json: %w", err)
	}

	var manifest MigrationManifest
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		return fmt.Errorf("failed to unmarshal manifest.json: %w", err)
	}
	if len(registries) != manifest.Totals.Registries {
		return fmt.Errorf("registries.json has %d registries, manifest.json expects %d", len(registries), manifest.Totals.Registries)
	}

	adminAddr, err := sdk.AccAddressFromBech32(MainnetRegistryAdmin)
	if err != nil {
		return fmt.Errorf("invalid MainnetRegistryAdmin address %q: %w", MainnetRegistryAdmin, err)
	}

	ctx.Logger().Info(fmt.Sprintf("Seeding %d anchoring registries...", len(registries)))
	registryIds := make(map[string]uint64, len(registries))
	for _, r := range registries {
		registryId, err := keepers.AnchoringKeeper.AddRegistry(ctx, adminAddr, r.Name, r.Description, r.Metadata)
		if err != nil {
			return fmt.Errorf("registry %q: %w", r.Name, err)
		}
		registryIds[r.Name] = registryId
	}
	ctx.Logger().Info(fmt.Sprintf("Seeded %d anchoring registries", len(registries)))

	files := append([]ManifestFileEntry(nil), manifest.Files...)
	sort.Slice(files, func(i, j int) bool {
		if files[i].Tranche != files[j].Tranche {
			return files[i].Tranche < files[j].Tranche
		}
		return files[i].Registry < files[j].Registry
	})

	recordCount := 0
	currentTranche := 0
	for _, fileEntry := range files {
		if fileEntry.Tranche != currentTranche {
			currentTranche = fileEntry.Tranche
			ctx.Logger().Info(fmt.Sprintf("Loading tranche %d records...", currentTranche))
		}

		registryId, ok := registryIds[fileEntry.Registry]
		if !ok {
			return fmt.Errorf("manifest file %q: unknown registry %q", fileEntry.File, fileEntry.Registry)
		}

		fileRecordCount, err := loadTrancheFile(ctx, keepers, exportDir, adminAddr, registryId, fileEntry)
		if err != nil {
			return err
		}
		recordCount += fileRecordCount
	}

	if recordCount != manifest.Totals.Records {
		return fmt.Errorf("seeded %d records, manifest.json expects %d", recordCount, manifest.Totals.Records)
	}

	ctx.Logger().Info(fmt.Sprintf("Seeded %d anchoring records", recordCount))
	return nil
}

// loadTrancheFile reads, hash-verifies, decompresses and writes a single manifest-listed
// tranche file's records, returning the number of records written. registryId is the id
// (already created by SeedAnchoringData) of the registry this file's records belong to.
func loadTrancheFile(ctx sdk.Context, keepers *upgrades.UpgradeKeepers, exportDir string, adminAddr sdk.AccAddress, registryId uint64, fileEntry ManifestFileEntry) (int, error) {
	path := filepath.Join(exportDir, fileEntry.File)
	gzBytes, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read %s (expected staged at %s before the upgrade height): %w", fileEntry.File, exportDir, err)
	}
	if gotHash := sha256Hex(gzBytes); gotHash != fileEntry.Sha256Gz {
		return 0, fmt.Errorf("%s: sha256_gz mismatch: manifest expects %s, got %s", fileEntry.File, fileEntry.Sha256Gz, gotHash)
	}

	gzReader, err := gzip.NewReader(bytes.NewReader(gzBytes))
	if err != nil {
		return 0, fmt.Errorf("%s: failed to open gzip stream: %w", fileEntry.File, err)
	}
	content, err := io.ReadAll(gzReader)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to decompress: %w", fileEntry.File, err)
	}
	if err := gzReader.Close(); err != nil {
		return 0, fmt.Errorf("%s: gzip stream error: %w", fileEntry.File, err)
	}
	if gotHash := sha256Hex(content); gotHash != fileEntry.Sha256Uncompressed {
		return 0, fmt.Errorf("%s: sha256_uncompressed mismatch: manifest expects %s, got %s", fileEntry.File, fileEntry.Sha256Uncompressed, gotHash)
	}

	lineNum := 0
	fileRecordCount := 0
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), maxRecordLineBytes)
	for scanner.Scan() {
		lineNum++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var rec recordImport
		if err := json.Unmarshal(line, &rec); err != nil {
			return 0, fmt.Errorf("%s line %d: %w", fileEntry.File, lineNum, err)
		}
		if rec.Registry != fileEntry.Registry {
			return 0, fmt.Errorf("%s line %d: record registry %q does not match manifest registry %q", fileEntry.File, lineNum, rec.Registry, fileEntry.Registry)
		}

		_, err := keepers.AnchoringKeeper.AddRecord(ctx, adminAddr, anchoringtypes.Record{
			RegistryId:   registryId,
			Uri:          rec.Uri,
			Checksum:     rec.Checksum,
			ChecksumAlgo: rec.ChecksumAlgo,
			Metadata:     rec.Metadata,
			Status:       rec.Status,
		})
		if err != nil {
			return 0, fmt.Errorf("%s line %d (registry %q, checksum %q): %w", fileEntry.File, lineNum, rec.Registry, rec.Checksum, err)
		}
		fileRecordCount++
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed to read %s: %w", fileEntry.File, err)
	}

	if fileRecordCount != fileEntry.Records {
		return 0, fmt.Errorf("%s: wrote %d records, manifest expects %d", fileEntry.File, fileRecordCount, fileEntry.Records)
	}

	return fileRecordCount, nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
