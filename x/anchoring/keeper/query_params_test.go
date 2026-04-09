package keeper_test

import (
	"testing"

	appparams "github.com/MANTRA-Chain/inveniam/app/params"
	keepertest "github.com/MANTRA-Chain/inveniam/testutil/keeper"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/keeper"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
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
		_, err := k.AddRegistry(ctx, admin, name, "", "{}")
		require.NoError(t, err)

		_, err = k.AddRecord(ctx, admin, types.Record{
			Registry:     name,
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
			Registry:     "reg-a",
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
		Registry:     "reg-b",
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
	require.Equal(t, "reg-b", rsp.Records[0].GetRegistry())
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
			Registry:     "reg-limit",
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
