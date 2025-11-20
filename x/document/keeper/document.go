package keeper

import (
	"cosmossdk.io/collections"
	"github.com/MANTRA-Chain/inveniam/x/document/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) setDocument(ctx sdk.Context, doc types.Document) error {
	hasCounter, err := k.DocumentCounters.Has(ctx, doc.Denom)
	if err != nil {
		return err
	}
	var index uint64
	if !hasCounter {
		err = k.DocumentCounters.Set(ctx, doc.Denom, 1)
		if err != nil {
			return err
		}
		index = 1
	} else {
		counter, err := k.DocumentCounters.Get(ctx, doc.Denom)
		if err != nil {
			return err
		}
		index = counter + 1
		err = k.DocumentCounters.Set(ctx, doc.Denom, index)
		if err != nil {
			return err
		}
	}

	if err := k.Documents.Set(ctx, collections.Join(doc.Denom, index), doc); err != nil {
		return err
	}
	return nil
}

// func (k Keeper) GetDocumentDenomPrefixStore(ctx sdk.Context, denom string) prefix.Store {
// 	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
// 	return prefix.NewStore(store, types.DocumentKeyByDenom(denom))
// }

// func (k Keeper) GetAllDocumentsPrefixStore(ctx sdk.Context) prefix.Store {
// 	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
// 	return prefix.NewStore(store, types.DocumentKeyPrefix)
// }

// func (k Keeper) GetAllDocumentsIterator(ctx sdk.Context) storetypes.Iterator {
// 	return k.GetAllDocumentsPrefixStore(ctx).Iterator(nil, nil)
// }
