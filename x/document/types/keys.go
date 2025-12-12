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
	DocumentByDenomKeyPrefix  = collections.NewPrefix(1)
	DocumentByChecksumKey     = collections.NewPrefix(2)
	DocumentCountersKeyPrefix = collections.NewPrefix(3)
)

func DocumentKey(documentId, denom string) []byte {
	return append(DocumentKeyByDenom(denom), documentId...)
}

func DocumentKeyByDenom(denom string) []byte {
	return append(DocumentByDenomKeyPrefix, denom...)
}
