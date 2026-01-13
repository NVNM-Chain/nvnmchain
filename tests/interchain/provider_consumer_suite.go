package interchain_test

import (
	"context"
	"time"

	sdkmath "cosmossdk.io/math"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/stretchr/testify/suite"

	"github.com/MANTRA-Chain/inveniam/tests/interchain/chainsuite"
)

// ProviderConsumerSuite is the test suite for provider-consumer chain interactions
type ProviderConsumerSuite struct {
	suite.Suite
	Provider *chainsuite.Chain
	Consumer *chainsuite.Chain
	Relayer  *chainsuite.Relayer
	ctx      context.Context
}

func (s *ProviderConsumerSuite) SetupSuite() {
	ctx, err := chainsuite.NewSuiteContext(&s.Suite)
	s.Require().NoError(err)
	s.ctx = ctx

	env := chainsuite.GetEnvironment()

	// Create and start provider chain (mantrachain)
	s.Provider, err = chainsuite.CreateChain(s.GetContext(), s.T(), chainsuite.GetProviderSpec(env, 1, providerModifiedGenesis()))
	s.Require().NoError(err)

	// Setup hermes relayer
	relayer, err := chainsuite.NewRelayer(s.GetContext(), s.T())
	s.Require().NoError(err)
	s.Relayer = relayer

	// Create consumer chain (inveniam) via ICS
	spawnTime := time.Now().Add(time.Hour)
	initParams := consumerInitParamsTemplate(&spawnTime)
	initParams.InitialHeight = clienttypes.Height{
		RevisionNumber: clienttypes.ParseChainID(chainsuite.ConsumerChainID),
		RevisionHeight: 1,
	}
	proposalMsg := msgCreateConsumer(
		chainsuite.ConsumerChainID,
		initParams,
		powerShapingParamsTemplate(),
		nil,
		chainsuite.ProviderGovModuleAddress,
	)

	s.Consumer, err = s.Provider.AddConsumerChain(
		s.GetContext(),
		relayer,
		chainsuite.GetConsumerSpec(s.GetContext(), env, s.Provider, proposalMsg),
	)
	s.Require().NoError(err)

	// Setup relayer keys for both chains
	s.Require().NoError(relayer.SetupChainKeys(s.GetContext(), s.Provider))
	s.Require().NoError(relayer.SetupChainKeys(s.GetContext(), s.Consumer))
	s.Require().NoError(relayer.RestartRelayer(s.GetContext()))

	// Confirm that tx on consumer cannot be sent before consumer and provider are connected
	// Note: The specific error may vary - it could be "tx contains unsupported message types"
	// on some ICS implementations, or "account not found" if the validator account hasn't
	// received funds yet. The key assertion is that the transaction fails.
	err = s.Consumer.SendFunds(ctx, chainsuite.ValidatorMoniker, ibc.WalletAmount{
		Amount:  sdkmath.NewInt(1000),
		Denom:   s.Consumer.Config().Denom,
		Address: s.Consumer.RelayerWallet.FormattedAddress(),
	})
	s.Require().Error(err, "expected transaction to fail before ICS connection is established")

	// Connect consumer and provider via IBC
	s.Require().NoError(s.Relayer.ConnectProviderConsumer(s.GetContext(), s.Provider, s.Consumer))
	s.Require().NoError(relayer.RestartRelayer(s.GetContext()))

	// Verify stake change propagates from provider to consumer
	// Amount must be >= 1e18 (1 token in atto units) to register as voting power
	// With AttoPowerReduction (1e18), voting power = stake / 1e18
	s.Require().NoError(s.Provider.UpdateAndVerifyStakeChange(s.GetContext(), s.Consumer, s.Relayer, 1_000_000_000_000_000_000, 0))

	// Verify provider info on consumer
	providerInfo, err := s.Consumer.GetProviderInfo(s.GetContext())
	s.Require().NoError(err)
	s.Require().Equal("connection-0", providerInfo.Provider.ConnectionID)

	// Build test wallets for consumer after ICS connection is established
	testWallets, err := chainsuite.SetupTestWallets(ctx, s.Consumer.CosmosChain, chainsuite.TestWalletsNumber)
	s.Require().NoError(err)
	s.Consumer.TestWallets = testWallets
}

func (s *ProviderConsumerSuite) GetContext() context.Context {
	s.Require().NotNil(s.ctx, "Tried to GetContext before it was set. SetupSuite must run first")
	return s.ctx
}

// providerModifiedGenesis returns genesis modifications for the provider chain
func providerModifiedGenesis() []cosmos.GenesisKV {
	// Start with the base provider genesis including EVM denom metadata
	genesis := chainsuite.ProviderModifiedGenesis()

	// Add test-specific overrides
	genesis = append(genesis,
		cosmos.NewGenesisKV("app_state.staking.params.unbonding_time", (chainsuite.ProviderUnbondingTime*10000000).String()),
		cosmos.NewGenesisKV("app_state.slashing.params.slash_fraction_double_sign", chainsuite.SlashFractionDoubleSign),
		cosmos.NewGenesisKV("app_state.staking.params.max_validators", "1"),
	)

	return genesis
}
