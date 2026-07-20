package data

import _ "embed"

// RegistriesJSON is this upgrade's slice of the mainnet-full-export registries.json: tranche 4
// only (1,228 registries) — see /mainnet-full-export/README.md for provenance. Committed to
// source so every validator builds an identical, code-reviewed registry list.
//
//go:embed registries.json
var RegistriesJSON []byte

// ManifestJSON is this upgrade's slice of the mainnet-full-export manifest.json: per-file
// record counts and sha256 digests for tranche 4's files only. Committed to source — rather
// than read alongside the tranche files themselves — so the expected hashes used to verify
// those (large, gitignored, disk-provided) files at upgrade time are fixed by reviewed code
// and can't be swapped together with a tampered/incomplete export.
//
//go:embed manifest.json
var ManifestJSON []byte
