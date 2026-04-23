package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/store"
	"github.com/MANTRA-Chain/nvnmchain/x/anchoring/rbac"
	"github.com/MANTRA-Chain/nvnmchain/x/anchoring/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		addressCodec address.Codec
		storeService store.KVStoreService
		Schema       collections.Schema
		Params       collections.Item[types.Params]
		// registryId => registry
		Registries collections.Map[uint64, types.Registry]
		// registry name => registryId
		RegistryIdByName collections.Map[string, uint64]
		// total number of registries
		RegistryCount collections.Item[uint64]
		// document registryId + recordId + index => record
		Records collections.Map[collections.Triple[uint64, uint64, uint64], types.Record]
		// registryId + document checksum => recordId
		RecordIdByRegistryAndChecksum collections.Map[collections.Pair[uint64, string], uint64]
		// registryId => record count
		RecordsCountByRegistry collections.Map[uint64, uint64]
		// registryId + recordId => latest record index
		RecordIndices collections.Map[collections.Pair[uint64, uint64], uint64]
		// document checksum + registryId => recordId
		RecordIdByChecksumAndRegistry collections.Map[collections.Pair[string, uint64], uint64]
		// registryId + user address + document checksum => role
		// checksum is omitted for registry level roles
		Roles collections.Map[collections.Triple[uint64, string, string], string]
		RBAC  rbac.RBACKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	addressCodec address.Codec,
	storeService store.KVStoreService,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:          cdc,
		storeService: storeService,
		addressCodec: addressCodec,
		Params:       collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		Registries: collections.NewMap(
			sb,
			types.RegistriesKeyPrefix,
			"registries",
			collections.Uint64Key,
			codec.CollValue[types.Registry](cdc),
		),
		RegistryIdByName: collections.NewMap(
			sb,
			types.RegistryIdByNameKeyPrefix,
			"registry_id_by_name",
			collections.StringKey,
			collections.Uint64Value,
		),
		RegistryCount: collections.NewItem(
			sb,
			types.RegistryCountKey,
			"registry_count",
			collections.Uint64Value,
		),
		Records: collections.NewMap(
			sb,
			types.RecordsKeyPrefix,
			"records",
			collections.TripleKeyCodec(collections.Uint64Key, collections.Uint64Key, collections.Uint64Key),
			codec.CollValue[types.Record](cdc),
		),
		RecordIdByRegistryAndChecksum: collections.NewMap(
			sb,
			types.RecordIdByRegistryAndChecksumKeyPrefix,
			"record_id_by_registry_and_checksum",
			collections.PairKeyCodec(collections.Uint64Key, collections.StringKey),
			collections.Uint64Value,
		),
		RecordsCountByRegistry: collections.NewMap(
			sb,
			types.RecordsCountByRegistryIdKeyPrefix,
			"records_count_by_registry_id",
			collections.Uint64Key,
			collections.Uint64Value,
		),
		RecordIndices: collections.NewMap(
			sb,
			types.RecordIndicesKeyPrefix,
			"record_indices",
			collections.PairKeyCodec(collections.Uint64Key, collections.Uint64Key),
			collections.Uint64Value,
		),
		RecordIdByChecksumAndRegistry: collections.NewMap(
			sb,
			types.RecordIdByChecksumAndRegistryKeyPrefix,
			"record_id_by_checksum_and_registry",
			collections.PairKeyCodec(collections.StringKey, collections.Uint64Key),
			collections.Uint64Value,
		),
		Roles: collections.NewMap(
			sb,
			types.RolesKeyPrefix,
			"roles",
			collections.TripleKeyCodec(
				collections.Uint64Key,
				collections.StringKey,
				collections.StringKey,
			),
			collections.StringValue,
		),
		RBAC: rbac.NewRBACKeeper(
			sb,
			types.RoleAdminsKeyPrefix,
			types.RoleMembersKeyPrefix,
		),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

func (k Keeper) initializeRegistryRecordCounter(ctx sdk.Context, registryId uint64) error {
	return k.RecordsCountByRegistry.Set(ctx, registryId, 0)
}

// checkRecordRoles checks if sender has any of the specified roles at the record level.
// Record-level roles are scoped by registry and checksum.
func (k Keeper) checkRecordRoles(ctx sdk.Context, sender sdk.AccAddress, registryId uint64, checksum string, roles []string) (bool, error) {
	if checksum == "" || len(roles) == 0 {
		return false, nil
	}
	for _, roleStr := range roles {
		recordRole := k.RecordRole(registryId, checksum, roleStr)
		hasRole, err := k.RBAC.HasRole(ctx, recordRole, sender)
		if err != nil {
			return false, err
		}
		if hasRole {
			return true, nil
		}
	}
	return false, nil
}

// checkRegistryRoles checks if sender has any of the specified roles at the registry level
func (k Keeper) checkRegistryRoles(ctx sdk.Context, sender sdk.AccAddress, registryId uint64, roles []string) (bool, error) {
	if len(roles) == 0 {
		return false, nil
	}
	for _, roleStr := range roles {
		registryRole := k.RegistryRole(registryId, roleStr)
		hasRole, err := k.RBAC.HasRole(ctx, registryRole, sender)
		if err != nil {
			return false, err
		}
		if hasRole {
			return true, nil
		}
	}
	return false, nil
}

func (k Keeper) checkPermission(ctx sdk.Context, sender sdk.AccAddress, registryId uint64, checksum string, requiredRoles []string) error {
	hasRecordRole, err := k.checkRecordRoles(ctx, sender, registryId, checksum, requiredRoles)
	if err != nil {
		return err
	}
	if hasRecordRole {
		return nil
	}

	hasRegRole, err := k.checkRegistryRoles(ctx, sender, registryId, requiredRoles)
	if err != nil {
		return err
	}
	if hasRegRole {
		return nil
	}

	return sdkerrors.ErrUnauthorized
}

// RecordRole creates a role identifier for a specific record.
// User-controlled fields are hex-encoded to keep separators unambiguous.
func (k Keeper) RecordRole(registryId uint64, checksum string, role string) rbac.Role {
	return rbac.NewRoleHash(fmt.Sprintf("record:%d:%x:%x", registryId, checksum, role))
}

// RegistryRole creates a role identifier for a registry: hash("registry:id", role)
func (k Keeper) RegistryRole(registryId uint64, role string) rbac.Role {
	return rbac.NewRoleHash(fmt.Sprintf("registry:%d:%s", registryId, role))
}

// BytesToString converts bytes to a string address using the keeper's address codec
func (k Keeper) BytesToString(account []byte) (string, error) {
	return k.addressCodec.BytesToString(account)
}
