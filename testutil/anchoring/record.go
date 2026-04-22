package anchoring

import (
	"github.com/MANTRA-Chain/nvnmchain/x/anchoring/types"
)

// ValidRecord returns a minimal valid Record for the given registry, suitable
// for use in tests. The checksum is derived from the registry name so that
// different registries get distinct checksums.
func ValidRecord(registry string) types.Record {
	checksum := SHA256Hex(registry + "-record")
	return types.Record{
		Registry:     registry,
		Uri:          "ipfs://bafy...",
		Checksum:     checksum,
		ChecksumAlgo: "sha256",
		Metadata:     `{"k":"v"}`,
		Status:       "active",
	}
}
