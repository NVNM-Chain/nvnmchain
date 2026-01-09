package keeper

import (
	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) AddRecord(ctx sdk.Context, sender sdk.AccAddress, record types.Record) error {
	registryId, err := k.RegistryIdByName.Get(ctx, record.Registry)
	if err != nil {
		return err
	}

	// check permissions
	if err := k.checkPermission(ctx, sender, registryId, record.Checksum, []string{RoleAdmin, RoleEditor}); err != nil {
		return err
	}

	recordId, err := k.RecordIdByRegistryAndChecksum.Get(ctx, collections.Join(registryId, record.Checksum))
	if errorsmod.IsOf(err, collections.ErrNotFound) {
		// add new recordId and update recordCount
		recordCount, err := k.RecordsCountByRegistry.Get(ctx, registryId)
		if err != nil {
			return err
		}
		recordId = recordCount + 1
		if err := k.RecordsCountByRegistry.Set(ctx, registryId, recordId); err != nil {
			return err
		}
		if err := k.RecordIdByChecksumAndRegistry.Set(ctx, collections.Join(record.Checksum, registryId), recordId); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	record.RecordId = recordId

	// update fields for new record
	index, err := k.RecordIndices.Get(ctx, collections.Join(registryId, record.RecordId))
	switch {
	case errorsmod.IsOf(err, collections.ErrNotFound):
		index = 1
	case err != nil:
		return err
	default:
		index++
	}
	record.Index = index
	if err := k.RecordIndices.Set(ctx, collections.Join(registryId, record.RecordId), index); err != nil {
		return err
	}
	record.Timestamp = ctx.BlockTime().String()
	record.IsLatest = true

	// store the record
	if err := k.Records.Set(ctx, collections.Join3(registryId, record.RecordId, record.Index), record); err != nil {
		return err
	}
	if err := k.RecordIdByRegistryAndChecksum.Set(ctx, collections.Join(registryId, record.Checksum), record.RecordId); err != nil {
		return err
	}

	// update prev record of the same checksum to not latest
	if index > 1 {
		// Mark previous record as not latest
		prevRecord, err := k.Records.Get(ctx, collections.Join3(registryId, record.RecordId, record.Index-1))
		if err != nil {
			return err
		}
		prevRecord.IsLatest = false
		if err := k.Records.Set(ctx, collections.Join3(registryId, record.RecordId, record.Index-1), prevRecord); err != nil {
			return err
		}
	}

	return nil
}

func (k Keeper) UpdateRecordStatus(ctx sdk.Context, sender sdk.AccAddress, registryId, recordId, index uint64, newStatus string) error {
	// get the record
	record, err := k.Records.Get(ctx, collections.Join3(registryId, recordId, index))
	if err != nil {
		return err
	}
	// check permissions
	if err := k.checkPermission(ctx, sender, registryId, record.Checksum, []string{RoleAdmin, RoleEditor}); err != nil {
		return err
	}
	record.Status = newStatus
	if err := k.Records.Set(ctx, collections.Join3(registryId, recordId, index), record); err != nil {
		return err
	}
	return nil
}
