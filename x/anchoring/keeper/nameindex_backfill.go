package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
)

// BackfillNameIndex brings k.NameIndex up to date with the Registries
// collection. Registries are add-only with dense ids, so an index holding as
// many rows as RegistryCount is complete and nothing is read. Otherwise every
// registry is upserted in one transaction, which is what catches the index up
// after it was enabled late, after a block whose ListenCommit failed, or on
// the restart after state sync (the snapshot lands after this runs).
func (k Keeper) BackfillNameIndex(ctx context.Context) error {
	if k.NameIndex == nil {
		return nil
	}
	total, err := k.RegistryCount.Get(ctx)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	indexed, err := k.NameIndex.Count()
	if err != nil {
		return err
	}
	if indexed == total {
		return nil
	}

	batch, err := k.NameIndex.Begin()
	if err != nil {
		return err
	}
	defer batch.Close()

	err = k.Registries.Walk(ctx, nil, func(_ uint64, reg types.Registry) (bool, error) {
		if err := batch.Upsert(&reg); err != nil {
			return true, fmt.Errorf("backfill registry %d: %w", reg.Id, err)
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	return batch.Commit()
}
