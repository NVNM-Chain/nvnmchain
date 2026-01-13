package chainsuite

import (
	"strconv"
	"strings"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
)

const (
	// Provider chain native denom - mantrachain uses "amantra" for staking and EVM
	ProviderDenom         = "amantra"
	ProviderHumanCoinUnit = "mantra"
	// ProviderCoinDecimals is the actual token decimals (18 for atto-denomination)
	ProviderCoinDecimals = int64(18)
	// InterchaintestCoinDecimals is used only for interchaintest's faucet amount calculation.
	// We use 10 instead of 18 because interchaintest has an int64 overflow bug when
	// calculating faucet amount (100M * 10^decimals). With 10 decimals, faucet gets
	// 10^18 amantra which is sufficient for tests and avoids the overflow.
	InterchaintestCoinDecimals = int64(10)
)

// GetProviderSpec returns the chain spec for the provider chain (mantrachain)
func GetProviderSpec(env Environment, validatorCount int, modifiedGenesis []cosmos.GenesisKV) *interchaintest.ChainSpec {
	fullNodes := 0
	validators := validatorCount
	coinDecimals := InterchaintestCoinDecimals

	return &interchaintest.ChainSpec{
		// Leave Name empty to avoid looking up built-in configs
		ChainName:     "mantra",
		NumFullNodes:  &fullNodes,
		NumValidators: &validators,
		Version:       env.OldMantraImageVersion,
		ChainConfig: ibc.ChainConfig{
			Type:           "cosmos",
			Name:           "mantra",
			ChainID:        ProviderChainID,
			Bin:            ProviderBin,
			Bech32Prefix:   ProviderBech32Prefix,
			Denom:          ProviderDenom,
			CoinDecimals:   &coinDecimals,
			GasPrices:      ProviderGasPrices,
			GasAdjustment:  2.0,
			TrustingPeriod: "504h",
			CoinType:       "118",
			ConfigFileOverrides: map[string]any{
				"config/config.toml": DefaultConfigToml(),
			},
			Images: []ibc.DockerImage{{
				Repository: ProviderImageName(env),
				Version:    env.OldMantraImageVersion,
				UIDGID:     "1025:1025",
			}},
			ModifyGenesis:        cosmos.ModifyGenesis(modifiedGenesis),
			ModifyGenesisAmounts: DefaultGenesisAmounts(ProviderDenom),
			EncodingConfig:       EVMEncoding(),
		},
	}
}

// ProviderModifiedGenesis returns the genesis modifications for the provider chain
func ProviderModifiedGenesis() []cosmos.GenesisKV {
	return []cosmos.GenesisKV{
		// Staking params - bond_denom must match the native token
		cosmos.NewGenesisKV("app_state.staking.params.unbonding_time", ProviderUnbondingTime.String()),
		cosmos.NewGenesisKV("app_state.staking.params.bond_denom", ProviderDenom),
		cosmos.NewGenesisKV("app_state.staking.params.max_validators", strconv.Itoa(ValidatorCount)),
		// Mint params - mint_denom must match the native token
		cosmos.NewGenesisKV("app_state.mint.params.mint_denom", ProviderDenom),
		// Gov params
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", GovVotingPeriod.String()),
		cosmos.NewGenesisKV("app_state.gov.params.expedited_voting_period", (GovVotingPeriod / 2).String()), // Must be less than voting_period
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", GovDepositPeriod.String()),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", ProviderDenom),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", strconv.Itoa(GovMinDepositAmount)),
		// Slashing params
		cosmos.NewGenesisKV("app_state.slashing.params.signed_blocks_window", strconv.Itoa(ProviderSlashingWindow)),
		cosmos.NewGenesisKV("app_state.slashing.params.downtime_jail_duration", DowntimeJailDuration.String()),
		// Provider params
		cosmos.NewGenesisKV("app_state.provider.params.slash_meter_replenish_period", ProviderReplenishPeriod),
		cosmos.NewGenesisKV("app_state.provider.params.slash_meter_replenish_fraction", ProviderReplenishFraction),
		cosmos.NewGenesisKV("app_state.provider.params.blocks_per_epoch", "1"),
		// EVM configuration - use the same denom as staking
		{
			Key:   "app_state.evm.params.evm_denom",
			Value: ProviderDenom,
		},
		{
			Key:   "app_state.evm.params.extended_denom_options.extended_denom",
			Value: ProviderDenom,
		},
		// Bank denom metadata required by EVM module
		{
			Key: "app_state.bank.denom_metadata",
			Value: []map[string]interface{}{
				{
					"base":        ProviderDenom,
					"display":     ProviderHumanCoinUnit,
					"name":        "MANTRA Token",
					"symbol":      strings.ToUpper(ProviderHumanCoinUnit),
					"description": "The native token of the MANTRA network",
					"denom_units": []map[string]interface{}{
						{
							"denom":    ProviderDenom,
							"exponent": 0,
						},
						{
							"denom":    ProviderHumanCoinUnit,
							"exponent": ProviderCoinDecimals,
						},
					},
				},
			},
		},
	}
}

// ProviderImageName returns the full image name for the provider
func ProviderImageName(env Environment) string {
	if env.DockerRegistry == "" {
		return env.MantraImageName
	}
	return env.DockerRegistry + "/" + env.MantraImageName
}
