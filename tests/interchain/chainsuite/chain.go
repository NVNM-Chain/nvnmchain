package chainsuite

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/tidwall/gjson"
	"golang.org/x/sync/errgroup"

	sdkmath "cosmossdk.io/math"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	providertypes "github.com/cosmos/interchain-security/v7/x/ccv/provider/types"
)

// This moniker is hardcoded into interchaintest
const validatorMoniker = "validator"

type Chain struct {
	*cosmos.CosmosChain
	ValidatorWallets []ValidatorWallet
	RelayerWallet    ibc.Wallet
	TestWallets      []ibc.Wallet
}

type ValidatorWallet struct {
	Moniker        string
	Address        string
	ValoperAddress string
	ValConsAddress string
}

func chainFromCosmosChain(cosmos *cosmos.CosmosChain, relayerWallet ibc.Wallet) (*Chain, error) {
	c := &Chain{CosmosChain: cosmos}
	wallets, err := getValidatorWallets(context.Background(), c)
	if err != nil {
		return nil, err
	}
	c.ValidatorWallets = wallets
	c.RelayerWallet = relayerWallet
	return c, nil
}

// CreateChain creates a single new chain with the given version and returns the chain object.
func CreateChain(ctx context.Context, testName interchaintest.TestName, spec *interchaintest.ChainSpec) (*Chain, error) {
	cf := interchaintest.NewBuiltinChainFactory(
		GetLogger(ctx),
		[]*interchaintest.ChainSpec{spec},
	)

	chains, err := cf.Chains(testName.Name())
	if err != nil {
		return nil, err
	}
	cosmosChain := chains[0].(*cosmos.CosmosChain)
	relayerWallet, err := cosmosChain.BuildRelayerWallet(ctx, "relayer-"+cosmosChain.Config().ChainID)
	if err != nil {
		return nil, err
	}

	ic := interchaintest.NewInterchain().AddChain(cosmosChain, ibc.WalletAmount{
		Address: relayerWallet.FormattedAddress(),
		Denom:   cosmosChain.Config().Denom,
		Amount:  sdkmath.NewInt(ValidatorFunds),
	})

	dockerClient, dockerNetwork := GetDockerContext(ctx)

	if err := ic.Build(ctx, GetRelayerExecReporter(ctx), interchaintest.InterchainBuildOptions{
		Client:    dockerClient,
		NetworkID: dockerNetwork,
		TestName:  testName.Name(),
	}); err != nil {
		return nil, err
	}

	chain, err := chainFromCosmosChain(cosmosChain, relayerWallet)
	if err != nil {
		return nil, err
	}
	return chain, nil
}

func (c *Chain) GenerateTx(ctx context.Context, valIdx int, command ...string) (string, error) {
	command = append([]string{"tx"}, command...)
	command = append(command, "--generate-only", "--keyring-backend", "test", "--chain-id", c.Config().ChainID)
	command = c.Validators[valIdx].NodeCommand(command...)
	stdout, _, err := c.Validators[valIdx].Exec(ctx, command, nil)
	if err != nil {
		return "", err
	}
	return string(stdout), nil
}

func (c *Chain) WaitForProposalStatus(ctx context.Context, proposalID string, status govv1.ProposalStatus) error {
	propID, err := strconv.ParseInt(proposalID, 10, 64)
	if err != nil {
		return err
	}
	chainHeight, err := c.Height(ctx)
	if err != nil {
		return err
	}
	// At 4s per block, 75 blocks is about 5 minutes.
	maxHeight := chainHeight + 75
	_, err = cosmos.PollForProposalStatusV1(ctx, c.CosmosChain, chainHeight, maxHeight, uint64(propID), status)
	return err
}

func (c *Chain) PassProposal(ctx context.Context, proposalID string) error {
	propID, err := strconv.ParseInt(proposalID, 10, 64)
	if err != nil {
		return err
	}
	err = c.VoteOnProposalAllValidators(ctx, uint64(propID), cosmos.ProposalVoteYes)
	if err != nil {
		return err
	}
	return c.WaitForProposalStatus(ctx, proposalID, govv1.StatusPassed)
}

// VoteForProposal votes on a proposal with all validators
func (c *Chain) VoteForProposal(ctx context.Context, proposalID string, vote string) error {
	propID, err := strconv.ParseInt(proposalID, 10, 64)
	if err != nil {
		return err
	}
	err = c.VoteOnProposalAllValidators(ctx, uint64(propID), vote)
	if err != nil {
		return err
	}
	return nil
}

// SubmitAndVoteForProposal submits a proposal and votes on it
func (c *Chain) SubmitAndVoteForProposal(ctx context.Context, prop cosmos.TxProposalv1, vote string) (string, error) {
	propTx, err := c.SubmitProposal(ctx, ValidatorMoniker, prop)
	if err != nil {
		return "", err
	}

	if err := c.WaitForProposalStatus(ctx, propTx.ProposalID, govv1.StatusVotingPeriod); err != nil {
		return "", err
	}

	if err := c.VoteForProposal(ctx, propTx.ProposalID, vote); err != nil {
		return "", err
	}

	return propTx.ProposalID, nil
}

// ExecuteProposalMsg builds proposal message, submits, votes and waits for proposal expected status
func (c *Chain) ExecuteProposalMsg(ctx context.Context, proposalMsg cosmos.ProtoMessage, proposer string, title string, vote string, expectedStatus govv1.ProposalStatus, expedited bool) error {
	// Use the chain's native denom for governance deposits
	deposit := fmt.Sprintf("%d%s", GovMinDepositAmount, c.Config().Denom)
	proposal, err := c.BuildProposal([]cosmos.ProtoMessage{proposalMsg}, title, "summary", "", deposit, proposer, false)
	if err != nil {
		return err
	}

	// submit and vote for the proposal
	proposalId, err := c.SubmitAndVoteForProposal(ctx, proposal, vote)
	if err != nil {
		return err
	}

	if err = c.WaitForProposalStatus(ctx, proposalId, expectedStatus); err != nil {
		return err
	}

	return nil
}

func (c *Chain) ReplaceImagesAndRestart(ctx context.Context, version string) error {
	// bring down nodes to prepare for upgrade
	err := c.StopAllNodes(ctx)
	if err != nil {
		return err
	}

	// upgrade version on all nodes
	repository := c.GetNode().Image.Repository
	// Support for using the public v8.0.0-rc0 image from GitHub Container Registry
	if version == "local" {
		repository = "mantra-chain/mantrachain"
		version = "latest"
	} else if version == "v8.0.0-rc0" {
		repository = "ghcr.io/mantra-chain/mantrachain"
	}
	c.UpgradeVersion(ctx, c.GetNode().DockerClient, repository, version)

	// start all nodes back up.
	// validators reach consensus on first block after upgrade height
	// and block production resumes.
	err = c.StartAllNodes(ctx)
	if err != nil {
		return err
	}

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 60*time.Second)
	defer timeoutCancel()
	err = testutil.WaitForBlocks(timeoutCtx, 5, c)
	if err != nil {
		return fmt.Errorf("failed to wait for blocks after upgrade: %w", err)
	}

	// Flush "successfully migrated key info" messages
	for _, val := range c.Validators {
		_, _, err := val.ExecBin(ctx, "keys", "list", "--keyring-backend", "test")
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Chain) Upgrade(ctx context.Context, upgradeName, version string) error {
	height, err := c.Height(ctx)
	if err != nil {
		return err
	}

	haltHeight := height + UpgradeDelta

	// Use the chain's native denom for governance deposits
	deposit := fmt.Sprintf("%d%s", GovMinDepositAmount*5000, c.Config().Denom) // greater than min deposit
	proposal := cosmos.SoftwareUpgradeProposal{
		Deposit:     deposit,
		Title:       "Upgrade to " + upgradeName,
		Name:        upgradeName,
		Description: "Upgrade to " + upgradeName,
		Height:      haltHeight,
	}
	upgradeTx, err := c.UpgradeProposal(ctx, interchaintest.FaucetAccountKeyName, proposal)
	if err != nil {
		return err
	}
	if err := c.PassProposal(ctx, upgradeTx.ProposalID); err != nil {
		return err
	}

	height, err = c.Height(ctx)
	if err != nil {
		return err
	}

	// wait for the chain to halt. We're asking for blocks after the halt height, so we should time out.
	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, (time.Duration(haltHeight-height)+10)*CommitTimeout)
	defer timeoutCtxCancel()
	err = testutil.WaitForBlocks(timeoutCtx, int(haltHeight-height)+3, c)
	if err == nil {
		return fmt.Errorf("chain should not produce blocks after halt height")
	} else if timeoutCtx.Err() == nil {
		return fmt.Errorf("chain should not produce blocks after halt height")
	}

	height, err = c.Height(ctx)
	if err != nil {
		return err
	}

	// make sure that chain is halted; some chains may produce one more block after halt height
	if height-haltHeight > 1 {
		return fmt.Errorf("height %d is not within one block of halt height %d; chain isn't halted", height, haltHeight)
	}

	return c.ReplaceImagesAndRestart(ctx, version)
}

func (c *Chain) GetValidatorPower(ctx context.Context, hexaddr string) (int64, error) {
	var power int64
	err := CheckEndpoint(ctx, c.GetHostRPCAddress()+"/validators", func(b []byte) error {
		power = gjson.GetBytes(b, fmt.Sprintf("result.validators.#(address==\"%s\").voting_power", hexaddr)).Int()
		if power == 0 {
			return fmt.Errorf("validator %s power not found; validators are: %s", hexaddr, string(b))
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return power, nil
}

func (c *Chain) GetValidatorHex(ctx context.Context, val int) (string, error) {
	json, err := c.Validators[val].ReadFile(ctx, "config/priv_validator_key.json")
	if err != nil {
		return "", err
	}
	providerHex := gjson.GetBytes(json, "address").String()
	return providerHex, nil
}

// GetValidatorHexAddress returns the hex address of a validator (alias for GetValidatorHex)
func (c *Chain) GetValidatorHexAddress(ctx context.Context, valIdx int) (string, error) {
	return c.GetValidatorHex(ctx, valIdx)
}

// UpdateAndVerifyStakeChange updates the staking amount on the provider chain and verifies that the change is reflected on the consumer side
func (p *Chain) UpdateAndVerifyStakeChange(ctx context.Context, consumer *Chain, relayer *Relayer, amount, valIdx int) error {
	providerAddress := p.ValidatorWallets[valIdx]

	providerHex, err := p.GetValidatorHexAddress(ctx, valIdx)
	if err != nil {
		return err
	}
	consumerHex, err := consumer.GetValidatorHexAddress(ctx, valIdx)
	if err != nil {
		return err
	}

	providerPowerBefore, err := p.GetValidatorPower(ctx, providerHex)
	if err != nil {
		return err
	}

	// increase the stake for the given validator
	_, err = p.Validators[valIdx].ExecTx(ctx, providerAddress.Moniker,
		"staking", "delegate",
		providerAddress.ValoperAddress, fmt.Sprintf("%d%s", amount, p.Config().Denom),
		"--gas", "500000",
	)
	if err != nil {
		return err
	}

	// check that the validator power is updated on both, provider and consumer chains
	tCtx, tCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer tCancel()
	var retErr error
	for tCtx.Err() == nil {
		retErr = nil
		providerPower, err := p.GetValidatorPower(ctx, providerHex)
		if err != nil {
			return err
		}
		consumerPower, err := consumer.GetValidatorPower(ctx, consumerHex)
		if err != nil {
			return err
		}
		if providerPowerBefore >= providerPower {
			retErr = fmt.Errorf("provider power did not increase after delegation")
		} else if providerPower != consumerPower {
			retErr = fmt.Errorf("consumer power did not update after provider delegation")
		}
		if retErr == nil {
			break
		}
		time.Sleep(CommitTimeout)
	}
	return retErr
}

func getValidatorWallets(ctx context.Context, chain *Chain) ([]ValidatorWallet, error) {
	validatorCount := len(chain.Validators)
	wallets := make([]ValidatorWallet, validatorCount)
	lock := new(sync.Mutex)
	eg := new(errgroup.Group)
	for i := 0; i < validatorCount; i++ {
		i := i
		eg.Go(func() error {
			// This moniker is hardcoded into the chain's genesis process.
			moniker := validatorMoniker
			address, err := chain.Validators[i].KeyBech32(ctx, moniker, "acc")
			if err != nil {
				return err
			}
			valoperAddress, err := chain.Validators[i].KeyBech32(ctx, moniker, "val")
			if err != nil {
				return err
			}
			valCons, _, err := chain.Validators[i].ExecBin(ctx, "comet", "show-address")
			if err != nil {
				return err
			}
			lock.Lock()
			defer lock.Unlock()
			wallets[i] = ValidatorWallet{
				Moniker:        moniker,
				Address:        address,
				ValoperAddress: valoperAddress,
				ValConsAddress: strings.TrimSpace(string(valCons)),
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return wallets, nil
}

func (c *Chain) QueryJSON(ctx context.Context, jsonPath string, query ...string) (gjson.Result, error) {
	stdout, _, err := c.GetNode().ExecQuery(ctx, query...)
	if err != nil {
		return gjson.Result{}, err
	}
	retval := gjson.GetBytes(stdout, jsonPath)
	if !retval.Exists() {
		return gjson.Result{}, fmt.Errorf("json path %s not found in query result %s", jsonPath, stdout)
	}
	return retval, nil
}

// GetProposalID parses the proposal ID from the tx; necessary when the proposal type isn't accessible to interchaintest yet
func (c *Chain) GetProposalID(ctx context.Context, txhash string) (string, error) {
	stdout, _, err := c.GetNode().ExecQuery(ctx, "tx", txhash)
	if err != nil {
		return "", err
	}
	result := struct {
		Events []abcitypes.Event `json:"events"`
	}{}
	if err := json.Unmarshal(stdout, &result); err != nil {
		return "", err
	}
	for _, event := range result.Events {
		if event.Type == "submit_proposal" {
			for _, attr := range event.Attributes {
				if string(attr.Key) == "proposal_id" {
					return string(attr.Value), nil
				}
			}
		}
	}
	return "", fmt.Errorf("proposal ID not found in tx %s", txhash)
}

// Provider chain methods for ICS

// AddConsumerChain creates and adds a consumer chain to the provider
func (p *Chain) AddConsumerChain(ctx context.Context, relayer *Relayer, spec *interchaintest.ChainSpec) (*Chain, error) {
	dockerClient, dockerNetwork := GetDockerContext(ctx)

	cf := interchaintest.NewBuiltinChainFactory(
		GetLogger(ctx),
		[]*interchaintest.ChainSpec{spec},
	)

	chains, err := cf.Chains(p.GetNode().TestName)
	if err != nil {
		return nil, err
	}
	consumer := chains[0].(*cosmos.CosmosChain)

	// We can't use AddProviderConsumerLink here because the provider chain is already built
	p.Consumers = append(p.Consumers, consumer)
	consumer.Provider = p.CosmosChain
	relayerWallet, err := consumer.BuildRelayerWallet(ctx, "relayer-"+consumer.Config().ChainID)
	if err != nil {
		return nil, err
	}
	wallets := make([]ibc.Wallet, len(p.Validators)+1)
	wallets[0] = relayerWallet
	// Create wallets for the validators with the right moniker
	for i := 1; i <= len(p.Validators); i++ {
		wallets[i], err = consumer.BuildRelayerWallet(ctx, validatorMoniker)
		if err != nil {
			return nil, err
		}
	}
	walletAmounts := make([]ibc.WalletAmount, len(wallets))
	for i, wallet := range wallets {
		walletAmounts[i] = ibc.WalletAmount{
			Address: wallet.FormattedAddress(),
			Denom:   consumer.Config().Denom,
			Amount:  sdkmath.NewInt(ValidatorFunds),
		}
	}

	ic := interchaintest.NewInterchain().
		AddChain(consumer, walletAmounts...).
		AddRelayer(relayer, "relayer")

	if err := ic.Build(ctx, GetRelayerExecReporter(ctx), interchaintest.InterchainBuildOptions{
		Client:    dockerClient,
		NetworkID: dockerNetwork,
		TestName:  p.GetNode().TestName,
	}); err != nil {
		return nil, err
	}

	for i, val := range consumer.Validators {
		if err := val.RecoverKey(ctx, validatorMoniker, wallets[i+1].Mnemonic()); err != nil {
			return nil, err
		}
	}
	consumerChain, err := chainFromCosmosChain(consumer, relayerWallet)
	if err != nil {
		return nil, err
	}

	return consumerChain, nil
}

// CreateConsumer creates a consumer chain on the provider
func (c *Chain) CreateConsumer(ctx context.Context, msg *providertypes.MsgCreateConsumer, keyName string) (string, error) {
	content, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	jsonFile := "create-consumer.json"
	if err = c.GetNode().WriteFile(ctx, content, jsonFile); err != nil {
		return "", err
	}

	filePath := path.Join(c.GetNode().HomeDir(), jsonFile)
	txHash, err := c.GetNode().ExecTx(ctx, keyName, "provider", "create-consumer", filePath)
	if err != nil {
		return "", err
	}

	response, err := c.GetNode().TxHashToResponse(ctx, txHash)
	if err != nil {
		return "", err
	}

	consumerId, found := getEvtAttribute(response.Events, providertypes.EventTypeCreateConsumer, providertypes.AttributeConsumerId)
	if !found {
		return "", fmt.Errorf("consumer id is not found")
	}

	return consumerId, err
}

// UpdateConsumer updates a consumer chain on the provider
func (c *Chain) UpdateConsumer(ctx context.Context, msg *providertypes.MsgUpdateConsumer, ownerKeyName string) error {
	content, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	jsonFile := "update-consumer.json"
	if err = c.GetNode().WriteFile(ctx, content, jsonFile); err != nil {
		return err
	}

	filePath := path.Join(c.GetNode().HomeDir(), jsonFile)

	_, err = c.GetNode().ExecTx(ctx, ownerKeyName, "provider", "update-consumer", filePath)
	return err
}

// OptIn opts a validator into a consumer chain
func (c *Chain) OptIn(ctx context.Context, consumerID string, valIndex int) error {
	_, err := c.Validators[valIndex].ExecTx(ctx, validatorMoniker, "provider", "opt-in", consumerID)
	return err
}

// OptOut opts a validator out of a consumer chain
func (c *Chain) OptOut(ctx context.Context, consumerID string, valIndex int) error {
	_, err := c.Validators[valIndex].ExecTx(ctx, validatorMoniker, "provider", "opt-out", consumerID)
	return err
}

// ListConsumerChains returns the list of consumer chains
func (c *Chain) ListConsumerChains(ctx context.Context) (ListConsumerChainsResponse, error) {
	queryRes, _, err := c.GetNode().ExecQuery(ctx, "provider", "list-consumer-chains")
	if err != nil {
		return ListConsumerChainsResponse{}, err
	}

	var queryResponse ListConsumerChainsResponse
	err = json.Unmarshal(queryRes, &queryResponse)
	if err != nil {
		return ListConsumerChainsResponse{}, err
	}

	return queryResponse, nil
}

// GetConsumerChainByChainId returns consumer chain info by chain ID
func (c *Chain) GetConsumerChainByChainId(ctx context.Context, chainId string) (ConsumerChain, error) {
	chains, err := c.ListConsumerChains(ctx)
	if err != nil {
		return ConsumerChain{}, err
	}

	for _, chain := range chains.Chains {
		if chain.ChainID == chainId {
			return chain, nil
		}
	}
	return ConsumerChain{}, fmt.Errorf("chain not found")
}

// GetProviderInfo returns the provider info from a consumer chain
func (c *Chain) GetProviderInfo(ctx context.Context) (ProviderInfoResponse, error) {
	queryRes, _, err := c.GetNode().ExecQuery(ctx, "ccvconsumer", "provider-info")
	if err != nil {
		return ProviderInfoResponse{}, err
	}

	var queryResponse ProviderInfoResponse
	err = json.Unmarshal(queryRes, &queryResponse)
	if err != nil {
		return ProviderInfoResponse{}, err
	}

	return queryResponse, nil
}

// SetupTestWallets creates test wallets for a chain
// Note: Wallets are created but NOT funded - tests should fund them as needed
// This avoids gas cost issues with the limited faucet balance on consumer chains
func SetupTestWallets(ctx context.Context, cosmosChain *cosmos.CosmosChain, walletCount int) ([]ibc.Wallet, error) {
	wallets := make([]ibc.Wallet, walletCount)
	for i := 0; i < walletCount; i++ {
		keyName := "testAccount" + strconv.Itoa(i)
		wallet, err := cosmosChain.BuildWallet(ctx, keyName, "")
		if err != nil {
			return nil, err
		}
		wallets[i] = wallet
	}
	return wallets, nil
}

func getEvtAttribute(events []abcitypes.Event, evtType string, key string) (string, bool) {
	for _, evt := range events {
		if evt.GetType() == evtType {
			for _, attr := range evt.Attributes {
				if attr.Key == key {
					return attr.Value, true
				}
			}
		}
	}

	return "", false
}
