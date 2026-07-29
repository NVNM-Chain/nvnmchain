package v1_2

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/NVNM-Chain/nvnmchain/app/upgrades"
	"github.com/NVNM-Chain/nvnmchain/app/upgrades/v1_2/data"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// exportDirEnvVar optionally overrides the on-disk location of this upgrade's tranche data
// (all 4 tranches of the mainnet-full-export). Unset by default; only needed if an operator
// stages the export somewhere other than the default path under the node's home directory.
const exportDirEnvVar = "NVNMCHAIN_V1_2_EXPORT_DIR"

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *upgrades.UpgradeKeepers,
	storekeys map[string]*storetypes.KVStoreKey,
	homeDir string,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)
		ctx.Logger().Info("Starting v1.2.0 upgrade...")

		ctx.Logger().Info("Running module migrations...")
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		exportDir := ResolveExportDir(homeDir, "v1_2", exportDirEnvVar)
		if err := SeedAnchoringData(ctx, keepers, exportDir, data.RegistriesJSON, data.ManifestJSON); err != nil {
			return vm, fmt.Errorf("failed to seed anchoring data: %w", err)
		}

		ctx.Logger().Info("Upgrade v1.2.0 complete")
		return vm, nil
	}
}
