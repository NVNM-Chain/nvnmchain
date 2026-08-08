package keeper_test

import (
	"testing"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
)

func TestQueryRegistries_NameFilter(t *testing.T) {
	k, ctx, qs, admin := setupAnchoringQueryServer(t)

	// Two registries deliberately share a name, with a third interleaved
	// between them, so the filter has to be multi-valued rather than a lookup.
	firstID, err := k.AddRegistry(ctx, admin, "shared", "first", "{}")
	require.NoError(t, err)
	otherID, err := k.AddRegistry(ctx, admin, "other", "", "{}")
	require.NoError(t, err)
	secondID, err := k.AddRegistry(ctx, admin, "shared", "second", "{}")
	require.NoError(t, err)

	ids := func(req *types.QueryRegistriesRequest) []uint64 {
		t.Helper()
		rsp, err := qs.Registries(ctx, req)
		require.NoError(t, err)
		out := make([]uint64, 0, len(rsp.Registries))
		for _, reg := range rsp.Registries {
			out = append(out, reg.Id)
		}
		return out
	}

	require.ElementsMatch(t, []uint64{firstID, secondID}, ids(&types.QueryRegistriesRequest{Name: "shared"}))
	require.Equal(t, []uint64{otherID}, ids(&types.QueryRegistriesRequest{Name: "other"}))

	// An unknown name is an empty page rather than an error, and the match is
	// byte-exact: no case folding, trimming or prefix match.
	for _, name := range []string{"nope", "Shared", "SHARED", "share", "shared "} {
		require.Emptyf(t, ids(&types.QueryRegistriesRequest{Name: name}), "name %q must not match", name)
	}

	// An empty name leaves the listing unfiltered.
	require.Len(t, ids(&types.QueryRegistriesRequest{}), 3)

	// Prefix, suffix and contains match on substrings of the name.
	shared := []uint64{firstID, secondID}
	require.ElementsMatch(t, shared, ids(&types.QueryRegistriesRequest{NamePrefix: "shar"}))
	require.ElementsMatch(t, shared, ids(&types.QueryRegistriesRequest{NameContains: "hare"}))
	require.Equal(t, []uint64{otherID}, ids(&types.QueryRegistriesRequest{NameSuffix: "er"}))

	// They are byte-exact too, and anchored: a prefix only matches at the
	// start, a suffix only at the end.
	require.Empty(t, ids(&types.QueryRegistriesRequest{NamePrefix: "Shar"}))
	require.Empty(t, ids(&types.QueryRegistriesRequest{NamePrefix: "hared"}))
	require.Empty(t, ids(&types.QueryRegistriesRequest{NameSuffix: "othe"}))

	// Filters combine with AND rather than one winning, so a contradictory set
	// returns nothing instead of silently honouring whichever came first.
	require.ElementsMatch(t, shared, ids(&types.QueryRegistriesRequest{
		NamePrefix: "sh", NameSuffix: "ed",
	}))
	require.Empty(t, ids(&types.QueryRegistriesRequest{NamePrefix: "sh", NameSuffix: "er"}))
	require.Empty(t, ids(&types.QueryRegistriesRequest{Name: "shared", NameSuffix: "er"}))

	// registry_id and name combine rather than one silently overriding the other.
	require.Equal(t, []uint64{firstID}, ids(&types.QueryRegistriesRequest{RegistryId: firstID, Name: "shared"}))
	require.Empty(t, ids(&types.QueryRegistriesRequest{RegistryId: otherID, Name: "shared"}))

	// Paging walks matches only, stepping over the non-matching registry.
	page, err := qs.Registries(ctx, &types.QueryRegistriesRequest{
		Name:       "shared",
		Pagination: &query.PageRequest{Limit: 1},
	})
	require.NoError(t, err)
	require.Len(t, page.Registries, 1)
	require.Equal(t, firstID, page.Registries[0].Id)
	require.NotEmpty(t, page.Pagination.NextKey)

	require.Equal(t, []uint64{secondID}, ids(&types.QueryRegistriesRequest{
		Name:       "shared",
		Pagination: &query.PageRequest{Limit: 1, Key: page.Pagination.NextKey},
	}))
}
