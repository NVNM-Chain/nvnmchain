package keeper

import (
	"github.com/MANTRA-Chain/inveniam/x/document/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the tokenfactory module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) error {
	err := k.Params.Set(ctx, genState.Params)
	if err != nil {
		return err
	}

	for key, genDoc := range genState.GetDocuments() {
		_, key, err := k.DocumentsByDenom.KeyCodec().Decode([]byte(key))
		if err != nil {
			return err
		}
		err = k.DocumentsByDenom.Set(ctx, key, genDoc)
		if err != nil {
			return err
		}
		err = k.DocumentsByChecksum.Set(ctx, genDoc.Checksum, genDoc)
		if err != nil {
			return err
		}
	}

	for denom, counter := range genState.GetDocumentCounters() {
		err = k.DocumentCounters.Set(ctx, denom, counter)
		if err != nil {
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

	genDocs := map[string]types.Document{}
	iterator, err := k.DocumentsByDenom.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var doc types.Document
		doc, err := iterator.Value()
		if err != nil {
			return nil, err
		}
		key, err := iterator.Key()
		if err != nil {
			return nil, err
		}
		var keyBytes []byte
		if _, err = k.DocumentsByDenom.KeyCodec().Encode(keyBytes, key); err != nil {
			return nil, err
		}
		genDocs[string(keyBytes)] = doc
	}

	genCounters := map[string]uint64{}
	counterIterator, err := k.DocumentCounters.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer counterIterator.Close()
	for ; counterIterator.Valid(); counterIterator.Next() {
		denom, err := counterIterator.Key()
		if err != nil {
			return nil, err
		}
		counter, err := counterIterator.Value()
		if err != nil {
			return nil, err
		}
		genCounters[denom] = counter
	}

	return &types.GenesisState{
		Params:           params,
		Documents:        genDocs,
		DocumentCounters: genCounters,
	}, nil
}
