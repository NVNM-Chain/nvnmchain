package keeper

import (
	"context"
	"fmt"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
)

// BackfillNameIndex walks every existing registry into k.NameIndex. Upsert is
// idempotent, so this is safe to call on every startup: it is a no-op past
// the first run (or the first run after a new registry range was added while
// the index was disabled), and it is what makes newly-enabled or
// freshly-state-synced nodes catch up without replaying blocks.
func (k Keeper) BackfillNameIndex(ctx context.Context) error {
	if k.NameIndex == nil {
		return nil
	}
	return k.Registries.Walk(ctx, nil, func(_ uint64, reg types.Registry) (bool, error) {
		if err := k.NameIndex.Upsert(&reg); err != nil {
			return true, fmt.Errorf("backfill registry %d: %w", reg.Id, err)
		}
		return false, nil
	})
}
