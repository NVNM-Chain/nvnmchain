package keeper_test

import (
	"path/filepath"
	"testing"

	appparams "github.com/NVNM-Chain/nvnmchain/app/params"
	keepertest "github.com/NVNM-Chain/nvnmchain/testutil/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/nameindex"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestBackfillNameIndex_ThenSearch(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	sender := keepertest.TestSenderAddr

	keepertest.MustCreateAnchoringRegistry(t, k, ctx, sender, "kyc_registry")
	keepertest.MustCreateAnchoringRegistry(t, k, ctx, sender, "aml_registry")

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	store, err := nameindex.Open(filepath.Join(t.TempDir(), "test.db"), cdc)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	k.NameIndex = store

	require.NoError(t, k.BackfillNameIndex(ctx))

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.SearchRegistriesByName(ctx, &types.QuerySearchRegistriesByNameRequest{
		Name: "kyc",
		Mode: types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_CONTAINS,
	})
	require.NoError(t, err)
	require.Len(t, resp.Registries, 1)
	require.Equal(t, "kyc_registry", resp.Registries[0].Name)

	// Idempotent: calling backfill again must not duplicate or error.
	require.NoError(t, k.BackfillNameIndex(ctx))
	resp, err = qs.SearchRegistriesByName(ctx, &types.QuerySearchRegistriesByNameRequest{
		Name: "reg",
		Mode: types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_CONTAINS,
	})
	require.NoError(t, err)
	require.Len(t, resp.Registries, 2)

	// Complete index: the backfill compares counts and reads nothing. A row
	// altered behind its back survives, which a re-walk would overwrite.
	require.NoError(t, store.Upsert(&types.Registry{Id: 1, Name: "renamed_locally"}))
	require.NoError(t, k.BackfillNameIndex(ctx))
	got, err := store.Search(types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_EXACT, "renamed_locally", 50, 0)
	require.NoError(t, err)
	require.Len(t, got, 1, "a complete index must not be re-walked on start")

	// Behind by one registry: the counts differ and the walk repairs it.
	keepertest.MustCreateAnchoringRegistry(t, k, ctx, sender, "late_registry")
	require.NoError(t, k.BackfillNameIndex(ctx))
	got, err = store.Search(types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_EXACT, "kyc_registry", 50, 0)
	require.NoError(t, err)
	require.Len(t, got, 1, "the re-walk restores the chain's name")
	got, err = store.Search(types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_EXACT, "late_registry", 50, 0)
	require.NoError(t, err)
	require.Len(t, got, 1)
}

func TestSearchRegistriesByName_ContainsTooShort(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	store, err := nameindex.Open(filepath.Join(t.TempDir(), "test.db"), cdc)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	k.NameIndex = store

	// A CONTAINS query below the trigram width is the caller's mistake, not
	// the node's, so it surfaces as InvalidArgument rather than Internal.
	qs := keeper.NewQueryServerImpl(k)
	_, err = qs.SearchRegistriesByName(ctx, &types.QuerySearchRegistriesByNameRequest{
		Name: "ab",
		Mode: types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_CONTAINS,
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.ErrorContains(t, err, "at least 3 characters")
}

func TestSearchRegistriesByName_DisabledOnNode(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	require.Nil(t, k.NameIndex)

	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.SearchRegistriesByName(ctx, &types.QuerySearchRegistriesByNameRequest{Name: "anything"})
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}
