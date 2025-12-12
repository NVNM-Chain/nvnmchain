package keeper

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/store"
	"github.com/MANTRA-Chain/inveniam/x/document/types"
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
		cdc                 codec.BinaryCodec
		addressCodec        address.Codec
		storeService        store.KVStoreService
		Schema              collections.Schema
		Params              collections.Item[types.Params]
		DocumentsByDenom    collections.Map[collections.Pair[string, uint64], types.Document]
		DocumentsByChecksum collections.Map[string, types.Document]
		DocumentCounters    collections.Map[string, uint64]
		Roles               collections.Map[string, string]
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
		DocumentsByDenom: collections.NewMap(
			sb,
			types.DocumentByDenomKeyPrefix,
			"documents_by_denom",
			collections.PairKeyCodec(collections.StringKey, collections.Uint64Key),
			codec.CollValue[types.Document](cdc),
		),
		DocumentsByChecksum: collections.NewMap(
			sb,
			types.DocumentByChecksumKey,
			"documents_by_checksum",
			collections.StringKey,
			codec.CollValue[types.Document](cdc),
		),
		DocumentCounters: collections.NewMap(
			sb,
			types.DocumentCountersKeyPrefix,
			"document_counters",
			collections.StringKey,
			collections.Uint64Value,
		),
		Roles: collections.NewMap(
			sb,
			collections.NewPrefix("roles"),
			"roles",
			collections.StringKey,
			collections.StringValue,
		),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

func (k Keeper) RemoveDocumentInner(ctx sdk.Context, sender sdk.AccAddress, denom string, index uint64) error {
	senderStr := sender.String()
	role, err := k.Roles.Get(ctx, senderStr)
	if err != nil || (role != RoleAdmin && role != RoleEditor) {
		return sdkerrors.ErrUnauthorized
	}
	doc, err := k.DocumentsByDenom.Get(ctx, collections.Join(denom, index))
	if err != nil {
		return err
	}
	if err = k.DocumentsByDenom.Remove(ctx, collections.Join(denom, index)); err != nil {
		return err
	}
	if err = k.DocumentsByChecksum.Remove(ctx, doc.Checksum); err != nil {
		return err
	}
	return nil
}

func (k Keeper) AddDocumentInner(ctx sdk.Context, sender sdk.AccAddress, doc types.Document) error {
	senderStr := sender.String()
	role, err := k.Roles.Get(ctx, senderStr)
	if err != nil || (role != RoleAdmin && role != RoleEditor) {
		return sdkerrors.ErrUnauthorized
	}
	if err = k.setDocument(ctx, doc); err != nil {
		return err
	}
	return nil
}
