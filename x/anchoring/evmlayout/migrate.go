package evmlayout

import (
	"encoding/binary"
	"fmt"
	"maps"
	"slices"

	"cosmossdk.io/collections"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/rbac"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// Sink takes each slot a migration computes. Returning an error stops the walk.
type Sink func(Write) error

// Migrate writes the module's state as the contract's storage, each slot once. It copies rather
// than replays, so creators and timestamps keep the strings they were written with.
func Migrate(ctx sdk.Context, k keeper.Keeper, sink Sink) error {
	emit := skipZero(sink)
	for _, step := range []struct {
		what string
		run  func(sdk.Context, keeper.Keeper, Sink) error
	}{
		{"registries", migrateRegistries},
		{"record counts", migrateRecordCounts},
		{"records", migrateRecords},
		{"latest indexes", migrateLatestIndexes},
		{"checksum lookup", migrateRecordIDs},
		{"header", migrateHeader},
		{"checksum index", migrateChecksumIndex},
		{"roles", migrateRoles},
	} {
		if err := step.run(ctx, k, emit); err != nil {
			return fmt.Errorf("migrating %s: %w", step.what, err)
		}
	}
	return nil
}

// skipZero drops zero writes: an unwritten slot already reads as zero.
func skipZero(sink Sink) Sink {
	return func(w Write) error {
		if w.Value == (common.Hash{}) {
			return nil
		}
		return sink(w)
	}
}

// migrateRegistries writes each registry and, from the same walk, the name index: names are not
// unique, so each gets a list of ids, in sorted name order so a dump is reproducible.
func migrateRegistries(ctx sdk.Context, k keeper.Keeper, emit Sink) error {
	var previous uint64
	byName := map[string][]uint64{}
	err := k.Registries.Walk(ctx, nil, func(id uint64, registry types.Registry) (bool, error) {
		// The contract pages registries by arithmetic, so a gap would read back as an empty
		// registry. InitGenesis allows gaps, so refuse them here.
		if id != previous+1 {
			return true, fmt.Errorf("ids must run from 1 with no gaps, got %d after %d", id, previous)
		}
		previous = id
		byName[registry.Name] = append(byName[registry.Name], id) // ascending, as the contract appends
		return false, emitAll(emit, registryWrites(registryBase(id), registry))
	})
	if err != nil {
		return err
	}
	// The header's count is where the paging ends, so it has to be the last id.
	count, err := k.RegistryCount.Get(ctx)
	if err != nil {
		return err
	}
	if count != previous {
		return fmt.Errorf("RegistryCount is %d but the ids end at %d", count, previous)
	}
	for _, name := range slices.Sorted(maps.Keys(byName)) {
		if err := emitAll(emit, idsWrites(Word(SlotRegistriesByName), name, byName[name])); err != nil {
			return err
		}
	}
	return nil
}

func migrateRecordCounts(ctx sdk.Context, k keeper.Keeper, emit Sink) error {
	return k.RecordsCountByRegistry.Walk(ctx, nil, func(id, count uint64) (bool, error) {
		return false, emit(Write{Slot: mapUint64(id, Word(SlotRecordCount)), Value: Word(count)})
	})
}

func migrateRecords(ctx sdk.Context, k keeper.Keeper, emit Sink) error {
	return k.Records.Walk(ctx, nil,
		func(key collections.Triple[uint64, uint64, uint64], record types.Record) (bool, error) {
			base := recordBase(key.K1(), key.K2(), key.K3())
			return false, emitAll(emit, recordWrites(base, record))
		})
}

func migrateLatestIndexes(ctx sdk.Context, k keeper.Keeper, emit Sink) error {
	return k.RecordIndices.Walk(ctx, nil,
		func(key collections.Pair[uint64, uint64], index uint64) (bool, error) {
			return false, emit(Write{
				Slot:  mapUint64(key.K2(), mapUint64(key.K1(), Word(SlotLatestIndex))),
				Value: Word(index),
			})
		})
}

func migrateRecordIDs(ctx sdk.Context, k keeper.Keeper, emit Sink) error {
	return k.RecordIdByRegistryAndChecksum.Walk(ctx, nil,
		func(key collections.Pair[uint64, string], recordID uint64) (bool, error) {
			return false, emit(Write{
				Slot:  MapString(key.K2(), mapUint64(key.K1(), Word(SlotRecordIDByChecksum))),
				Value: Word(recordID),
			})
		})
}

func migrateHeader(ctx sdk.Context, k keeper.Keeper, emit Sink) error {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}
	admin, err := sdk.AccAddressFromBech32(params.Admin)
	if err != nil {
		return fmt.Errorf("params admin %q: %w", params.Admin, err)
	}
	count, err := k.RegistryCount.Get(ctx)
	if err != nil {
		return err
	}
	var header common.Hash
	copy(header[12:], common.BytesToAddress(admin).Bytes())
	binary.BigEndian.PutUint64(header[4:12], count)
	return emit(Write{Slot: Word(SlotHeader), Value: header})
}

// RecordIdByChecksumAndRegistry is keyed (checksum, registryId), so each checksum's registries
// arrive together and ascending, as the contract's list keeps them.
func migrateChecksumIndex(ctx sdk.Context, k keeper.Keeper, emit Sink) error {
	var current string
	var registries []uint64

	flush := func() error {
		if len(registries) == 0 {
			return nil
		}
		return emitAll(emit, idsWrites(Word(SlotRegistriesByChecksum), current, registries))
	}

	err := k.RecordIdByChecksumAndRegistry.Walk(ctx, nil,
		func(key collections.Pair[string, uint64], _ uint64) (bool, error) {
			if key.K1() != current {
				if err := flush(); err != nil {
					return true, err
				}
				current, registries = key.K1(), registries[:0]
			}
			registries = append(registries, key.K2())
			return false, nil
		})
	if err != nil {
		return err
	}
	return flush()
}

// The two rbac tables, plus the member count only the contract keeps. RoleMembers is keyed
// (role, account), so each role's count is known when its group ends.
func migrateRoles(ctx sdk.Context, k keeper.Keeper, emit Sink) error {
	err := k.RBAC.RoleAdmins.Walk(ctx, nil, func(role rbac.Role, adminRole []byte) (bool, error) {
		return false, emit(Write{
			Slot:  mapHash(common.Hash(role), Word(SlotRoleAdmin)),
			Value: common.BytesToHash(adminRole),
		})
	})
	if err != nil {
		return err
	}

	var current rbac.Role
	var members uint64

	flushCount := func() error {
		if members == 0 {
			return nil
		}
		return emit(Write{
			Slot:  mapHash(common.Hash(current), Word(SlotRoleMemberCount)),
			Value: Word(members),
		})
	}

	err = k.RBAC.RoleMembers.Walk(ctx, nil,
		func(key collections.Pair[rbac.Role, sdk.AccAddress], _ []byte) (bool, error) {
			if key.K1() != current {
				if err := flushCount(); err != nil {
					return true, err
				}
				current, members = key.K1(), 0
			}
			members++
			return false, emit(Write{
				Slot: mapHash(
					common.BytesToHash(key.K2()),
					mapHash(common.Hash(key.K1()), Word(SlotRoleMembers)),
				),
				Value: Word(1),
			})
		})
	if err != nil {
		return err
	}
	return flushCount()
}

func emitAll(emit Sink, writes []Write) error {
	for _, w := range writes {
		if err := emit(w); err != nil {
			return err
		}
	}
	return nil
}
