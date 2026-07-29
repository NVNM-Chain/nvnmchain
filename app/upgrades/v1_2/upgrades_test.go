package v1_2

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	appparams "github.com/NVNM-Chain/nvnmchain/app/params"
	"github.com/NVNM-Chain/nvnmchain/app/upgrades"
	"github.com/NVNM-Chain/nvnmchain/app/upgrades/v1_2/data"
	anchoringkeeper "github.com/NVNM-Chain/nvnmchain/x/anchoring/keeper"
	anchoringtypes "github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// newTestAnchoringKeeper mirrors testutil/keeper.AnchoringKeeper, duplicated here because that
// helper (transitively) imports the app package, which imports this package - reusing it would
// create an import cycle in this package's own tests.
func newTestAnchoringKeeper(t *testing.T) (anchoringkeeper.Keeper, sdk.Context) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(anchoringtypes.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())

	k := anchoringkeeper.NewKeeper(cdc, addressCodec, runtime.NewKVStoreService(storeKey))
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	require.NoError(t, k.Params.Set(ctx, anchoringtypes.DefaultParams()))
	require.NoError(t, k.RegistryCount.Set(ctx, 0))

	return k, ctx
}

// findRegistryIdByName scans the Registries collection for a registry with the given
// name and returns its id. The keeper no longer maintains a dedicated name index, so
// tests that create a registry by name must look its id up this way.
func findRegistryIdByName(t *testing.T, k anchoringkeeper.Keeper, ctx sdk.Context, name string) uint64 {
	t.Helper()
	var found uint64
	require.NoError(t, k.Registries.Walk(ctx, nil, func(id uint64, registry anchoringtypes.Registry) (bool, error) {
		if registry.Name == name {
			found = id
			return true, nil
		}
		return false, nil
	}))
	require.NotZero(t, found, "no registry found with name %q", name)
	return found
}

// emptyManifestJSON builds a manifest with no tranche files, matching wantRegistries — useful
// for exercising the registry-seeding half of SeedAnchoringData without needing any on-disk
// tranche data.
func emptyManifestJSON(t *testing.T, wantRegistries int) []byte {
	t.Helper()
	manifest := MigrationManifest{}
	manifest.Totals.Registries = wantRegistries
	manifest.Totals.Records = 0
	manifest.Files = []ManifestFileEntry{}
	b, err := json.Marshal(manifest)
	require.NoError(t, err)
	return b
}

// TestSeedAnchoringData_RealRegistriesJSON loads the actual embedded registries.json (all 4
// tranches, 2,114 registries) to catch data-level regressions (duplicate names, bad JSON,
// count drift) without needing the real tranche files on disk, and spot-checks one registry
// from tranches 1-3 and one from tranche 4 to confirm the merged file actually covers both.
func TestSeedAnchoringData_RealRegistriesJSON(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx := newTestAnchoringKeeper(t)

	blockTime := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(blockTime)

	var registries []RegistryImport
	require.NoError(t, json.Unmarshal(data.RegistriesJSON, &registries))

	err := SeedAnchoringData(ctx, &upgrades.UpgradeKeepers{AnchoringKeeper: k}, t.TempDir(), data.RegistriesJSON, emptyManifestJSON(t, len(registries)))
	require.NoError(t, err)

	count, err := k.RegistryCount.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(len(registries)), count)

	scotusId := findRegistryIdByName(t, k, ctx, "us-scotus")
	reg, err := k.Registries.Get(ctx, scotusId)
	require.NoError(t, err)
	require.Equal(t, blockTime.String(), reg.CreatedAt, "created_at must be stamped with upgrade block time, not exported value")
	require.Equal(t, MainnetRegistryAdmin, reg.Creator)

	admiraltyId := findRegistryIdByName(t, k, ctx, "us-admiraltyctsc")
	regAdmiralty, err := k.Registries.Get(ctx, admiraltyId)
	require.NoError(t, err)
	require.Equal(t, blockTime.String(), regAdmiralty.CreatedAt)
}

// TestSeedAnchoringData_RegistryManifestCountMismatchRejected guards against registries.json
// silently drifting from manifest.json (e.g. a bad edit to either file).
func TestSeedAnchoringData_RegistryManifestCountMismatchRejected(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx := newTestAnchoringKeeper(t)

	err := SeedAnchoringData(ctx, &upgrades.UpgradeKeepers{AnchoringKeeper: k}, t.TempDir(), data.RegistriesJSON, emptyManifestJSON(t, 1))
	require.Error(t, err)
	require.Contains(t, err.Error(), "manifest.json expects 1")
}

func TestExportDir(t *testing.T) {
	t.Run("default path is derived from home dir", func(t *testing.T) {
		t.Setenv(exportDirEnvVar, "")
		got := ResolveExportDir("/home/validator/.nvnmchain", "v1_2", exportDirEnvVar)
		require.Equal(t, filepath.Join("/home/validator/.nvnmchain", "upgrades", "v1_2", "mainnet-full-export"), got)
	})

	t.Run("env var overrides default path", func(t *testing.T) {
		t.Setenv(exportDirEnvVar, "/mnt/export-data")
		got := ResolveExportDir("/home/validator/.nvnmchain", "v1_2", exportDirEnvVar)
		require.Equal(t, "/mnt/export-data", got)
	})
}
