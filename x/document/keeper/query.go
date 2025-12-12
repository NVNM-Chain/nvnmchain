package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"github.com/MANTRA-Chain/inveniam/x/document/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

var _ types.QueryServer = queryServer{}

// NewQueryServerImpl returns an implementation of the QueryServer interface
// for the provided Keeper.
func NewQueryServerImpl(k Keeper) types.QueryServer {
	return queryServer{k}
}

type queryServer struct {
	k Keeper
}

func (q queryServer) Documents(ctx context.Context, req *types.QueryDocumentsRequest) (*types.QueryDocumentsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req.Denom != "" && req.Index != 0 {
		doc, err := q.k.DocumentsByDenom.Get(sdkCtx, collections.Join(req.Denom, req.Index))
		if err != nil {
			return nil, err
		}
		return &types.QueryDocumentsResponse{Documents: []*types.Document{&doc}}, nil
	} else if req.Denom != "" {
		filteredDocs, pageRes, err := query.CollectionPaginate(ctx, q.k.DocumentsByDenom, req.Pagination, func(_ collections.Pair[string, uint64], value types.Document) (*types.Document, error) {
			return &value, nil
		}, query.WithCollectionPaginationPairPrefix[string, uint64](req.Denom))
		if err != nil {
			return nil, err
		}
		return &types.QueryDocumentsResponse{Documents: filteredDocs, Pagination: pageRes}, nil
	}

	filteredDocs, pageRes, err := query.CollectionPaginate(ctx, q.k.DocumentsByDenom, req.Pagination, func(_ collections.Pair[string, uint64], value types.Document) (*types.Document, error) {
		return &value, nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryDocumentsResponse{Documents: filteredDocs, Pagination: pageRes}, nil
}

func (q queryServer) Document(ctx context.Context, req *types.QueryDocumentRequest) (*types.QueryDocumentResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	doc, err := q.k.DocumentsByChecksum.Get(sdkCtx, req.Checksum)
	if err != nil {
		return nil, err
	}

	return &types.QueryDocumentResponse{Documents: &doc}, nil
}
