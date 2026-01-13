package chainsuite

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/types"
)

type ChainScope int

const (
	ChainScopeSuite ChainScope = iota
	ChainScopeTest  ChainScope = iota
)

type SuiteConfig struct {
	ChainSpec      *interchaintest.ChainSpec
	UpgradeOnSetup bool
	CreateRelayer  bool
	Scope          ChainScope
}

const (
	CommitTimeout           = 4 * time.Second
	Uom                     = "anvnm"
	HumanCoinUnit           = "nvnm"
	GovMinDepositAmount     = 1000
	GovDepositPeriod        = 60 * time.Second
	GovVotingPeriod         = 40 * time.Second
	DowntimeJailDuration    = 60 * time.Second // mantrachain requires at least 1 minute
	ProviderSlashingWindow  = 10
	SlashFractionDoubleSign = "0.05"
	// Gas prices must meet the EVM fee market minimum (base fee ~1e9 wei)
	// Using 1000000000 (1 gwei equivalent) to ensure transactions pass
	// Each chain must use its own native denom for gas prices
	GasPrices         = "1000000000" + Uom  // Consumer chain gas prices (anvnm)
	ProviderGasPrices = "1000000000amantra" // Provider chain gas prices (amantra)
	ValidatorCount    = 6
	UpgradeDelta      = 12
	// ValidatorFunds must be large enough to cover stake + gas fees
	// DefaultPowerReduction for 18-decimal tokens is 1e18 (AttoPowerReduction)
	// Each validator needs at least 1e18 to have 1 voting power
	// With gas prices of 1e9 and typical tx gas of 200k, each tx costs ~2e14 amantra
	// Setting to 9e18 (9 tokens) to cover stake (3e18) plus many transactions
	// Note: max int64 is ~9.2e18, so we use 9e18 to stay within limits
	ValidatorFunds         = 9_000_000_000_000_000_000 // 9e18 (9 tokens in 18-decimal representation)
	ChainSpawnWait         = 155 * time.Second
	SlashingWindowConsumer = 20
	BlocksPerDistribution  = 10
	ValidatorMoniker       = "validator"

	// Provider chain constants
	ProviderBin = "mantrachaind"
	// Chain ID must be in EVMChainIDMap (mantra-1, mantra-dukong-1, mantra-canary-net-1)
	// or parseable as <name>_<EIP155>-<epoch> format for EVM compatibility
	ProviderChainID           = "mantra-dukong-1"
	ProviderBech32Prefix      = "mantra"
	ProviderUnbondingTime     = 10 * time.Second
	ProviderReplenishPeriod   = "2s"
	ProviderReplenishFraction = "1.00"
	// ProviderGovModuleAddress is the governance module address for the provider chain
	// This is a deterministic address based on the cosmos-sdk gov module
	ProviderGovModuleAddress = "mantra10d07y265gmmuvt4z0w9aw880jnsr700jz4h7cr"

	// Consumer chain constants
	ConsumerBin = "inveniamd"
	// Chain ID must be in EVMChainIDMap (inveniam-1, manveniam-1, inveniam-canary-net-1)
	// or parseable as <name>_<EIP155>-<epoch> format for EVM compatibility
	ConsumerChainID          = "manveniam-1"
	ConsumerBech32Prefix     = "inveniam"
	ConsumerGovModuleAddress = "inveniam10d07y265gmmuvt4z0w9aw880jnsr700jdl2ath"

	// Test wallet constants
	// Only create 2 test wallets since that's all the tests need
	TestWalletsNumber = 2

	// Pre-funded test account for consumer chain
	// This address is derived from a known mnemonic for testing
	// Mnemonic: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	// We fund both coin-type 118 (Cosmos standard) and coin-type 60 (EVM) addresses for compatibility
	TestAccountMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

	// Address derived with coin-type 118 (Cosmos standard path m/44'/118'/0'/0/0)
	// Used by interchaintest's BuildWallet and standard Cosmos operations
	TestAccountAddress = "inveniam10sds9dt423w57fpy8s3fhjd9autynv6xg52prm"

	// Address for EVM compatibility - derived from EVM address bytes
	// EVM address 0x9858EfFD232B4033E47d90003D41EC34EcaEda94 corresponds to this Cosmos address
	// This is the address that eth_getBalance will query in the bank module
	TestAccountAddressEVM = "inveniam1npvwllfr9dqr8erajqqr6s0vxnk2ak55tjyq5h"

	// TestAccountFunds is 5e18 (5 tokens) - enough for many test transactions
	// With gas prices of 1e9 and typical tx gas of 200k with 2x adjustment, each tx costs ~4e14
	// Both addresses will be funded with this amount in genesis
	TestAccountFunds = 5_000_000_000_000_000_000
)

func (c SuiteConfig) Merge(other SuiteConfig) SuiteConfig {
	if c.ChainSpec == nil {
		c.ChainSpec = other.ChainSpec
	} else if other.ChainSpec != nil {
		c.ChainSpec.ChainConfig = c.ChainSpec.MergeChainSpecConfig(other.ChainSpec.ChainConfig)
		if other.ChainSpec.Name != "" {
			c.ChainSpec.Name = other.ChainSpec.Name
		}
		if other.ChainSpec.ChainName != "" {
			c.ChainSpec.ChainName = other.ChainSpec.ChainName
		}
		if other.ChainSpec.Version != "" {
			c.ChainSpec.Version = other.ChainSpec.Version
		}
		if other.ChainSpec.NoHostMount != nil {
			c.ChainSpec.NoHostMount = other.ChainSpec.NoHostMount
		}
		if other.ChainSpec.NumValidators != nil {
			c.ChainSpec.NumValidators = other.ChainSpec.NumValidators
		}
		if other.ChainSpec.NumFullNodes != nil {
			c.ChainSpec.NumFullNodes = other.ChainSpec.NumFullNodes
		}
	}
	c.UpgradeOnSetup = other.UpgradeOnSetup
	c.CreateRelayer = other.CreateRelayer
	c.Scope = other.Scope
	return c
}

func DefaultGenesisAmounts(denom string) func(i int) (types.Coin, types.Coin) {
	// DefaultPowerReduction for 18-decimal tokens is 1e18 (AttoPowerReduction)
	// Validator stakes must be >= 1e18 to have at least 1 voting power
	// Each validator needs at least 1 token (1e18 base units) to have voting power
	return func(i int) (types.Coin, types.Coin) {
		return types.Coin{
				Denom:  denom,
				Amount: sdkmath.NewInt(ValidatorFunds),
			}, types.Coin{
				Denom: denom,
				Amount: sdkmath.NewInt([ValidatorCount]int64{
					// Stake amounts in base units (amantra for provider, anvnm for consumer)
					// Each stake must be >= 1e18 (1 token) to have voting power
					// With DefaultPowerReduction = 1e18, voting power = stake / 1e18
					3_000_000_000_000_000_000, // 3 tokens = 3 voting power
					2_900_000_000_000_000_000, // 2.9 tokens = 2 voting power
					2_000_000_000_000_000_000, // 2 tokens = 2 voting power
					1_500_000_000_000_000_000, // 1.5 tokens = 1 voting power
					1_200_000_000_000_000_000, // 1.2 tokens = 1 voting power
					1_000_000_000_000_000_000, // 1 token = 1 voting power
				}[i]),
			}
	}
}

// ConsumerGenesisAmounts returns genesis amounts for consumer chains
// Consumer chains don't stake (validators are inherited from provider via ICS)
// The balance here is used for the faucet and test accounts
// With gas prices of 1e9 and typical tx gas of 200k with 2x adjustment,
// each tx costs ~4e14. Need enough for multiple test wallet funding txs.
func ConsumerGenesisAmounts(denom string) func(i int) (types.Coin, types.Coin) {
	return func(i int) (types.Coin, types.Coin) {
		return types.Coin{
				Denom:  denom,
				Amount: sdkmath.NewInt(ValidatorFunds), // 9e18 - enough for many txs
			}, types.Coin{
				Denom:  denom,
				Amount: sdkmath.NewInt(0), // Consumer validators don't stake
			}
	}
}

func DefaultSuiteConfig(env Environment) SuiteConfig {
	fullNodes := 0
	validators := ValidatorCount
	coinDecimals := InterchaintestCoinDecimals
	var repository string
	if env.DockerRegistry == "" {
		repository = env.MantraImageName
	} else {
		repository = fmt.Sprintf("%s/%s", env.DockerRegistry, env.MantraImageName)
	}
	return SuiteConfig{
		ChainSpec: &interchaintest.ChainSpec{
			ChainName:     "mantra",
			Name:          "mantra",
			NumFullNodes:  &fullNodes,
			NumValidators: &validators,
			Version:       env.OldMantraImageVersion,
			ChainConfig: ibc.ChainConfig{
				Denom:         ProviderDenom,
				CoinDecimals:  &coinDecimals,
				GasPrices:     ProviderGasPrices,
				Gas:           "auto",
				GasAdjustment: 2.0,
				ConfigFileOverrides: map[string]any{
					"config/config.toml": DefaultConfigToml(),
				},
				Images: []ibc.DockerImage{{
					Repository: repository,
					Version:    env.OldMantraImageVersion,
					UIDGID:     "1025:1025", // this is the user in heighliner docker images
				}},
				Type:                 "cosmos",
				Name:                 "mantra",
				ChainID:              "mantra-test-1",
				Bin:                  "mantrachaind",
				Bech32Prefix:         "mantra",
				CoinType:             "118",
				TrustingPeriod:       "48h",
				NoHostMount:          false,
				ModifyGenesis:        cosmos.ModifyGenesis(DefaultGenesis()),
				ModifyGenesisAmounts: DefaultGenesisAmounts(ProviderDenom),
				EncodingConfig:       EVMEncoding(),
			},
		},
	}
}

func DefaultConfigToml() testutil.Toml {
	configToml := make(testutil.Toml)
	consensusToml := make(testutil.Toml)
	consensusToml["timeout_commit"] = CommitTimeout
	configToml["consensus"] = consensusToml
	configToml["block_sync"] = false
	configToml["fast_sync"] = false
	return configToml
}

func DefaultGenesis() []cosmos.GenesisKV {
	return []cosmos.GenesisKV{
		// Staking params - bond_denom must match the native token
		cosmos.NewGenesisKV("app_state.staking.params.bond_denom", ProviderDenom),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", GovVotingPeriod.String()),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", GovDepositPeriod.String()),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", ProviderDenom),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", strconv.Itoa(GovMinDepositAmount)),
		cosmos.NewGenesisKV("app_state.slashing.params.signed_blocks_window", strconv.Itoa(ProviderSlashingWindow)),
		cosmos.NewGenesisKV("app_state.slashing.params.downtime_jail_duration", DowntimeJailDuration.String()),
		// EVM configuration - mantrachain uses "aatom" as the native/EVM denom
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
							"exponent": 18,
						},
					},
				},
			},
		},
	}
}
