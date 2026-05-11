package v1_1

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NVNM-Chain/nvnmchain/app/upgrades"
)

const UpgradeName = "v1.1.0"

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *upgrades.UpgradeKeepers,
	_ map[string]*storetypes.KVStoreKey,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		toVM, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return toVM, err
		}
		if err := ResolveAnchoringFeeDenom(sdk.UnwrapSDKContext(ctx), keepers); err != nil {
			return toVM, err
		}
		return toVM, nil
	}
}

// ResolveAnchoringFeeDenom delegates to Params.ResolveAnchoringFeeDenom and
// writes back only if the substitution actually changed the denom.
func ResolveAnchoringFeeDenom(ctx sdk.Context, keepers *upgrades.UpgradeKeepers) error {
	params, err := keepers.AnchoringKeeper.Params.Get(ctx)
	if err != nil {
		return fmt.Errorf("get anchoring params: %w", err)
	}
	if params.AnchoringFee.Denom != "" {
		return nil
	}
	params.ResolveAnchoringFeeDenom()
	return keepers.AnchoringKeeper.Params.Set(ctx, params)
}

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades:        storetypes.StoreUpgrades{},
}
