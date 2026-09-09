package nameindex

import (
	"bytes"
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
)

// Listener is a storetypes.ABCIListener that keeps a Store in sync with the
// anchoring module's Registries collection. It only ever observes state that
// has already been committed, so a rolled-back tx (e.g. a later msg in the
// same tx failing) never reaches the index.
type Listener struct {
	store *Store
}

var _ storetypes.ABCIListener = (*Listener)(nil)

func NewListener(store *Store) *Listener {
	return &Listener{store: store}
}

// ListenFinalizeBlock is unused: the registries collection is only read from
// the finalized changeset delivered to ListenCommit.
func (l *Listener) ListenFinalizeBlock(_ context.Context, _ abci.RequestFinalizeBlock, _ abci.ResponseFinalizeBlock) error {
	return nil
}

// ListenCommit indexes every Registry write in the just-committed changeset.
func (l *Listener) ListenCommit(_ context.Context, _ abci.ResponseCommit, changeSet []*storetypes.StoreKVPair) error {
	for _, pair := range changeSet {
		if pair.StoreKey != types.StoreKey || pair.Delete {
			continue
		}
		if !bytes.HasPrefix(pair.Key, types.RegistriesKeyPrefix) {
			continue
		}

		reg := &types.Registry{}
		if err := l.store.cdc.Unmarshal(pair.Value, reg); err != nil {
			return fmt.Errorf("nameindex: decode registry write: %w", err)
		}
		if err := l.store.Upsert(reg); err != nil {
			return err
		}
	}
	return nil
}
