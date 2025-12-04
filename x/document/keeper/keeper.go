package keeper

import (
	"bytes"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/store"
	"github.com/MANTRA-Chain/inveniam/x/document/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type (
	Keeper struct {
		cdc                codec.BinaryCodec
		addressCodec       address.Codec
		storeService       store.KVStoreService
		tokenFactoryKeeper types.TokenFactoryKeeper

		Schema           collections.Schema
		Params           collections.Item[types.Params]
		Documents        collections.Map[collections.Pair[string, uint64], types.Document]
		DocumentCounters collections.Map[string, uint64]
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	addressCodec address.Codec,
	storeService store.KVStoreService,
	tokenFactoryKeeper types.TokenFactoryKeeper,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		cdc:          cdc,
		storeService: storeService,

		Params: collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		Documents: collections.NewMap(
			sb,
			types.DocumentKeyPrefix,
			"documents",
			collections.PairKeyCodec(collections.StringKey, collections.Uint64Key),
			codec.CollValue[types.Document](cdc),
		),
		DocumentCounters: collections.NewMap(
			sb,
			types.DocumentCountersKeyPrefix,
			"document_counters",
			collections.StringKey,
			collections.Uint64Value,
		),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	k.tokenFactoryKeeper = tokenFactoryKeeper

	return k
}

func (k Keeper) RemoveDocumentInner(ctx sdk.Context, sender sdk.AccAddress, denom string, index uint64) error {
	tokenAdmin, err := k.tokenFactoryKeeper.GetAuthorityMetadata(ctx, denom)
	if err != nil {
		return err
	}
	tokenAdminAddress, err := k.addressCodec.StringToBytes(tokenAdmin.Admin)
	if err != nil {
		return err
	}
	if !bytes.Equal(tokenAdminAddress, sender) {
		params, err := k.Params.Get(ctx)
		if err != nil {
			return err
		}
		paramsAdmin, err := k.addressCodec.StringToBytes(params.Admin)
		if err != nil {
			return err
		}
		if !bytes.Equal(paramsAdmin, sender) {
			return sdkerrors.ErrUnauthorized
		}
	}

	err = k.Documents.Remove(ctx, collections.Join(denom, index))
	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) AddDocumentInner(ctx sdk.Context, sender sdk.AccAddress, doc types.Document) error {
	tokenAdmin, err := k.tokenFactoryKeeper.GetAuthorityMetadata(ctx, doc.Denom)
	if err != nil {
		return err
	}
	tokenAdminAddress, err := k.addressCodec.StringToBytes(tokenAdmin.Admin)
	if err != nil {
		return err
	}
	if !bytes.Equal(tokenAdminAddress, sender) {
		params, err := k.Params.Get(ctx)
		if err != nil {
			return err
		}
		paramsAdmin, err := k.addressCodec.StringToBytes(params.Admin)
		if err != nil {
			return err
		}
		if !bytes.Equal(paramsAdmin, sender) {
			return sdkerrors.ErrUnauthorized
		}
	}

	return k.setDocument(ctx, doc)
}
