package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/rbac"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the anchoring module's state from a provided genesis
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

	type versionGroupKey struct {
		registryID uint64
		recordID   uint64
	}
	groups := make(map[versionGroupKey]*types.RecordVersionGroup)

	for i, record := range genState.Records {
		if err := types.ValidateRecord(record); err != nil {
			return errorsmod.Wrapf(err, "invalid genesis record (registry=%s, record_id=%d, index=%d, checksum=%s, position=%d)", record.Registry, record.RecordId, record.Index, record.Checksum, i)
		}

		registryId, err := k.RegistryIdByName.Get(ctx, record.Registry)
		if err != nil {
			return err
		}

		recordKey := collections.Join3(registryId, record.RecordId, record.Index)
		hasRecord, err := k.Records.Has(ctx, recordKey)
		if err != nil {
			return err
		}
		if hasRecord {
			return errorsmod.Wrapf(types.ErrDuplicateRecordKey, "duplicate record key (registry_id=%d, record_id=%d, index=%d)", registryId, record.RecordId, record.Index)
		}

		index, err := k.RecordIndices.Get(ctx, collections.Join(registryId, record.RecordId))
		if errorsmod.IsOf(err, collections.ErrNotFound) || record.Index > index {
			if err := k.RecordIndices.Set(ctx, collections.Join(registryId, record.RecordId), record.Index); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if err := k.Records.Set(ctx, recordKey, record); err != nil {
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

		gk := versionGroupKey{registryId, record.RecordId}
		if groups[gk] == nil {
			groups[gk] = &types.RecordVersionGroup{}
		}
		groups[gk].Add(record)
	}
	for gk, g := range groups {
		label := fmt.Sprintf("record group (registry_id=%d, record_id=%d)", gk.registryID, gk.recordID)
		if err := g.ValidateIsLatest(label); err != nil {
			return errorsmod.Wrapf(types.ErrInvalidGenesisState, "%s", err)
		}
	}

	for keyStr, adminRole := range genState.RoleAdmins {
		_, key, err := k.RBAC.RoleAdmins.KeyCodec().Decode([]byte(keyStr))
		if err != nil {
			return err
		}
		if err := k.RBAC.RoleAdmins.Set(ctx, key, adminRole); err != nil {
			return err
		}
	}

	for keyStr, marker := range genState.RoleMembers {
		_, key, err := k.RBAC.RoleMembers.KeyCodec().Decode([]byte(keyStr))
		if err != nil {
			return err
		}
		if err := k.RBAC.RoleMembers.Set(ctx, key, marker); err != nil {
			return err
		}
	}

	return nil
}

// ExportGenesis returns the anchoring module's exported genesis.
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

	genRoleAdmins := map[string][]byte{}
	if err := k.RBAC.RoleAdmins.Walk(ctx, nil, func(key rbac.Role, adminRole []byte) (stop bool, err error) {
		keyBytes, err := encodeRBACRoleAdminKey(k, key)
		if err != nil {
			return true, err
		}
		genRoleAdmins[string(keyBytes)] = adminRole
		return false, nil
	}); err != nil {
		return nil, err
	}

	genRoleMembers := map[string][]byte{}
	if err := k.RBAC.RoleMembers.Walk(ctx, nil, func(key collections.Pair[rbac.Role, sdk.AccAddress], marker []byte) (stop bool, err error) {
		keyBytes, err := encodeRBACRoleMemberKey(k, key)
		if err != nil {
			return true, err
		}
		genRoleMembers[string(keyBytes)] = marker
		return false, nil
	}); err != nil {
		return nil, err
	}

	return &types.GenesisState{
		Params:      params,
		Registries:  genRegistries,
		Records:     genRecords,
		RoleAdmins:  genRoleAdmins,
		RoleMembers: genRoleMembers,
	}, nil
}

// encodeRBACRoleAdminKey safely encodes an RBAC role-admin key to bytes.
func encodeRBACRoleAdminKey(k Keeper, key rbac.Role) ([]byte, error) {
	size := k.RBAC.RoleAdmins.KeyCodec().Size(key)
	buffer := make([]byte, size)
	n, err := k.RBAC.RoleAdmins.KeyCodec().Encode(buffer, key)
	if err != nil {
		return nil, err
	}
	return buffer[:n], nil
}

// encodeRBACRoleMemberKey safely encodes an RBAC role-member key to bytes.
func encodeRBACRoleMemberKey(k Keeper, key collections.Pair[rbac.Role, sdk.AccAddress]) ([]byte, error) {
	size := k.RBAC.RoleMembers.KeyCodec().Size(key)
	buffer := make([]byte, size)
	n, err := k.RBAC.RoleMembers.KeyCodec().Encode(buffer, key)
	if err != nil {
		return nil, err
	}
	return buffer[:n], nil
}
