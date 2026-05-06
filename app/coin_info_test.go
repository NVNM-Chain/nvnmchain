package app

import (
	"os"
	"path/filepath"
	"testing"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/stretchr/testify/require"
)

// writeGenesis writes body to <tmp>/config/genesis.json and returns the
// home path.
func writeGenesis(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config")
	require.NoError(t, os.MkdirAll(cfg, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(cfg, "genesis.json"), []byte(body), 0o644))
	return dir
}

// info18 builds an 18-decimal EvmCoinInfo from a single denom (the
// no-metadata fallback shape).
func info18(denom string) evmtypes.EvmCoinInfo {
	return evmtypes.EvmCoinInfo{
		Denom:         denom,
		ExtendedDenom: denom,
		DisplayDenom:  denom,
		Decimals:      evmtypes.EighteenDecimals.Uint32(),
	}
}

func TestLoadEvmCoinInfoFromGenesis_Success(t *testing.T) {
	const ibc = "ibc/AAAA"
	enriched := evmtypes.EvmCoinInfo{
		Denom: ibc, ExtendedDenom: ibc, DisplayDenom: "wmantrausd", Decimals: 18,
	}
	fullBody := func(evmFirst bool) string {
		bank := `"bank":{
			"balances":[{"address":"a","coins":[{"denom":"x","amount":"1"}]}],
			"supply":[{"denom":"x","amount":"1"}],
			"denom_metadata":[{
				"base":"ibc/AAAA","display":"wmantrausd",
				"denom_units":[
					{"denom":"ibc/AAAA","exponent":0},
					{"denom":"wmantrausd","exponent":18}
				]
			}]
		}`
		evm := `"evm":{"params":{"evm_denom":"ibc/AAAA"}}`
		if evmFirst {
			return `{"app_state":{` + evm + `,` + bank + `}}`
		}
		return `{"app_state":{` + bank + `,` + evm + `}}`
	}

	cases := []struct {
		name string
		body string
		want evmtypes.EvmCoinInfo
	}{
		{"full metadata, bank before evm", fullBody(false), enriched},
		{"full metadata, evm before bank", fullBody(true), enriched},
		{
			"metadata array empty falls back to evm_denom",
			`{"app_state":{
				"bank":{"denom_metadata":[]},
				"evm":{"params":{"evm_denom":"ibc/AAAA"}}
			}}`,
			info18(ibc),
		},
		{
			"bank section absent falls back to evm_denom",
			`{"app_state":{"evm":{"params":{"evm_denom":"ibc/AAAA"}}}}`,
			info18(ibc),
		},
		{
			"non-18 decimals with extended_denom_options",
			`{"app_state":{
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
			evmtypes.EvmCoinInfo{
				Denom: "umantra", ExtendedDenom: "extended/umantra", DisplayDenom: "mantra", Decimals: 6,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := loadEvmCoinInfoFromGenesis(writeGenesis(t, tc.body))
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestLoadEvmCoinInfoFromGenesis_Absent(t *testing.T) {
	// Empty home and home with no genesis.json both signal "in-memory test
	// app, fall back is fine" via errGenesisAbsent.
	for name, home := range map[string]string{
		"empty home":        "",
		"home without file": t.TempDir(),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := loadEvmCoinInfoFromGenesis(home)
			require.ErrorIs(t, err, errGenesisAbsent)
		})
	}
}

func TestLoadEvmCoinInfoFromGenesis_ParseError(t *testing.T) {
	// Real misconfigurations must surface as non-absent errors so
	// defaultEvmCoinInfo panics rather than silently using FutureStakingDenom.
	cases := map[string]string{
		"missing app_state":  `{"chain_id":"test"}`,
		"missing evm_denom":  `{"app_state":{"evm":{"params":{}}}}`,
		"malformed JSON":     `{not json`,
		"non-18 without ext": `{"app_state":{"bank":{"denom_metadata":[{"base":"u","display":"m","denom_units":[{"denom":"m","exponent":6}]}]},"evm":{"params":{"evm_denom":"u"}}}}`,
		"non-18 empty ext":   `{"app_state":{"bank":{"denom_metadata":[{"base":"u","display":"m","denom_units":[{"denom":"m","exponent":6}]}]},"evm":{"params":{"evm_denom":"u","extended_denom_options":{"extended_denom":""}}}}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := loadEvmCoinInfoFromGenesis(writeGenesis(t, body))
			require.Error(t, err)
			require.NotErrorIs(t, err, errGenesisAbsent)
		})
	}
}

func TestDefaultEvmCoinInfo(t *testing.T) {
	t.Run("falls back to FutureStakingDenom when genesis is absent", func(t *testing.T) {
		got := defaultEvmCoinInfo("")
		require.Equal(t, FutureStakingDenom, got.Denom)
		require.Equal(t, "nvnm", got.DisplayDenom)
	})

	t.Run("uses genesis evm_denom when present", func(t *testing.T) {
		home := writeGenesis(t, `{"app_state":{"evm":{"params":{"evm_denom":"ibc/BBBB"}}}}`)
		got := defaultEvmCoinInfo(home)
		require.Equal(t, "ibc/BBBB", got.Denom)
		require.Equal(t, "ibc/BBBB", got.DisplayDenom)
	})

	t.Run("panics on genesis parse failure", func(t *testing.T) {
		home := writeGenesis(t, `{not json`)
		require.Panics(t, func() { defaultEvmCoinInfo(home) })
	})
}
