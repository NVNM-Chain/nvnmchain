package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// Records return the list of records based on the input fields
// unless index is otherwise stated, it will only return the latest record
// of a document checksum
func (q queryServer) Records(ctx context.Context, req *types.QueryRecordsRequest) (recordResponse *types.QueryRecordsResponse, err error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	const (
		defaultLimit uint64 = 50
		maxLimit     uint64 = 200
	)

	type paginator struct {
		countTotal bool
		offset     uint64
		limit      uint64
		seen       uint64
		collected  uint64
		total      uint64
	}

	newPaginator := func(pr *query.PageRequest, defaultLimit, maxLimit uint64) paginator {
		pageReq := pr
		if pageReq == nil {
			pageReq = &query.PageRequest{}
		}
		limit := pageReq.Limit
		if limit == 0 {
			limit = defaultLimit
		}
		if limit > maxLimit {
			limit = maxLimit
		}
		return paginator{
			countTotal: pageReq.CountTotal,
			offset:     pageReq.Offset,
			limit:      limit,
		}
	}

	step := func(p *paginator) (include bool, stop bool) {
		if p.countTotal {
			p.total++
		}
		if p.seen < p.offset {
			p.seen++
			return false, false
		}
		if p.collected < p.limit {
			p.seen++
			p.collected++
			// Stop when the page is full (unless counting total)
			if !p.countTotal && p.collected == p.limit {
				return true, true
			}
			return true, false
		}
		if !p.countTotal {
			return false, true
		}
		p.seen++
		return false, false
	}

	if req.RecordId == 0 && req.Checksum != "" && req.RegistryId != 0 {
		recordId, err := q.k.RecordIdByRegistryAndChecksum.Get(ctx, collections.Join(req.RegistryId, req.Checksum))
		if err != nil {
			return nil, err
		}
		req.RecordId = recordId
	}

	if req.RegistryId != 0 && req.RecordId != 0 {
		var record types.Record
		index := req.Index
		if req.RegistryId != 0 && req.RecordId != 0 {
			if index == 0 {
				index, err = q.k.RecordIndices.Get(ctx, collections.Join(req.RegistryId, req.RecordId))
				if err != nil {
					return nil, err
				}
			}
			record, err = q.k.Records.Get(sdkCtx, collections.Join3(req.RegistryId, req.RecordId, index))
			if err != nil {
				return nil, err
			}
		}
		return &types.QueryRecordsResponse{Records: []*types.Record{&record}}, nil
	}

	if req.RegistryId != 0 && req.RecordId == 0 {
		// Paginate within the registry instead of across all registries
		p := newPaginator(req.Pagination, defaultLimit, maxLimit)
		records := make([]*types.Record, 0, p.limit)

		walkErr := q.k.RecordIndices.Walk(
			ctx,
			collections.NewPrefixedPairRange[uint64, uint64](req.RegistryId),
			func(key collections.Pair[uint64, uint64], index uint64) (bool, error) {
				include, stop := step(&p)
				if include {
					recordId := key.K2()
					record, err := q.k.Records.Get(sdkCtx, collections.Join3(req.RegistryId, recordId, index))
					if err != nil {
						return true, err
					}
					records = append(records, &record)
				}
				return stop, nil
			},
		)
		if walkErr != nil {
			return nil, walkErr
		}

		var pageRes *query.PageResponse
		if req.Pagination != nil {
			pageRes = &query.PageResponse{Total: p.total}
		}
		return &types.QueryRecordsResponse{Records: records, Pagination: pageRes}, nil
	}

	if req.Checksum != "" && req.RegistryId == 0 {
		// Paginate checksum-only queries to avoid unbounded work
		p := newPaginator(req.Pagination, defaultLimit, maxLimit)
		selected := make([]collections.Pair[uint64, uint64], 0, p.limit) // (registryId, recordId)

		walkErr := q.k.RecordIdByChecksumAndRegistry.Walk(
			ctx,
			collections.NewPrefixedPairRange[string, uint64](req.Checksum),
			func(key collections.Pair[string, uint64], recordId uint64) (bool, error) {
				include, stop := step(&p)
				if include {
					selected = append(selected, collections.Join(key.K2(), recordId))
				}
				return stop, nil
			},
		)
		if walkErr != nil {
			return nil, walkErr
		}

		records := make([]*types.Record, 0, len(selected))
		for _, pair := range selected {
			registryId := pair.K1()
			recordId := pair.K2()
			index, err := q.k.RecordIndices.Get(ctx, collections.Join(registryId, recordId))
			if err != nil {
				return nil, err
			}
			record, err := q.k.Records.Get(sdkCtx, collections.Join3(registryId, recordId, index))
			if err != nil {
				return nil, err
			}
			records = append(records, &record)
		}

		var pageRes *query.PageResponse
		if req.Pagination != nil {
			pageRes = &query.PageResponse{Total: p.total}
		}
		return &types.QueryRecordsResponse{Records: records, Pagination: pageRes}, nil
	}

	return &types.QueryRecordsResponse{Records: nil, Pagination: nil}, nil
}

// Registries returns all registries with pagination, or a specific registry if registry_id or name is provided
func (q queryServer) Registries(ctx context.Context, req *types.QueryRegistriesRequest) (*types.QueryRegistriesResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req.RegistryId != 0 {
		registry, err := q.k.Registries.Get(sdkCtx, req.RegistryId)
		if err != nil {
			return nil, err
		}
		return &types.QueryRegistriesResponse{
			Registries: []*types.Registry{&registry},
			Pagination: nil,
		}, nil
	}

	if req.Name != "" {
		registryId, err := q.k.RegistryIdByName.Get(sdkCtx, req.Name)
		if err != nil {
			return nil, err
		}
		registry, err := q.k.Registries.Get(sdkCtx, registryId)
		if err != nil {
			return nil, err
		}
		return &types.QueryRegistriesResponse{
			Registries: []*types.Registry{&registry},
			Pagination: nil,
		}, nil
	}

	registries, pageRes, err := query.CollectionPaginate(ctx, q.k.Registries, req.Pagination, func(key uint64, registry types.Registry) (*types.Registry, error) {
		return &registry, nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryRegistriesResponse{Registries: registries, Pagination: pageRes}, nil
}

// Registry returns a specific registry by id or name
func (q queryServer) Registry(ctx context.Context, req *types.QueryRegistryRequest) (*types.QueryRegistryResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var id uint64
	var err error

	switch {
	case req.Id != 0:
		id = req.Id
	case req.Name != "":
		id, err = q.k.RegistryIdByName.Get(sdkCtx, req.Name)
		if err != nil {
			return nil, err
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "either id or name must be provided")
	}

	registry, err := q.k.Registries.Get(sdkCtx, id)
	if err != nil {
		return nil, err
	}

	return &types.QueryRegistryResponse{Registry: &registry}, nil
}
