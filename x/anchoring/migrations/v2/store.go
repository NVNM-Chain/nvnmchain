package v2

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// recordsCollection builds the same Records collection the keeper uses (same prefix,
// name, and key/value codecs), independent of the keeper package, so this migration
// stays valid regardless of future keeper changes.
func recordsCollection(storeService store.KVStoreService, cdc codec.BinaryCodec) (collections.Map[collections.Triple[uint64, uint64, uint64], types.Record], error) {
	sb := collections.NewSchemaBuilder(storeService)
	records := collections.NewMap(
		sb,
		types.RecordsKeyPrefix,
		"records",
		collections.TripleKeyCodec(collections.Uint64Key, collections.Uint64Key, collections.Uint64Key),
		codec.CollValue[types.Record](cdc),
	)
	if _, err := sb.Build(); err != nil {
		return collections.Map[collections.Triple[uint64, uint64, uint64], types.Record]{}, err
	}
	return records, nil
}

// registryIdByNameCollection rebuilds the retired registry_id_by_name index (the
// keeper no longer maintains it) purely so this migration can locate and clear its
// entries from existing chain state.
func registryIdByNameCollection(storeService store.KVStoreService) (collections.Map[string, uint64], error) {
	sb := collections.NewSchemaBuilder(storeService)
	m := collections.NewMap(
		sb,
		types.RegistryIdByNameKeyPrefix,
		"registry_id_by_name",
		collections.StringKey,
		collections.Uint64Value,
	)
	if _, err := sb.Build(); err != nil {
		return collections.Map[string, uint64]{}, err
	}
	return m, nil
}

// MigrateStore backfills Record.RegistryId from the record's own KVStore key
// (registryId, recordId, index) for every existing record, and clears the retired
// registry_id_by_name index. This is the x/anchoring v1->v2 store migration: records
// created before Record carried an explicit registry_id field only had a registry
// name, resolved to an id at write time and never persisted; the registry_id_by_name
// index existed solely to support that name-based resolution and is no longer used.
func MigrateStore(ctx sdk.Context, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	records, err := recordsCollection(storeService, cdc)
	if err != nil {
		return err
	}

	var keys []collections.Triple[uint64, uint64, uint64]
	if err := records.Walk(ctx, nil, func(key collections.Triple[uint64, uint64, uint64], _ types.Record) (bool, error) {
		keys = append(keys, key)
		return false, nil
	}); err != nil {
		return err
	}

	for _, key := range keys {
		record, err := records.Get(ctx, key)
		if err != nil {
			return err
		}
		record.RegistryId = key.K1()
		if err := records.Set(ctx, key, record); err != nil {
			return err
		}
	}
	ctx.Logger().Info("anchoring: migrated records to registry_id", "count", len(keys))

	registryIdByName, err := registryIdByNameCollection(storeService)
	if err != nil {
		return err
	}
	if err := registryIdByName.Clear(ctx, nil); err != nil {
		return err
	}
	ctx.Logger().Info("anchoring: cleared retired registry_id_by_name index")

	return nil
}
