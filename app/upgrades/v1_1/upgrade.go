package v1_1

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NVNM-Chain/nvnmchain/app/upgrades"
	anchoringtypes "github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
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
		if err := ResolveAnchoringFee(sdk.UnwrapSDKContext(ctx), keepers); err != nil {
			return toVM, err
		}
		return toVM, nil
	}
}

// ResolveAnchoringFee fills any missing field on Params.AnchoringFee: the denom
// (via Params.ResolveAnchoringFeeDenom) and the amount (via DefaultAnchoringFeeAmount
// when nil or non-positive). Chains upgrading from a pre-anchoring_fee version
// have a zero-value Coin after proto unmarshal — both fields must be healed,
// not just the denom, or the refund path panics on the nil amount.
func ResolveAnchoringFee(ctx sdk.Context, keepers *upgrades.UpgradeKeepers) error {
	params, err := keepers.AnchoringKeeper.Params.Get(ctx)
	if err != nil {
		return fmt.Errorf("get anchoring params: %w", err)
	}
	changed := false
	if params.AnchoringFee.Denom == "" {
		params.ResolveAnchoringFeeDenom()
		changed = true
	}
	if params.AnchoringFee.Amount.IsNil() || !params.AnchoringFee.Amount.IsPositive() {
		params.AnchoringFee.Amount = anchoringtypes.DefaultAnchoringFeeAmount
		changed = true
	}
	if !changed {
		return nil
	}
	return keepers.AnchoringKeeper.Params.Set(ctx, params)
}

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades:        storetypes.StoreUpgrades{},
}
