package precompile

import (
	"github.com/MANTRA-Chain/inveniam/x/document/types"
)

func FromABIDocument(doc Document) types.Document {
	return types.Document{
		Document:     doc.Name,
		Denom:        doc.Denom,
		Uri:          doc.Uri,
		Checksum:     doc.Checksum,
		ChecksumAlgo: doc.ChecksumAlgo,
		Timestamp:    doc.Timestamp,
		Figi:         doc.Figi,
		IndividualId: doc.IndividualId,
	}
}

func ToABIDocument(doc types.Document) Document {
	return Document{
		Name:         doc.Document,
		Denom:        doc.Denom,
		Uri:          doc.Uri,
		Checksum:     doc.Checksum,
		ChecksumAlgo: doc.ChecksumAlgo,
		Timestamp:    doc.Timestamp,
		Figi:         doc.Figi,
		IndividualId: doc.IndividualId,
	}
}
