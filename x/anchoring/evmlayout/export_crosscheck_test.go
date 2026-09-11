package evmlayout_test

import (
	"fmt"
	"os"
	"testing"

	"cosmossdk.io/collections"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	"github.com/stretchr/testify/require"
)

// TestTheSeedMatchesTheExport reads every row of the export back out of the seeded state, so
// the rows that go on to Tempo are checked against where they came from rather than sampled.
//
//	NVNMCHAIN_EXPORT_DIR=/private/tmp/from-chain \
//	go test -run TestTheSeedMatchesTheExport -v -timeout 2h ./x/anchoring/evmlayout/
func TestTheSeedMatchesTheExport(t *testing.T) {
	dir := os.Getenv(exportDirEnvVar)
	if dir == "" {
		t.Skipf("set %s to cross-check the seed against the export", exportDirEnvVar)
	}

	k, ctx := anchoringKeeper(t)
	state := replayPreseed(t, k, ctx)
	seedCtx := ctx.WithBlockTime(state.SeedTime)
	e := readExport(t, dir)
	ids := seedTheExport(t, k, seedCtx, e, 0)
	seededAt := seedCtx.BlockTime().String()
	require.Len(t, e.Registries, e.Totals.Registries)

	// Every registry the export names, with the creator and time the v1.2 seed gave it.
	for _, r := range e.Registries {
		id, ok := ids[r.Name]
		require.Truef(t, ok, "registry %q was not seeded", r.Name)
		got, err := k.Registries.Get(ctx, id)
		require.NoError(t, err)
		require.Equal(t, types.Registry{
			Id:          id,
			Name:        r.Name,
			Description: r.Description,
			Creator:     mainnetRegistryAdmin,
			CreatedAt:   seededAt,
			Metadata:    r.Metadata,
		}, got)
	}

	// Every record row, in the order the seed read it. A checksum seen again in the same
	// registry is the next version of that record, so its occurrence is its index, and only the
	// last one is latest.
	rows, bad := 0, 0
	for _, entry := range e.Files {
		id := ids[entry.Registry]
		recs := readTranche(t, e.Dir, entry)
		versions := make(map[string]uint64, entry.Records)
		for _, rec := range recs {
			versions[rec.Checksum]++
		}
		seen := make(map[string]uint64, len(versions))
		for _, rec := range recs {
			seen[rec.Checksum]++
			index := seen[rec.Checksum]
			recordID, err := k.RecordIdByRegistryAndChecksum.Get(ctx, collections.Join(id, rec.Checksum))
			require.NoErrorf(t, err, "%s: no record id for %s", entry.File, rec.Checksum)
			got, err := k.Records.Get(ctx, collections.Join3(id, recordID, index))
			require.NoErrorf(t, err, "%s: %s version %d is missing", entry.File, rec.Checksum, index)
			want := types.Record{
				Uri:          rec.Uri,
				Checksum:     rec.Checksum,
				ChecksumAlgo: rec.ChecksumAlgo,
				Metadata:     rec.Metadata,
				Timestamp:    seededAt,
				Status:       rec.Status,
				RecordId:     recordID,
				Index:        index,
				IsLatest:     index == versions[rec.Checksum],
				RegistryId:   id,
			}
			if !got.Equal(&want) {
				if bad < 10 {
					t.Errorf("%s: %s version %d\n  export %+v\n  seeded %+v", entry.File, rec.Checksum, index, want, got)
				}
				bad++
			}
			rows++
		}
	}
	require.Zerof(t, bad, "%d rows differ from the export", bad)
	require.Equal(t, e.Totals.Records, rows)

	// Nothing else is in there: the export's rows plus the ones that predate the seed.
	stored := 0
	require.NoError(t, k.Records.Walk(ctx, nil, func(collections.Triple[uint64, uint64, uint64], types.Record) (bool, error) {
		stored++
		return false, nil
	}))
	require.Equal(t, rows+len(state.Records), stored)
	count, err := k.RegistryCount.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(len(e.Registries)+len(state.Registries)), count)

	fmt.Printf("cross-checked %d record versions and %d registries against %s\n", rows, len(e.Registries), dir)
}
