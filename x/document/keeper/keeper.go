package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/MANTRA-Chain/inveniam/x/document/types"
	"github.com/cosmos/cosmos-sdk/codec"
)

type (
	Keeper struct {
		cdc                codec.BinaryCodec
		addressCodec       address.Codec
		storeService       store.KVStoreService
		logger             log.Logger
		authKeeper         types.AccountKeeper
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
	logger log.Logger,
	ak types.AccountKeeper,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		cdc:          cdc,
		addressCodec: addressCodec,
		storeService: storeService,
		logger:       logger,
		authKeeper:   ak,

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

	return k
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
