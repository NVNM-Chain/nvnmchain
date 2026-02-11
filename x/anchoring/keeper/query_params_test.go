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
	appparams.SetAddressPrefixes()
	k, ctx, addressCodec := keepertest.AnchoringKeeper(t)

	require.NoError(t, k.RegistryCount.Set(ctx, 0))

	adminStr := "inveniam1axznhnm82lah8qqvp9hxdad49yx3s5dcmnx072"
	adminBz, err := addressCodec.StringToBytes(adminStr)
	require.NoError(t, err)
	admin := sdk.AccAddress(adminBz)

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

	qs := keeper.NewQueryServerImpl(k)

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
