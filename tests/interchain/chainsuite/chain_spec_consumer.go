package chainsuite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"
	providertypes "github.com/cosmos/interchain-security/v7/x/ccv/provider/types"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
)

// GetConsumerSpec returns the chain spec for the consumer chain (inveniam)
func GetConsumerSpec(ctx context.Context, env Environment, providerChain *Chain, proposalMsg *providertypes.MsgCreateConsumer) *interchaintest.ChainSpec {
	fullNodes := 0
	validators := 1
	coinDecimals := InterchaintestCoinDecimals

	return &interchaintest.ChainSpec{
		ChainName:     ConsumerChainID,
		NumFullNodes:  &fullNodes,
		NumValidators: &validators,
		ChainConfig: ibc.ChainConfig{
			ChainID:        ConsumerChainID,
			Name:           "inveniam",
			Bin:            ConsumerBin,
			Denom:          Uom,
			CoinDecimals:   &coinDecimals,
			Type:           "cosmos",
			GasPrices:      GasPrices,
			GasAdjustment:  2.0,
			TrustingPeriod: "336h",
			CoinType:       "118",
			Images: []ibc.DockerImage{
				{
					Repository: ConsumerImageName(env),
					Version:    ConsumerImageVersion(env),
					UIDGID:     "1025:1025",
				},
			},
			ConfigFileOverrides: map[string]any{
				"config/config.toml": DefaultConfigToml(),
			},
			// Expose EVM JSON-RPC port for tests
			ExposeAdditionalPorts: []string{"8545/tcp"},
			// Enable JSON-RPC server for EVM transactions and bind to 0.0.0.0 to allow external access
			AdditionalStartArgs: []string{"--json-rpc.enable", "--json-rpc.address=0.0.0.0:8545"},
			PreGenesis: func(consumer ibc.Chain) error {
				// Create consumer on provider chain
				consumerID, err := providerChain.CreateConsumer(ctx, proposalMsg, validatorMoniker)
				if err != nil {
					return err
				}

				// Opt-in all provider validators
				for index := 0; index < len(providerChain.Validators); index++ {
					if err := providerChain.OptIn(ctx, consumerID, index); err != nil {
						return err
					}
				}

				// Speed up chain launch by submitting update msg with current time as spawn time
				proposalMsg.InitializationParameters.SpawnTime = time.Now()
				updateMsg := &providertypes.MsgUpdateConsumer{
					ConsumerId:               consumerID,
					Owner:                    providerChain.ValidatorWallets[0].Address,
					NewOwnerAddress:          providerChain.ValidatorWallets[0].Address,
					InitializationParameters: proposalMsg.InitializationParameters,
					PowerShapingParameters:   proposalMsg.PowerShapingParameters,
				}
				if err := providerChain.UpdateConsumer(ctx, updateMsg, validatorMoniker); err != nil {
					return err
				}

				if err := testutil.WaitForBlocks(ctx, 2, providerChain); err != nil {
					return err
				}

				consumerChain, err := providerChain.GetConsumerChainByChainId(ctx, proposalMsg.ChainId)
				if err != nil {
					return err
				}

				if consumerChain.Phase != providertypes.CONSUMER_PHASE_LAUNCHED.String() {
					return errors.New("consumer chain is not launched")
				}

				return nil
			},
			Bech32Prefix:         ConsumerBech32Prefix,
			ModifyGenesisAmounts: ConsumerGenesisAmounts(Uom),
			ModifyGenesis:        ConsumerGenesisModifier(),
			InterchainSecurityConfig: ibc.ICSConfig{
				ConsumerCopyProviderKey: func(int) bool { return true },
			},
		},
	}
}

// ConsumerModifiedGenesis returns the genesis modifications for the consumer chain
func ConsumerModifiedGenesis() []cosmos.GenesisKV {
	return []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.slashing.params.signed_blocks_window", strconv.Itoa(SlashingWindowConsumer)),
		cosmos.NewGenesisKV("consensus.params.block.max_gas", "50000000"),
		cosmos.NewGenesisKV("app_state.ccvconsumer.params.soft_opt_out_threshold", "0.0"),
		cosmos.NewGenesisKV("app_state.ccvconsumer.params.blocks_per_distribution_transmission", strconv.Itoa(BlocksPerDistribution)),
		cosmos.NewGenesisKV("app_state.ccvconsumer.params.reward_denoms", []string{Uom}),
		// EVM configuration
		{
			Key:   "app_state.evm.params.evm_denom",
			Value: Uom,
		},
		{
			Key:   "app_state.evm.params.extended_denom_options.extended_denom",
			Value: Uom,
		},
		// Bank denom metadata for EVM
		{
			Key: "app_state.bank.denom_metadata",
			Value: []map[string]interface{}{
				{
					"base":        Uom,
					"display":     HumanCoinUnit,
					"symbol":      strings.ToUpper(HumanCoinUnit),
					"description": "The native staking token of the inveniam network",
					"denom_units": []map[string]interface{}{
						{
							"denom":    Uom,
							"exponent": 0,
						},
						{
							"denom":    HumanCoinUnit,
							"exponent": 18,
						},
					},
				},
			},
		},
	}
}

// ConsumerGenesisModifier returns a genesis modifier function that applies
// the standard genesis KV modifications and adds pre-funded test accounts
func ConsumerGenesisModifier() func(ibc.ChainConfig, []byte) ([]byte, error) {
	return func(cfg ibc.ChainConfig, genesis []byte) ([]byte, error) {
		// First apply the standard GenesisKV modifications
		genesis, err := cosmos.ModifyGenesis(ConsumerModifiedGenesis())(cfg, genesis)
		if err != nil {
			return nil, fmt.Errorf("failed to apply genesis KV modifications: %w", err)
		}

		// Parse genesis to add test account balance
		var genesisMap map[string]interface{}
		if err := json.Unmarshal(genesis, &genesisMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis: %w", err)
		}

		// Get app_state.bank.balances array
		appState, ok := genesisMap["app_state"].(map[string]interface{})
		if !ok {
			return nil, errors.New("app_state not found in genesis")
		}

		bankState, ok := appState["bank"].(map[string]interface{})
		if !ok {
			return nil, errors.New("bank state not found in genesis")
		}

		balances, ok := bankState["balances"].([]interface{})
		if !ok {
			// Initialize if not present
			balances = []interface{}{}
		}

		// Add pre-funded test accounts (both coin-type 118 and coin-type 60)
		// Coin-type 118 address (Cosmos standard) - used by interchaintest BuildWallet
		testAccountBalance := map[string]interface{}{
			"address": TestAccountAddress,
			"coins": []map[string]interface{}{
				{
					"denom":  Uom,
					"amount": sdkmath.NewInt(TestAccountFunds).String(),
				},
			},
		}
		balances = append(balances, testAccountBalance)

		// Coin-type 60 address (EVM compatibility) - used for EVM transactions
		testAccountBalanceEVM := map[string]interface{}{
			"address": TestAccountAddressEVM,
			"coins": []map[string]interface{}{
				{
					"denom":  Uom,
					"amount": sdkmath.NewInt(TestAccountFunds).String(),
				},
			},
		}
		balances = append(balances, testAccountBalanceEVM)
		bankState["balances"] = balances

		// Update supply to include both test account funds (2x TestAccountFunds)
		supply, ok := bankState["supply"].([]interface{})
		if ok && len(supply) > 0 {
			// Find the native denom in supply and add both test account funds
			for i, s := range supply {
				coin, ok := s.(map[string]interface{})
				if !ok {
					continue
				}
				if coin["denom"] == Uom {
					currentAmount, ok := coin["amount"].(string)
					if ok {
						current, ok := sdkmath.NewIntFromString(currentAmount)
						if ok {
							// Add funds for both test accounts
							testFunds := sdkmath.NewInt(TestAccountFunds)
							newAmount := current.Add(testFunds).Add(testFunds)
							coin["amount"] = newAmount.String()
							supply[i] = coin
						}
					}
					break
				}
			}
			bankState["supply"] = supply
		}

		// Add test account to auth module's accounts array
		// This is required so the account exists on-chain (not just has balance)
		authState, ok := appState["auth"].(map[string]interface{})
		if !ok {
			return nil, errors.New("auth state not found in genesis")
		}

		accounts, ok := authState["accounts"].([]interface{})
		if !ok {
			accounts = []interface{}{}
		}

		// Create a BaseAccount for the test account (coin-type 118)
		testAuthAccount := map[string]interface{}{
			"@type":          "/cosmos.auth.v1beta1.BaseAccount",
			"address":        TestAccountAddress,
			"pub_key":        nil,
			"account_number": "0",
			"sequence":       "0",
		}
		accounts = append(accounts, testAuthAccount)

		// Create a BaseAccount for the EVM test account (coin-type 60)
		testAuthAccountEVM := map[string]interface{}{
			"@type":          "/cosmos.auth.v1beta1.BaseAccount",
			"address":        TestAccountAddressEVM,
			"pub_key":        nil,
			"account_number": "0",
			"sequence":       "0",
		}
		accounts = append(accounts, testAuthAccountEVM)
		authState["accounts"] = accounts

		// Marshal back to JSON
		modifiedGenesis, err := json.Marshal(genesisMap)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal modified genesis: %w", err)
		}

		return modifiedGenesis, nil
	}
}

// ConsumerImageName returns the full image name for the consumer
func ConsumerImageName(env Environment) string {
	if env.DockerRegistry == "" {
		return env.ConsumerImageName
	}
	return env.DockerRegistry + "/" + env.ConsumerImageName
}

// ConsumerImageVersion returns the image version for the consumer
func ConsumerImageVersion(env Environment) string {
	if env.ConsumerImageVersion == "" {
		return "latest"
	}
	return env.ConsumerImageVersion
}
