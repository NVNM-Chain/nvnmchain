package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k Keeper) AddRegistry(ctx sdk.Context, sender sdk.AccAddress, name, description, metadata string) (uint64, error) {
	if err := types.ValidateRegistryForCreate(name, description, metadata); err != nil {
		return 0, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}
	exists, err := k.RegistryIdByName.Has(ctx, name)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, types.ErrRegistryExists
	}
	registryCount, err := k.RegistryCount.Get(ctx)
	if err != nil {
		return 0, err
	}
	registryId := registryCount + 1
	registry := types.Registry{
		Id:          registryId,
		Name:        name,
		Description: description,
		Creator:     sender.String(),
		CreatedAt:   ctx.BlockTime().String(),
		Metadata:    metadata,
	}
	// Only admin or editor can add
	if err := k.Registries.Set(ctx, registryId, registry); err != nil {
		return 0, err
	}
	if err := k.RegistryIdByName.Set(ctx, name, registryId); err != nil {
		return 0, err
	}

	adminRole := k.RegistryRole(registryId, RoleAdmin)
	if err := k.RBAC.SetRoleAdmin(ctx, adminRole, adminRole); err != nil {
		return 0, err
	}
	creator, err := k.addressCodec.StringToBytes(sender.String())
	if err != nil {
		return 0, err
	}
	if err := k.RBAC.RoleMembers.Set(ctx, collections.Join(adminRole, sdk.AccAddress(creator)), []byte{}); err != nil {
		return 0, err
	}

	// set record count to zero
	if err := k.initializeRegistryRecordCounter(ctx, registryId); err != nil {
		return 0, err
	}
	// update registry count
	if err := k.RegistryCount.Set(ctx, registryId); err != nil {
		return 0, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeAddRegistry,
			sdk.NewAttribute(types.AttributeKeyRegistryId, fmt.Sprint(registryId)),
			sdk.NewAttribute(types.AttributeKeyRegistryName, fmt.Sprint(name)),
		),
	})

	return registryId, nil
}

func (k Keeper) AddRecord(ctx sdk.Context, sender sdk.AccAddress, record types.Record) (uint64, error) {
	if err := types.ValidateRecordForCreate(record); err != nil {
		return 0, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}

	registryId, err := k.RegistryIdByName.Get(ctx, record.Registry)
	if err != nil {
		return 0, err
	}

	// check permissions
	if err := k.checkPermission(ctx, sender, registryId, record.Checksum, []string{RoleAdmin, RoleEditor}); err != nil {
		return 0, err
	}

	recordId, err := k.RecordIdByRegistryAndChecksum.Get(ctx, collections.Join(registryId, record.Checksum))
	if errorsmod.IsOf(err, collections.ErrNotFound) {
		// add new recordId and update recordCount
		recordCount, err := k.RecordsCountByRegistry.Get(ctx, registryId)
		if err != nil {
			return 0, err
		}
		recordId = recordCount + 1
		if err := k.RecordsCountByRegistry.Set(ctx, registryId, recordId); err != nil {
			return 0, err
		}
		if err := k.RecordIdByChecksumAndRegistry.Set(ctx, collections.Join(record.Checksum, registryId), recordId); err != nil {
			return 0, err
		}
		if err := k.RecordIdByRegistryAndChecksum.Set(ctx, collections.Join(registryId, record.Checksum), recordId); err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	}
	record.RecordId = recordId

	// update fields for new record
	index, err := k.RecordIndices.Get(ctx, collections.Join(registryId, record.RecordId))
	switch {
	case errorsmod.IsOf(err, collections.ErrNotFound):
		index = 1
	case err != nil:
		return 0, err
	default:
		index++
	}
	record.Index = index
	if err := k.RecordIndices.Set(ctx, collections.Join(registryId, record.RecordId), index); err != nil {
		return 0, err
	}
	record.Timestamp = ctx.BlockTime().String()
	record.IsLatest = true

	// store the record
	if err := k.Records.Set(ctx, collections.Join3(registryId, record.RecordId, record.Index), record); err != nil {
		return 0, err
	}

	// update prev record of the same checksum to not latest
	if index > 1 {
		// Mark previous record as not latest
		prevRecord, err := k.Records.Get(ctx, collections.Join3(registryId, record.RecordId, record.Index-1))
		if err != nil {
			return 0, err
		}
		prevRecord.IsLatest = false
		if err := k.Records.Set(ctx, collections.Join3(registryId, record.RecordId, record.Index-1), prevRecord); err != nil {
			return 0, err
		}
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeAddRecord,
			sdk.NewAttribute(types.AttributeKeyRegistryId, fmt.Sprint(registryId)),
			sdk.NewAttribute(types.AttributeKeyRecordId, fmt.Sprint(recordId)),
			sdk.NewAttribute(types.AttributeKeyRecordIndex, fmt.Sprint(index)),
			sdk.NewAttribute(types.AttributeKeyRecordChecksum, record.Checksum),
		),
	})

	return recordId, nil
}

func (k Keeper) UpdateRecordStatus(ctx sdk.Context, sender sdk.AccAddress, registryId, recordId, index uint64, newStatus string) error {
	if err := types.ValidateRecordStatus(newStatus); err != nil {
		return sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}

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

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpdateRecordStatus,
			sdk.NewAttribute(types.AttributeKeyRegistryId, fmt.Sprint(registryId)),
			sdk.NewAttribute(types.AttributeKeyRecordId, fmt.Sprint(recordId)),
			sdk.NewAttribute(types.AttributeKeyRecordIndex, fmt.Sprint(index)),
			sdk.NewAttribute(types.AttributeKeyRecordStatus, newStatus),
			sdk.NewAttribute(types.AttributeKeyRecordChecksum, record.Checksum),
		),
	})

	return nil
}
