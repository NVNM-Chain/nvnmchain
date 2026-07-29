package data

import _ "embed"

// RegistriesJSON is the mainnet-full-export registries.json payload: all 2,114 registry rows
// across all 4 tranches (see /mainnet-full-export/README.md for provenance). Committed to
// source so every validator builds an identical, code-reviewed registry list.
//
//go:embed registries.json
var RegistriesJSON []byte

// ManifestJSON is the mainnet-full-export manifest.json payload: per-file record counts and
// sha256 digests (of the .gz as shipped and of the uncompressed stream) for every tranche
// file. Committed to source — rather than read alongside the tranche files themselves — so
// the expected hashes used to verify those (large, gitignored, disk-provided) files at
// upgrade time are fixed by reviewed code and can't be swapped together with a
// tampered/incomplete export.
//
//go:embed manifest.json
var ManifestJSON []byte
