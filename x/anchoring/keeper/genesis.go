package keeper

import (
	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the tokenfactory module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) error {
	err := k.Params.Set(ctx, genState.Params)
	if err != nil {
		return err
	}

	registryCount := uint64(0)
	for _, registry := range genState.Registries {
		if err := k.Registries.Set(ctx, registry.Id, registry); err != nil {
			return err
		}
		// ensure name uniqueness
		if existingId, err := k.RegistryIdByName.Get(ctx, registry.Name); err == nil {
			return errorsmod.Wrapf(types.ErrRegistryExists, "duplicate registry name '%s' (ids: %d, %d)", registry.Name, existingId, registry.Id)
		}
		if err := k.RegistryIdByName.Set(ctx, registry.Name, registry.Id); err != nil {
			return err
		}
		if _, err := k.Roles.Get(ctx, collections.Join3(registry.Id, registry.Creator, "")); errorsmod.IsOf(err, collections.ErrNotFound) {
			if err := k.Roles.Set(ctx, collections.Join3(registry.Id, registry.Creator, ""), RoleAdmin); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if err := k.initializeRegistryRecordCounter(ctx, registry.Id); err != nil {
			return err
		}
		// update max registry id for registry count if genesis has non-sequential ids to avoid collisions
		if registry.Id > registryCount {
			registryCount = registry.Id
		}
	}
	if err := k.RegistryCount.Set(ctx, registryCount); err != nil {
		return err
	}

	for _, record := range genState.Records {
		registryId, err := k.RegistryIdByName.Get(ctx, record.Registry)
		if err != nil {
			return err
		}
		index, err := k.RecordIndices.Get(ctx, collections.Join(registryId, record.RecordId))
		if errorsmod.IsOf(err, collections.ErrNotFound) || record.Index > index {
			if err := k.RecordIndices.Set(ctx, collections.Join(registryId, record.RecordId), record.Index); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if err := k.Records.Set(ctx, collections.Join3(registryId, record.RecordId, record.Index), record); err != nil {
			return err
		}
		if err := k.RecordIdByRegistryAndChecksum.Set(ctx, collections.Join(registryId, record.Checksum), record.RecordId); err != nil {
			return err
		}
		if err := k.RecordIdByChecksumAndRegistry.Set(ctx, collections.Join(record.Checksum, registryId), record.RecordId); err != nil {
			return err
		}

		recordsCount, err := k.RecordsCountByRegistry.Get(ctx, registryId)
		if err != nil {
			return err
		}
		if record.RecordId > recordsCount {
			if err := k.RecordsCountByRegistry.Set(ctx, registryId, record.RecordId); err != nil {
				return err
			}
		}
	}

	for keyStr, role := range genState.Roles {
		_, key, err := k.Roles.KeyCodec().Decode([]byte(keyStr))
		if err != nil {
			return err
		}
		if err := k.Roles.Set(ctx, key, role); err != nil {
			return err
		}
	}

	return nil
}

// ExportGenesis returns the tokenfactory module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) (*types.GenesisState, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	genRegistries := map[uint64]types.Registry{}
	if err := k.Registries.Walk(ctx, nil, func(key uint64, registry types.Registry) (stop bool, err error) {
		genRegistries[key] = registry
		return false, nil
	}); err != nil {
		return nil, err
	}

	genRecords := []types.Record{}
	if err := k.Records.Walk(ctx, nil, func(_ collections.Triple[uint64, uint64, uint64], record types.Record) (stop bool, err error) {
		genRecords = append(genRecords, record)
		return false, nil
	}); err != nil {
		return nil, err
	}

	genRoles := map[string]string{}
	if err := k.Roles.Walk(ctx, nil, func(key collections.Triple[uint64, string, string], role string) (stop bool, err error) {
		keyBytes, err := encodeRoleKey(k, key)
		if err != nil {
			return true, err
		}
		genRoles[string(keyBytes)] = role
		return false, nil
	}); err != nil {
		return nil, err
	}

	return &types.GenesisState{
		Params:     params,
		Registries: genRegistries,
		Records:    genRecords,
		Roles:      genRoles,
	}, nil
}

// encodeRoleKey safely encodes a role key to bytes
func encodeRoleKey(k Keeper, key collections.Triple[uint64, string, string]) ([]byte, error) {
	size := k.Roles.KeyCodec().Size(key)
	buffer := make([]byte, size)
	n, err := k.Roles.KeyCodec().Encode(buffer, key)
	if err != nil {
		return nil, err
	}
	return buffer[:n], nil
}
