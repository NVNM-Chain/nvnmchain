package keeper_test

import (
	"testing"

	appparams "github.com/NVNM-Chain/nvnmchain/app/params"
	keepertest "github.com/NVNM-Chain/nvnmchain/testutil/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func setupAnchoringQueryServer(t *testing.T) (keeper.Keeper, sdk.Context, types.QueryServer, sdk.AccAddress) {
	t.Helper()
	appparams.SetAddressPrefixes()
	k, ctx, addressCodec := keepertest.AnchoringKeeper(t)
	require.NoError(t, k.RegistryCount.Set(ctx, 0))
	adminStr := keepertest.TestSenderAddr
	adminBz, err := addressCodec.StringToBytes(adminStr)
	require.NoError(t, err)
	admin := sdk.AccAddress(adminBz)
	qs := keeper.NewQueryServerImpl(k)
	return k, ctx, qs, admin
}

func TestParamsQuery(t *testing.T) {
	k, ctx, _ := keepertest.AnchoringKeeper(t)

	qs := keeper.NewQueryServerImpl(k)
	params := types.DefaultParams()
	require.NoError(t, k.Params.Set(ctx, params))

	response, err := qs.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}

func TestQueryRecords_ChecksumOnly_RespectsPagination(t *testing.T) {
	k, ctx, qs, admin := setupAnchoringQueryServer(t)

	checksum := "same-checksum"
	registries := []string{"reg-a", "reg-b", "reg-c"}
	for _, name := range registries {
		registryId, err := k.AddRegistry(ctx, admin, name, "", "{}")
		require.NoError(t, err)

		_, err = k.AddRecord(ctx, admin, types.Record{
			RegistryId:   registryId,
			Uri:          "ipfs://" + checksum,
			Checksum:     checksum,
			ChecksumAlgo: "sha256",
			Metadata:     "{\"k\":\"v\"}",
			Status:       "active",
		})
		require.NoError(t, err)
	}

	rsp1, err := qs.Records(ctx, &types.QueryRecordsRequest{
		Checksum:   checksum,
		RegistryId: 0,
		Pagination: &query.PageRequest{Limit: 1, Offset: 0},
	})
	require.NoError(t, err)
	require.Len(t, rsp1.Records, 1)

	rsp2, err := qs.Records(ctx, &types.QueryRecordsRequest{
		Checksum:   checksum,
		RegistryId: 0,
		Pagination: &query.PageRequest{Limit: 1, Offset: 1},
	})
	require.NoError(t, err)
	require.Len(t, rsp2.Records, 1)

	rspAll, err := qs.Records(ctx, &types.QueryRecordsRequest{
		Checksum:   checksum,
		RegistryId: 0,
		Pagination: &query.PageRequest{Limit: 50, Offset: 0},
	})
	require.NoError(t, err)
	require.Len(t, rspAll.Records, len(registries))
}

func TestQueryRecords_RegistryOnly_PaginationDoesNotSkipResults(t *testing.T) {
	k, ctx, qs, admin := setupAnchoringQueryServer(t)
	regAID, err := k.AddRegistry(ctx, admin, "reg-a", "", "{}")
	require.NoError(t, err)
	regBID, err := k.AddRegistry(ctx, admin, "reg-b", "", "{}")
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		_, err := k.AddRecord(ctx, admin, types.Record{
			RegistryId:   regAID,
			Uri:          "ipfs://a",
			Checksum:     "chk-a-" + string(rune('a'+i)),
			ChecksumAlgo: "sha256",
			Metadata:     "{\"k\":\"v\"}",
			Status:       "active",
		})
		require.NoError(t, err)
	}

	checksumB := "chk-b"
	_, err = k.AddRecord(ctx, admin, types.Record{
		RegistryId:   regBID,
		Uri:          "ipfs://b",
		Checksum:     checksumB,
		ChecksumAlgo: "sha256",
		Metadata:     "{\"k\":\"v\"}",
		Status:       "active",
	})
	require.NoError(t, err)

	exact, err := qs.Records(ctx, &types.QueryRecordsRequest{
		RegistryId: regBID,
		Checksum:   checksumB,
		RecordId:   0,
	})
	require.NoError(t, err)
	require.Len(t, exact.Records, 1)

	rsp, err := qs.Records(ctx, &types.QueryRecordsRequest{
		RegistryId: regBID,
		RecordId:   0,
		Pagination: &query.PageRequest{Limit: 1, Offset: 0},
	})
	require.NoError(t, err)
	require.NotEmpty(t, rsp.Records)
	require.Equal(t, regBID, rsp.Records[0].GetRegistryId())
	require.Equal(t, checksumB, rsp.Records[0].GetChecksum())

	rspA, err := qs.Records(ctx, &types.QueryRecordsRequest{
		RegistryId: regAID,
		RecordId:   0,
		Pagination: &query.PageRequest{Limit: 1, Offset: 0},
	})
	require.NoError(t, err)
	require.NotEmpty(t, rspA.Records)
}

func TestQueryRecords_CountTotalDoesNotBypassLimit(t *testing.T) {
	k, ctx, qs, admin := setupAnchoringQueryServer(t)
	regID, err := k.AddRegistry(ctx, admin, "reg-limit", "", "{}")
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		_, err = k.AddRecord(ctx, admin, types.Record{
			RegistryId:   regID,
			Uri:          "ipfs://limit",
			Checksum:     "chk-limit-" + string(rune('a'+i)),
			ChecksumAlgo: "sha256",
			Metadata:     "{\"k\":\"v\"}",
			Status:       "active",
		})
		require.NoError(t, err)
	}

	rsp, err := qs.Records(ctx, &types.QueryRecordsRequest{
		RegistryId: regID,
		Pagination: &query.PageRequest{Limit: 1, CountTotal: true},
	})
	require.NoError(t, err)
	require.Len(t, rsp.Records, 1)
	require.NotNil(t, rsp.Pagination)
	require.Zero(t, rsp.Pagination.Total)
}

func TestQueryRecords_EmptyRequest_ReturnsLatestRecords(t *testing.T) {
	k, ctx, qs, admin := setupAnchoringQueryServer(t)

	regID, err := k.AddRegistry(ctx, admin, "reg-empty-all", "", "{}")
	require.NoError(t, err)

	_, err = k.AddRecord(ctx, admin, types.Record{
		RegistryId:   regID,
		Uri:          "ipfs://v1",
		Checksum:     "chk-empty-all",
		ChecksumAlgo: "sha256",
		Metadata:     "{\"v\":\"1\"}",
		Status:       "active",
	})
	require.NoError(t, err)

	_, err = k.AddRecord(ctx, admin, types.Record{
		RegistryId:   regID,
		Uri:          "ipfs://v2",
		Checksum:     "chk-empty-all",
		ChecksumAlgo: "sha256",
		Metadata:     "{\"v\":\"2\"}",
		Status:       "active",
	})
	require.NoError(t, err)

	rsp, err := qs.Records(ctx, &types.QueryRecordsRequest{})
	require.NoError(t, err)
	require.Len(t, rsp.Records, 1, "empty request should return latest records")
	require.Equal(t, uint64(2), rsp.Records[0].GetIndex())
	require.Equal(t, "chk-empty-all", rsp.Records[0].GetChecksum())
}

func TestQueryRecords_RejectsUnmatchedFilters(t *testing.T) {
	k, ctx, qs, admin := setupAnchoringQueryServer(t)

	regID, err := k.AddRegistry(ctx, admin, "reg-reject", "", "{}")
	require.NoError(t, err)
	_, err = k.AddRecord(ctx, admin, types.Record{
		RegistryId:   regID,
		Uri:          "ipfs://r",
		Checksum:     "chk-reject",
		ChecksumAlgo: "sha256",
		Metadata:     "{\"k\":\"v\"}",
		Status:       "active",
	})
	require.NoError(t, err)

	cases := []struct {
		name string
		req  *types.QueryRecordsRequest
	}{
		{
			name: "record_id without registry_id",
			req:  &types.QueryRecordsRequest{RecordId: 1},
		},
		{
			name: "record_id with checksum but no registry_id",
			req:  &types.QueryRecordsRequest{RecordId: 1, Checksum: "chk-reject"},
		},
		{
			name: "index without registry_id and record_id",
			req:  &types.QueryRecordsRequest{Index: 1},
		},
		{
			name: "index with registry_id but no record_id",
			req:  &types.QueryRecordsRequest{RegistryId: regID, Index: 1},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rsp, err := qs.Records(ctx, tc.req)
			require.Error(t, err)
			require.Nil(t, rsp)
			st, ok := status.FromError(err)
			require.True(t, ok, "expected gRPC status error")
			require.Equal(t, codes.InvalidArgument, st.Code())
		})
	}
}

func TestQueryRegistries_CountTotalDoesNotTriggerFullScan(t *testing.T) {
	k, ctx, qs, admin := setupAnchoringQueryServer(t)

	// Seed enough registries to exercise pagination and next-key behavior.
	// CountTotal must still remain disabled to avoid full scans.
	for i := 0; i < 3; i++ {
		name := "reg-ct-" + string(rune('a'+i))
		_, err := k.AddRegistry(ctx, admin, name, "", "{}")
		require.NoError(t, err)
	}

	rsp, err := qs.Registries(ctx, &types.QueryRegistriesRequest{
		Pagination: &query.PageRequest{Limit: 1, CountTotal: true},
	})
	require.NoError(t, err)
	require.Len(t, rsp.Registries, 1)
	require.NotNil(t, rsp.Pagination)
	require.Zero(t, rsp.Pagination.Total, "count_total must be ignored to avoid expensive scans")
}

func TestQueryRecords_RegistryOnly_CountTotalDoesNotLeakCrossRegistry(t *testing.T) {
	k, ctx, qs, admin := setupAnchoringQueryServer(t)

	regAID, err := k.AddRegistry(ctx, admin, "reg-a", "", "{}")
	require.NoError(t, err)
	regBID, err := k.AddRegistry(ctx, admin, "reg-b", "", "{}")
	require.NoError(t, err)

	// 12 records in Registry A to fill early pages in the old full-collection scan
	for i := 0; i < 12; i++ {
		_, err = k.AddRecord(ctx, admin, types.Record{
			RegistryId:   regAID,
			Uri:          "ipfs://a",
			Checksum:     "chk-a-" + string(rune('a'+i)),
			ChecksumAlgo: "sha256",
			Metadata:     "{\"k\":\"v\"}",
			Status:       "active",
		})
		require.NoError(t, err)
	}

	checksumB := "chk-b"
	_, err = k.AddRecord(ctx, admin, types.Record{
		RegistryId:   regBID,
		Uri:          "ipfs://b",
		Checksum:     checksumB,
		ChecksumAlgo: "sha256",
		Metadata:     "{\"k\":\"v\"}",
		Status:       "active",
	})
	require.NoError(t, err)

	// exact query (registry + checksum) must return 1 record
	exact, err := qs.Records(ctx, &types.QueryRecordsRequest{
		RegistryId: regBID,
		Checksum:   checksumB,
	})
	require.NoError(t, err)
	require.Len(t, exact.Records, 1, "exact_count should be 1")

	// paginated registry-only query must also return 1 record (not 0)
	paged, err := qs.Records(ctx, &types.QueryRecordsRequest{
		RegistryId: regBID,
		Pagination: &query.PageRequest{Limit: 1, CountTotal: true},
	})
	require.NoError(t, err)
	require.Len(t, paged.Records, 1, "page_count should be 1, not 0")
	require.Equal(t, regBID, paged.Records[0].GetRegistryId())
	require.Equal(t, checksumB, paged.Records[0].GetChecksum())
	require.NotNil(t, paged.Pagination)
	require.Zero(t, paged.Pagination.Total, "page_total must be 0 (CountTotal disabled)")

	// sanity-check: registry A is unaffected
	pagedA, err := qs.Records(ctx, &types.QueryRecordsRequest{
		RegistryId: regAID,
		Pagination: &query.PageRequest{Limit: 1},
	})
	require.NoError(t, err)
	require.Len(t, pagedA.Records, 1)
	require.Equal(t, regAID, pagedA.Records[0].GetRegistryId())
}
