package nameindex

import (
	"path/filepath"

	"github.com/spf13/cast"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
)

// TomlSection is the app.toml section name node operators use to opt into
// the registry name index.
const TomlSection = "anchoring-name-index"

// Config controls whether this node builds and serves the registry name
// index, and where its backing SQLite file lives.
type Config struct {
	// Enabled turns the index on: it registers the ABCIListener, backfills
	// from the Registries collection on every start, and serves
	// Query/SearchRegistriesByName. Disabled by default.
	Enabled bool
	// DBPath is the SQLite file path. Relative paths are resolved under
	// <home>/data.
	DBPath string
}

// ReadConfig reads the [anchoring-name-index] section from AppOptions
// (app.toml), resolving a relative db-path under <home>/data.
func ReadConfig(appOpts servertypes.AppOptions, homeDir string) Config {
	enabled := cast.ToBool(appOpts.Get(TomlSection + ".enabled"))
	dbPath := cast.ToString(appOpts.Get(TomlSection + ".db-path"))
	if dbPath == "" {
		dbPath = "anchoring_name_index.db"
	}
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(homeDir, "data", dbPath)
	}
	return Config{Enabled: enabled, DBPath: dbPath}
}

// ConfigTemplate is appended to the node's app.toml template so operators can
// see and edit these settings.
const ConfigTemplate = `
###############################################################################
###                       Anchoring Name Index                            ###
###############################################################################

[anchoring-name-index]

# Enabled builds and serves an opt-in local SQLite index of anchoring registry
# names, supporting exact/prefix/suffix/contains lookups via
# Query/SearchRegistriesByName. It is not part of consensus: disabled nodes
# simply don't serve that one RPC. Disabled by default.
enabled = {{ .AnchoringNameIndex.Enabled }}

# DBPath is the SQLite file backing the index. Relative paths are resolved
# under <home>/data.
db-path = "{{ .AnchoringNameIndex.DBPath }}"
`

// DefaultConfig returns the disabled-by-default config used to seed a fresh
// app.toml.
func DefaultConfig() Config {
	return Config{Enabled: false, DBPath: "anchoring_name_index.db"}
}
