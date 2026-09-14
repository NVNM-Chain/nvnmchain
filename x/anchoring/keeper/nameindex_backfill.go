package keeper

import (
	"context"
	"fmt"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
)

// BackfillNameIndex walks every existing registry into k.NameIndex in one
// transaction. Upsert is idempotent, so it runs on every start; that is what
// catches the index up after it was enabled late, after a block whose
// ListenCommit failed, or after a restart following state sync.
//
// State sync applies its snapshot after the app is constructed, so a freshly
// synced node indexes the snapshot's registries on its next restart;
// registries created after the sync are indexed by the listener as they
// commit.
func (k Keeper) BackfillNameIndex(ctx context.Context) error {
	if k.NameIndex == nil {
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
