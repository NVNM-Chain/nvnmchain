// Package evmlayout writes x/anchoring state as the anchoring contract's storage, so the corpus
// reaches the contract through a genesis dump rather than transactions. The slot constants mirror
// contracts/layout/anchoring.json, and the tests check a migration against the slots the contract
// itself wrote.
package evmlayout

import (
	"encoding/binary"
	"math/big"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Slots of the contract's state variables, in declaration order; AnchoringRBAC's come first.
const (
	SlotRoleAdmin uint64 = iota
	SlotRoleMembers
	SlotRoleMemberCount
	SlotHeader // _moduleAdmin in the low 20 bytes, _registryCount in the 8 above
	SlotRegistries
	SlotRecordCount
	SlotLatestIndex
	SlotRecords
	SlotRecordIDByChecksum
	SlotRegistriesByChecksum
	SlotRegistriesByName
)

// A Registry's fields, one slot each from its base.
const (
	RegistryID uint64 = iota
	RegistryName
	RegistryDescription
	RegistryCreator
	RegistryCreatedAt
	RegistryMetadata
)

// A Record's fields, one slot each from its base; the last packs four (see packRecord).
const (
	RecordURI uint64 = iota
	RecordChecksum
	RecordChecksumAlgo
	RecordMetadata
	RecordTimestamp
	RecordStatus
	RecordPacked
)

// A Write is one storage slot and its value.
type Write struct {
	Slot  common.Hash
	Value common.Hash
}

// Word is v as a storage word: big-endian and left-padded, as Solidity pads a mapping key.
func Word(v uint64) common.Hash {
	var h common.Hash
	binary.BigEndian.PutUint64(h[24:], v)
	return h
}

// at is the slot n words past base: a struct's field or an array's word.
func at(base common.Hash, n uint64) common.Hash {
	return common.BigToHash(new(big.Int).Add(base.Big(), new(big.Int).SetUint64(n)))
}

func mapUint64(key uint64, parent common.Hash) common.Hash {
	return crypto.Keccak256Hash(Word(key).Bytes(), parent.Bytes())
}

func mapHash(key, parent common.Hash) common.Hash {
	return crypto.Keccak256Hash(key.Bytes(), parent.Bytes())
}

// MapString is mapping[key] for a string key, which is hashed unpadded.
func MapString(key string, parent common.Hash) common.Hash {
	return crypto.Keccak256Hash([]byte(key), parent.Bytes())
}

// ArrayData is where a dynamic array's elements start; its length stays at lengthSlot.
func ArrayData(lengthSlot common.Hash) common.Hash {
	return crypto.Keccak256Hash(lengthSlot.Bytes())
}

// stringWrites is a Solidity string at base. Under 32 bytes it sits in the slot, with 2·len in
// the last byte; otherwise the slot holds 2·len+1 and the bytes run from keccak(base).
func stringWrites(base common.Hash, s string) []Write {
	b := []byte(s)
	if len(b) < 32 {
		var packed common.Hash
		copy(packed[:], b)
		packed[31] = byte(len(b) * 2)
		return []Write{{Slot: base, Value: packed}}
	}

	writes := []Write{{Slot: base, Value: Word(uint64(len(b)*2 + 1))}}
	start := ArrayData(base)
	for i := 0; i < len(b); i += 32 {
		var word common.Hash
		copy(word[:], b[i:])
		writes = append(writes, Write{Slot: at(start, uint64(i/32)), Value: word})
	}
	return writes
}

func registryBase(registryID uint64) common.Hash {
	return mapUint64(registryID, Word(SlotRegistries))
}

func recordBase(registryID, recordID, index uint64) common.Hash {
	return mapUint64(index, mapUint64(recordID, mapUint64(registryID, Word(SlotRecords))))
}

func registryWrites(base common.Hash, reg types.Registry) []Write {
	writes := make([]Write, 0, 8)
	writes = append(writes, Write{Slot: at(base, RegistryID), Value: Word(reg.Id)})
	writes = append(writes, stringWrites(at(base, RegistryName), reg.Name)...)
	writes = append(writes, stringWrites(at(base, RegistryDescription), reg.Description)...)
	writes = append(writes, stringWrites(at(base, RegistryCreator), reg.Creator)...)
	writes = append(writes, stringWrites(at(base, RegistryCreatedAt), reg.CreatedAt)...)
	writes = append(writes, stringWrites(at(base, RegistryMetadata), reg.Metadata)...)
	return writes
}

func recordWrites(base common.Hash, rec types.Record) []Write {
	writes := make([]Write, 0, 16)
	writes = append(writes, stringWrites(at(base, RecordURI), rec.Uri)...)
	writes = append(writes, stringWrites(at(base, RecordChecksum), rec.Checksum)...)
	writes = append(writes, stringWrites(at(base, RecordChecksumAlgo), rec.ChecksumAlgo)...)
	writes = append(writes, stringWrites(at(base, RecordMetadata), rec.Metadata)...)
	writes = append(writes, stringWrites(at(base, RecordTimestamp), rec.Timestamp)...)
	writes = append(writes, stringWrites(at(base, RecordStatus), rec.Status)...)
	writes = append(writes, Write{
		Slot:  at(base, RecordPacked),
		Value: packRecord(rec.RecordId, rec.Index, rec.IsLatest, rec.RegistryId),
	})
	return writes
}

// idsWrites is a uint64[] under key: its length at the hashed slot, then four ids to a word,
// the first in the low 8 bytes.
func idsWrites(table common.Hash, key string, list []uint64) []Write {
	lengthSlot := MapString(key, table)
	writes := []Write{{Slot: lengthSlot, Value: Word(uint64(len(list)))}}
	data := ArrayData(lengthSlot)
	for i, id := range list {
		if i%4 == 0 {
			writes = append(writes, Write{Slot: at(data, uint64(i/4))})
		}
		end := 32 - 8*(i%4)
		binary.BigEndian.PutUint64(writes[len(writes)-1].Value[end-8:end], id)
	}
	return writes
}

// packRecord is a Record's last slot: recordId, index, isLatest and registryId at byte offsets
// 0, 8, 16 and 17 from the low end.
func packRecord(recordID, index uint64, isLatest bool, registryID uint64) common.Hash {
	var h common.Hash
	binary.BigEndian.PutUint64(h[24:32], recordID)
	binary.BigEndian.PutUint64(h[16:24], index)
	if isLatest {
		h[15] = 1
	}
	binary.BigEndian.PutUint64(h[7:15], registryID)
	return h
}
