package types

import (
	"cosmossdk.io/collections"
)

const (
	// ModuleName defines the module name
	ModuleName = "document"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName
)

var (
	ParamsKey                 = collections.NewPrefix(0)
	DocumentKeyPrefix         = collections.NewPrefix(1)
	DocumentCountersKeyPrefix = collections.NewPrefix(2)
)

func DocumentKey(documentId, denom string) []byte {
	return append(DocumentKeyByDenom(denom), documentId...)
}

func DocumentKeyByDenom(denom string) []byte {
	return append(DocumentKeyPrefix, denom...)
}
