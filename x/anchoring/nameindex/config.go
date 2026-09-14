package nameindex

import (
	"path/filepath"

	"github.com/spf13/cast"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
)

// TomlSection is the app.toml section name node operators use to opt into
// the registry name index.
const TomlSection = "anchoring-name-index"

const defaultDBPath = "anchoring_name_index.db"

// Config controls whether this node builds and serves the registry name
// index, and where its backing SQLite file lives.
type Config struct {
	// Enabled registers the ABCIListener, backfills on start, and serves
	// Query/SearchRegistriesByName. Off by default.
	Enabled bool
	// DBPath is the SQLite file. Relative paths resolve under <home>/data.
	DBPath string
}

// ReadConfig reads the [anchoring-name-index] section from AppOptions
// (app.toml), resolving a relative db-path under <home>/data.
func ReadConfig(appOpts servertypes.AppOptions, homeDir string) Config {
	enabled := cast.ToBool(appOpts.Get(TomlSection + ".enabled"))
	dbPath := cast.ToString(appOpts.Get(TomlSection + ".db-path"))
	if dbPath == "" {
		dbPath = defaultDBPath
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

# Build and serve a local SQLite index of anchoring registry names for
# Query/SearchRegistriesByName and the registriesByName precompile method.
# Not part of consensus: a disabled node just does not serve that lookup.
enabled = {{ .AnchoringNameIndex.Enabled }}

# SQLite file backing the index. Relative paths resolve under <home>/data.
db-path = "{{ .AnchoringNameIndex.DBPath }}"
`

// DefaultConfig returns the disabled-by-default config used to seed a fresh
// app.toml.
func DefaultConfig() Config {
	return Config{DBPath: defaultDBPath}
}
