package evmlayout_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/evmlayout"
	anchoringkeeper "github.com/NVNM-Chain/nvnmchain/x/anchoring/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// Written by contracts/test/SeedFixture.t.sol; `make layout` in contracts/ regenerates it.
const fixturePath = "../../../contracts/layout/seed-fixture.json"

var (
	moduleAdmin = common.HexToAddress("0x00000000000000000000000000000000000000a0")
	alice       = common.HexToAddress("0x00000000000000000000000000000000000a11ce")
	bob         = common.HexToAddress("0x0000000000000000000000000000000000000b0b")
	blockTime   = time.Unix(1757376000, 0).UTC()
)

// The fixture's corpus has to migrate to exactly the slots the contract wrote for it, save the
// header: the contract leaves the admin's 20 bytes empty and the migration fills them in.
func TestMigrationMatchesWhatTheContractWrote(t *testing.T) {
	k, ctx := anchoringKeeper(t)
	seedFixtureCorpus(t, k, ctx)

	written := migrated(t, k, ctx)
	header := evmlayout.Word(evmlayout.SlotHeader)
	slot := written[header]
	require.Equal(t, moduleAdmin, common.BytesToAddress(slot[12:]), "the admin the module had")

	fixture := readFixture(t)
	expected := fixture[header]
	copy(expected[12:], slot[12:]) // only the admin's bytes; the count still has to agree
	fixture[header] = expected
	require.Equal(t, fixture, written)
}

// The calls SeedFixture.t.sol makes, in the same order.
func seedFixtureCorpus(t *testing.T, k anchoringkeeper.Keeper, ctx sdk.Context) {
	t.Helper()
	msgs := anchoringkeeper.NewMsgServerImpl(k)

	first, err := k.AddRegistry(ctx, alice.Bytes(), "us-ca1", "First Circuit", `{"tranche":1}`)
	require.NoError(t, err)
	second, err := k.AddRegistry(ctx, bob.Bytes(), "us-ca9", "Ninth Circuit", "{}")
	require.NoError(t, err)

	for _, r := range []struct {
		sender   common.Address
		registry uint64
		checksum string
		uri      string
	}{
		{alice, first, "1 C.C.A. 144", "https://www.courtlistener.com/opinion/8857414/x/"},
		{alice, first, "1 C.C.A. 144", "https://www.courtlistener.com/opinion/8857414/y/"},
		{alice, first, "2 C.C.A. 7", "https://ex.test/0123456789abcdef"},
		{bob, second, "1 C.C.A. 144", "https://ex.test/o"},
	} {
		_, err := k.AddRecord(ctx, r.sender.Bytes(), record(r.registry, r.checksum, r.uri))
		require.NoError(t, err)
	}

	_, err = k.AddRegistry(ctx, alice.Bytes(), "US-CA1", "", "")
	require.NoError(t, err)

	for _, g := range []struct{ checksum, role string }{
		{"", "editor"},
		{"1 C.C.A. 144", "admin"},
	} {
		_, err := msgs.GrantRole(ctx, &types.MsgGrantRole{
			Admin:      sdk.AccAddress(alice.Bytes()).String(),
			Address:    sdk.AccAddress(bob.Bytes()).String(),
			RegistryId: first,
			Checksum:   g.checksum,
			Role:       g.role,
		})
		require.NoError(t, err)
	}
}

func record(registryID uint64, checksum, uri string) types.Record {
	return types.Record{
		Uri:          uri,
		Checksum:     checksum,
		ChecksumAlgo: "cite-canonical-v1",
		Metadata:     `{"cluster":8857414,"name":"Richmond v. Atwood"}`,
		Status:       "Active",
		RegistryId:   registryID,
	}
}

func readFixture(t *testing.T) map[common.Hash]common.Hash {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(fixturePath))
	require.NoError(t, err, "run `git submodule update --init contracts`")

	var fixture map[string]string
	require.NoError(t, json.Unmarshal(raw, &fixture))
	require.NotEmpty(t, fixture)
	slots := make(map[common.Hash]common.Hash, len(fixture))
	for slot, value := range fixture {
		slots[common.HexToHash(slot)] = common.HexToHash(value)
	}
	return slots
}
