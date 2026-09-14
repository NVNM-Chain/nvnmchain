package evmlayout_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/evmlayout"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// Not a precompile and not TIP20-prefixed, like the contract's own address.
var dumpAddress = common.HexToAddress("0x000000000000000000000000000000000000ac01")

// The loader reads back exactly the migration: every slot once, under the one address.
func TestTheDumpCarriesTheMigration(t *testing.T) {
	k, ctx := anchoringKeeper(t)
	seedFixtureCorpus(t, k, ctx)
	want := migrated(t, k, ctx)

	var out bytes.Buffer
	dump := evmlayout.NewDump(&out, dumpAddress)
	require.NoError(t, evmlayout.Migrate(ctx, k, dump.Put))
	require.NoError(t, dump.Close())

	got, blocks := readDump(t, out.Bytes())
	require.Equal(t, want, got)
	require.Equal(t, []uint64{uint64(len(want))}, blocks)
}

// A full block goes out with its header; the rest starts the next one.
func TestTheDumpSplitsIntoBlocks(t *testing.T) {
	var out bytes.Buffer
	dump := evmlayout.NewDump(&out, dumpAddress)
	pairs := uint64(evmlayout.DumpBlockPairs + 1)
	for i := range pairs {
		require.NoError(t, dump.Put(evmlayout.Write{Slot: evmlayout.Word(i), Value: evmlayout.Word(i + 1)}))
	}
	require.NoError(t, dump.Close())

	got, blocks := readDump(t, out.Bytes())
	require.Equal(t, []uint64{evmlayout.DumpBlockPairs, 1}, blocks)
	require.Len(t, got, int(pairs))
	require.Equal(t, evmlayout.Word(pairs), got[evmlayout.Word(pairs-1)])
}

func TestAnEmptyDumpIsEmpty(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, evmlayout.NewDump(&out, dumpAddress).Close())
	require.Zero(t, out.Len())
}

// readDump parses a dump as `init-from-binary-dump` does, refusing a repeated slot, and returns
// the pairs and each block's count.
func readDump(t *testing.T, raw []byte) (map[common.Hash]common.Hash, []uint64) {
	t.Helper()
	pairs := map[common.Hash]common.Hash{}
	var blocks []uint64
	for len(raw) > 0 {
		require.GreaterOrEqual(t, len(raw), 40, "a truncated header")
		require.Equal(t, "TEMPOSB\x00", string(raw[:8]))
		require.Equal(t, uint16(1), binary.BigEndian.Uint16(raw[8:10]), "version")
		require.Zero(t, binary.BigEndian.Uint16(raw[10:12]), "flags")
		require.Equal(t, dumpAddress, common.BytesToAddress(raw[12:32]))
		count := binary.BigEndian.Uint64(raw[32:40])
		raw = raw[40:]

		require.GreaterOrEqual(t, uint64(len(raw)), count*64, "a truncated block")
		for range count {
			slot, value := common.BytesToHash(raw[:32]), common.BytesToHash(raw[32:64])
			_, seen := pairs[slot]
			require.Falsef(t, seen, "slot %s twice", slot)
			pairs[slot] = value
			raw = raw[64:]
		}
		blocks = append(blocks, count)
	}
	return pairs, blocks
}
