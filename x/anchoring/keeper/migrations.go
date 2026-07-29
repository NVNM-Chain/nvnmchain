package keeper

import (
	v2 "github.com/NVNM-Chain/nvnmchain/x/anchoring/migrations/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrator is a wrapper around the anchoring keeper providing in-place store
// migrations for the module's consensus-version bumps.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator for the anchoring module.
func NewMigrator(k Keeper) Migrator {
	return Migrator{keeper: k}
}

// Migrate1to2 migrates the anchoring module's state from consensus version 1 to 2,
// backfilling Record.RegistryId on every existing record.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return v2.MigrateStore(ctx, m.keeper.storeService, m.keeper.cdc)
}
