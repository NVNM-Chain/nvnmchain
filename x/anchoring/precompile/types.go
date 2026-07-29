//go:build uint256

package precompile

import (
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
)

func FromABIRecord(rec Record) types.Record {
	return types.Record{
		Uri:          rec.Uri,
		Checksum:     rec.Checksum,
		ChecksumAlgo: rec.ChecksumAlgo,
		Metadata:     rec.Metadata,
		Timestamp:    rec.Timestamp,
		Status:       rec.Status,
		RecordId:     rec.RecordId,
		Index:        rec.Index,
		IsLatest:     rec.IsLatest,
		RegistryId:   rec.RegistryId,
	}
}

func ToABIRecord(rec types.Record) Record {
	return Record{
		Uri:          rec.Uri,
		Checksum:     rec.Checksum,
		ChecksumAlgo: rec.ChecksumAlgo,
		Metadata:     rec.Metadata,
		Timestamp:    rec.Timestamp,
		Status:       rec.Status,
		RecordId:     rec.RecordId,
		Index:        rec.Index,
		IsLatest:     rec.IsLatest,
		RegistryId:   rec.RegistryId,
	}
}

func ToABIRegistry(reg types.Registry) Registry {
	return Registry{
		Id:          reg.Id,
		Name:        reg.Name,
		Description: reg.Description,
		Creator:     reg.Creator,
		CreatedAt:   reg.CreatedAt,
		Metadata:    reg.Metadata,
	}
}
