package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = queryServer{}

const (
	defaultPageLimit uint64 = 50
	maxPageLimit     uint64 = 200
)

// NewQueryServerImpl returns an implementation of the QueryServer interface
// for the provided Keeper.
func NewQueryServerImpl(k Keeper) types.QueryServer {
	return queryServer{k}
}

type queryServer struct {
	k Keeper
}

func sanitizePageRequest(pr *query.PageRequest, defaultLimit, maxLimit uint64) *query.PageRequest {
	pageReq := &query.PageRequest{}
	if pr != nil {
		*pageReq = *pr
	}
	if pageReq.Limit == 0 {
		pageReq.Limit = defaultLimit
	}
	if pageReq.Limit > maxLimit {
		pageReq.Limit = maxLimit
	}
	// Reject expensive total-count scans in module queries.
	pageReq.CountTotal = false
	return pageReq
}

// Records return the list of records based on the input fields
// unless index is otherwise stated, it will only return the latest record
// of a document checksum
func (q queryServer) Records(ctx context.Context, req *types.QueryRecordsRequest) (recordResponse *types.QueryRecordsResponse, err error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if req == nil {
		req = &types.QueryRecordsRequest{}
	}

	// record_id is registry-scoped
	if req.RecordId != 0 && req.RegistryId == 0 {
		return nil, status.Error(codes.InvalidArgument, "record_id requires registry_id")
	}
	// index is per (registry_id, record_id), record_id may be resolved from checksum
	if req.Index != 0 && (req.RegistryId == 0 || (req.RecordId == 0 && req.Checksum == "")) {
		return nil, status.Error(codes.InvalidArgument, "index requires registry_id and either record_id or checksum")
	}

	type paginator struct {
		offset    uint64
		limit     uint64
		seen      uint64
		collected uint64
	}

	newPaginator := func(pr *query.PageRequest) paginator {
		pageReq := sanitizePageRequest(pr, defaultPageLimit, maxPageLimit)
		return paginator{
			offset: pageReq.Offset,
			limit:  pageReq.Limit,
		}
	}

	step := func(p *paginator) (include bool, stop bool) {
		if p.seen < p.offset {
			p.seen++
			return false, false
		}
		if p.collected < p.limit {
			p.seen++
			p.collected++
			// Stop as soon as the page is full.
			if p.collected == p.limit {
				return true, true
			}
			return true, false
		}
		return false, true
	}

	if req.RecordId == 0 && req.Checksum != "" && req.RegistryId != 0 {
		recordId, err := q.k.RecordIdByRegistryAndChecksum.Get(ctx, collections.Join(req.RegistryId, req.Checksum))
		if err != nil {
			return nil, err
		}
		req.RecordId = recordId
	}

	if req.RegistryId != 0 && req.RecordId != 0 {
		index := req.Index
		if index == 0 {
			index, err = q.k.RecordIndices.Get(ctx, collections.Join(req.RegistryId, req.RecordId))
			if err != nil {
				return nil, err
			}
		}
		record, err := q.k.Records.Get(sdkCtx, collections.Join3(req.RegistryId, req.RecordId, index))
		if err != nil {
			return nil, err
		}
		return &types.QueryRecordsResponse{Records: []*types.Record{&record}}, nil
	}

	if req.RegistryId != 0 && req.RecordId == 0 {
		// Paginate within the registry instead of across all registries
		p := newPaginator(req.Pagination)
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
			pageRes = &query.PageResponse{}
		}
		return &types.QueryRecordsResponse{Records: records, Pagination: pageRes}, nil
	}

	if req.Checksum != "" && req.RegistryId == 0 {
		// Paginate checksum-only queries to avoid unbounded work
		p := newPaginator(req.Pagination)
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
			pageRes = &query.PageResponse{}
		}
		return &types.QueryRecordsResponse{Records: records, Pagination: pageRes}, nil
	}

	// Empty filter: return latest records across all registries.
	p := newPaginator(req.Pagination)
	records := make([]*types.Record, 0, p.limit)

	walkErr := q.k.RecordIndices.Walk(
		ctx,
		nil,
		func(key collections.Pair[uint64, uint64], index uint64) (bool, error) {
			include, stop := step(&p)
			if include {
				registryId := key.K1()
				recordId := key.K2()
				record, err := q.k.Records.Get(sdkCtx, collections.Join3(registryId, recordId, index))
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
		pageRes = &query.PageResponse{}
	}
	return &types.QueryRecordsResponse{Records: records, Pagination: pageRes}, nil
}

// Registries returns all registries with pagination, or a specific registry if
// registry_id is provided. The optional name filter matches exactly and may
// match several registries, since names are not unique. It is a scan, not an
// index: a request matching nothing decodes every registry, bounded by the
// node's query-gas-limit rather than by the page limit.
func (q queryServer) Registries(ctx context.Context, req *types.QueryRegistriesRequest) (*types.QueryRegistriesResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req.RegistryId != 0 {
		registry, err := q.k.Registries.Get(sdkCtx, req.RegistryId)
		if err != nil {
			return nil, err
		}
		if req.Name != "" && registry.Name != req.Name {
			return &types.QueryRegistriesResponse{}, nil
		}
		return &types.QueryRegistriesResponse{
			Registries: []*types.Registry{&registry},
			Pagination: nil,
		}, nil
	}

	pageReq := sanitizePageRequest(req.Pagination, defaultPageLimit, maxPageLimit)
	registries, pageRes, err := query.CollectionFilteredPaginate(ctx, q.k.Registries, pageReq,
		func(_ uint64, registry types.Registry) (bool, error) {
			return req.Name == "" || registry.Name == req.Name, nil
		},
		func(_ uint64, registry types.Registry) (*types.Registry, error) {
			return &registry, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.QueryRegistriesResponse{Registries: registries, Pagination: pageRes}, nil
}

// Registry returns a specific registry by id
func (q queryServer) Registry(ctx context.Context, req *types.QueryRegistryRequest) (*types.QueryRegistryResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be provided")
	}

	registry, err := q.k.Registries.Get(sdkCtx, req.Id)
	if err != nil {
		return nil, err
	}

	return &types.QueryRegistryResponse{Registry: &registry}, nil
}
