package app

import (
	"os"
	"path/filepath"
	"testing"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/stretchr/testify/require"
)

// genesisHome writes body to <tmp>/config/genesis.json (skipped when body
// is empty) and returns the home path.
func genesisHome(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if body == "" {
		return dir
	}
	cfg := filepath.Join(dir, "config")
	require.NoError(t, os.MkdirAll(cfg, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(cfg, "genesis.json"), []byte(body), 0o644))
	return dir
}

// info18 builds an 18-decimal EvmCoinInfo from a single denom (used as
// Denom/ExtendedDenom/DisplayDenom — the no-metadata fallback shape).
func info18(denom string) evmtypes.EvmCoinInfo {
	return evmtypes.EvmCoinInfo{
		Denom:         denom,
		ExtendedDenom: denom,
		DisplayDenom:  denom,
		Decimals:      evmtypes.EighteenDecimals.Uint32(),
	}
}

func TestLoadEvmCoinInfoFromGenesis(t *testing.T) {
	const ibc = "ibc/AAAA"

	cases := []struct {
		name string
		body string
		want evmtypes.EvmCoinInfo // zero value means: expect ok=false
	}{
		{
			name: "full metadata enriches display + decimals",
			body: `{"app_state":{
				"bank":{
					"balances":[{"address":"a","coins":[{"denom":"x","amount":"1"}]}],
					"supply":[{"denom":"x","amount":"1"}],
					"denom_metadata":[{
						"base":"ibc/AAAA","display":"wmantrausd",
						"denom_units":[
							{"denom":"ibc/AAAA","exponent":0},
							{"denom":"wmantrausd","exponent":18}
						]
					}]
				},
				"evm":{"params":{"evm_denom":"ibc/AAAA"}}
			}}`,
			want: evmtypes.EvmCoinInfo{
				Denom: ibc, ExtendedDenom: ibc, DisplayDenom: "wmantrausd", Decimals: 18,
			},
		},
		{
			name: "metadata array empty falls back to evm_denom",
			body: `{"app_state":{
				"bank":{"denom_metadata":[]},
				"evm":{"params":{"evm_denom":"ibc/AAAA"}}
			}}`,
			want: info18(ibc),
		},
		{
			name: "bank section absent falls back to evm_denom",
			body: `{"app_state":{"evm":{"params":{"evm_denom":"ibc/AAAA"}}}}`,
			want: info18(ibc),
		},
		{
			name: "non-18 decimals with extended_denom_options",
			body: `{"app_state":{
				"bank":{"denom_metadata":[{
					"base":"umantra","display":"mantra",
					"denom_units":[
						{"denom":"umantra","exponent":0},
						{"denom":"mantra","exponent":6}
					]
				}]},
				"evm":{"params":{
					"evm_denom":"umantra",
					"extended_denom_options":{"extended_denom":"extended/umantra"}
				}}
			}}`,
			want: evmtypes.EvmCoinInfo{
				Denom: "umantra", ExtendedDenom: "extended/umantra", DisplayDenom: "mantra", Decimals: 6,
			},
		},
		{
			name: "non-18 decimals without extended_denom_options is rejected",
			body: `{"app_state":{
				"bank":{"denom_metadata":[{
					"base":"umantra","display":"mantra",
					"denom_units":[{"denom":"mantra","exponent":6}]
				}]},
				"evm":{"params":{"evm_denom":"umantra"}}
			}}`,
		},
		{name: "missing evm_denom", body: `{"app_state":{"evm":{"params":{}}}}`},
		{name: "missing app_state", body: `{"chain_id":"test"}`},
		{name: "malformed JSON", body: `{not json`},
		{name: "missing genesis file"}, // body == "" → no file written
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := loadEvmCoinInfoFromGenesis(genesisHome(t, tc.body))
			if (tc.want == evmtypes.EvmCoinInfo{}) {
				require.False(t, ok)
				return
			}
			require.True(t, ok)
			require.Equal(t, tc.want, got)
		})
	}

	// Empty homePath skips the read entirely (in-memory test apps).
	_, ok := loadEvmCoinInfoFromGenesis("")
	require.False(t, ok)
}

func TestDefaultEvmCoinInfo(t *testing.T) {
	// No on-disk genesis → FutureStakingDenom fallback for in-memory apps.
	require.Equal(t, info18(FutureStakingDenom).Denom, defaultEvmCoinInfo("").Denom)
	require.Equal(t, "nvnm", defaultEvmCoinInfo("").DisplayDenom)

	// Genesis present → genesis wins.
	home := genesisHome(t, `{"app_state":{"evm":{"params":{"evm_denom":"ibc/BBBB"}}}}`)
	got := defaultEvmCoinInfo(home)
	require.Equal(t, "ibc/BBBB", got.Denom)
	require.Equal(t, "ibc/BBBB", got.DisplayDenom)
}
