package precompile

import (
	"github.com/MANTRA-Chain/inveniam/x/anchoring/types"
)

func FromABIRecord(rec Record) types.Record {
	return types.Record{
		Registry:     rec.Registry,
		Uri:          rec.Uri,
		Checksum:     rec.Checksum,
		ChecksumAlgo: rec.ChecksumAlgo,
		Metadata:     rec.Metadata,
		Timestamp:    rec.Timestamp,
		Status:       rec.Status,
		RecordId:     rec.RecordId,
		Index:        rec.Index,
		IsLatest:     rec.IsLatest,
	}
}

func ToABIRecord(rec types.Record) Record {
	return Record{
		Registry:     rec.Registry,
		Uri:          rec.Uri,
		Checksum:     rec.Checksum,
		ChecksumAlgo: rec.ChecksumAlgo,
		Metadata:     rec.Metadata,
		Timestamp:    rec.Timestamp,
		Status:       rec.Status,
		RecordId:     rec.RecordId,
		Index:        rec.Index,
		IsLatest:     rec.IsLatest,
	}
}

func ToABIRegistry(reg types.Registry) Registry {
	return Registry{
		Id:          reg.Id,
		Name:        reg.Name,
		Description: reg.Description,
		Creator:     reg.Creator,
		CreatedAt:   reg.CreatedAt,
	}
}
