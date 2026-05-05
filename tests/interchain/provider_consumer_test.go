package interchain_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/suite"

	"github.com/NVNM-Chain/nvnmchain/tests/interchain/chainsuite"
)

func TestProviderConsumerSuite(t *testing.T) {
	s := &ProviderConsumerSuite{}
	suite.Run(t, s)
}

// TestConsumerBankSend tests that bank send transactions work on the consumer chain
// after ICS connection is established using a pre-funded test account
func (s *ProviderConsumerSuite) TestConsumerBankSend() {
	ctx := s.GetContext()

	// Recover the pre-funded test account wallet from mnemonic
	testWallet, err := s.Consumer.BuildWallet(ctx, "prefundedTest", chainsuite.TestAccountMnemonic)
	s.Require().NoError(err)

	// Verify the wallet address matches our expected pre-funded address
	s.Require().Equal(chainsuite.TestAccountAddress, testWallet.FormattedAddress(),
		"Recovered wallet address should match pre-funded address")

	// Query balance of the pre-funded account
	balance, err := s.Consumer.GetBalance(ctx, testWallet.FormattedAddress(), s.Consumer.Config().Denom)
	s.Require().NoError(err)
	s.T().Logf("Pre-funded test account balance: %s %s", balance.String(), s.Consumer.Config().Denom)
	s.Require().True(balance.GT(sdkmath.ZeroInt()), "Pre-funded account should have balance")

	// Create a recipient wallet
	recipientWallet, err := s.Consumer.BuildWallet(ctx, "recipient", "")
	s.Require().NoError(err)

	// Send funds from pre-funded account to recipient
	sendAmount := sdkmath.NewInt(1_000_000_000_000_000_000) // 1 token
	err = s.Consumer.SendFunds(ctx, "prefundedTest", ibc.WalletAmount{
		Address: recipientWallet.FormattedAddress(),
		Denom:   s.Consumer.Config().Denom,
		Amount:  sendAmount,
	})
	s.Require().NoError(err)

	// Verify recipient received the funds
	recipientBalance, err := s.Consumer.GetBalance(ctx, recipientWallet.FormattedAddress(), s.Consumer.Config().Denom)
	s.Require().NoError(err)
	s.T().Logf("Recipient balance after transfer: %s %s", recipientBalance.String(), s.Consumer.Config().Denom)
	s.Require().True(recipientBalance.GTE(sendAmount), "Recipient should have received funds")
}

// TestConsumerEVMTransaction tests EVM transactions on the consumer chain
// This test sends an actual ETH transaction using the JSON-RPC endpoint similar to test_ccv.py
// It verifies: assert w3.eth.get_balance(receiver) - balance == amt
func (s *ProviderConsumerSuite) TestConsumerEVMTransaction() {
	ctx := s.GetContext()

	// Wait for a few blocks to ensure the chain and JSON-RPC server are fully started
	err := testutil.WaitForBlocks(ctx, 3, s.Consumer.CosmosChain)
	s.Require().NoError(err, "Failed to wait for blocks")

	// Get the JSON-RPC endpoint using GetHostAddress for the exposed 8545/tcp port
	validator := s.Consumer.CosmosChain.Validators[0]
	evmRPCEndpoint, err := validator.GetHostAddress(ctx, "8545/tcp")
	s.Require().NoError(err, "Failed to get EVM JSON-RPC endpoint")
	s.T().Logf("Connecting to EVM JSON-RPC endpoint: %s", evmRPCEndpoint)

	// Retry connection to JSON-RPC endpoint with backoff
	var client *ethclient.Client
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		client, err = ethclient.Dial(evmRPCEndpoint)
		if err == nil {
			break
		}
		s.T().Logf("Attempt %d/%d: Failed to connect to JSON-RPC endpoint: %v. Retrying...", i+1, maxRetries, err)
		time.Sleep(2 * time.Second)
	}
	s.Require().NoError(err, "Failed to connect to EVM JSON-RPC endpoint after retries")
	defer client.Close()

	// Derive the EVM private key from the test account mnemonic
	// The test account mnemonic is used for the pre-funded Cosmos account
	// We derive the Ethereum key from it using BIP44 path for Ethereum (m/44'/60'/0'/0/0)
	derivedPriv, err := hd.Secp256k1.Derive()(chainsuite.TestAccountMnemonic, "", hd.CreateHDPath(60, 0, 0).String())
	s.Require().NoError(err, "Failed to derive private key from mnemonic")

	privateKey := hd.Secp256k1.Generate()(derivedPriv)

	// Convert to ECDSA private key for Ethereum
	privKeyBytes := privateKey.Bytes()
	senderPrivateKey, err := crypto.ToECDSA(privKeyBytes)
	s.Require().NoError(err, "Failed to convert to ECDSA private key")

	// Get sender address from private key
	senderAddress := crypto.PubkeyToAddress(senderPrivateKey.PublicKey)
	s.T().Logf("Sender EVM address (from test mnemonic): %s", senderAddress.Hex())

	// Log the expected Cosmos address for debugging
	// We expect this to match TestAccountAddressEVM (coin-type 60)
	s.T().Logf("Expected Cosmos address (EVM funded): %s", chainsuite.TestAccountAddressEVM)

	// Check Cosmos bank balance for debugging
	cosmosBalance, err := s.Consumer.GetBalance(ctx, chainsuite.TestAccountAddressEVM, s.Consumer.Config().Denom)
	s.Require().NoError(err)
	s.T().Logf("Cosmos bank balance for EVM address: %s %s", cosmosBalance.String(), s.Consumer.Config().Denom)

	// Check sender has funds (from genesis pre-funding)
	// The EVM JSON-RPC should query the bank module balance for this address
	senderBalance, err := client.BalanceAt(ctx, senderAddress, nil)
	s.Require().NoError(err)
	s.T().Logf("Sender EVM balance (via JSON-RPC): %s wei", senderBalance.String())
	s.Require().True(senderBalance.Cmp(big.NewInt(0)) > 0, "Sender should have funds from genesis")

	// Generate a receiver address
	receiverPrivateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	receiverAddress := crypto.PubkeyToAddress(receiverPrivateKey.PublicKey)
	s.T().Logf("Receiver address: %s", receiverAddress.Hex())

	// Get receiver balance before transaction
	balanceBefore, err := client.BalanceAt(ctx, receiverAddress, nil)
	s.Require().NoError(err)
	s.T().Logf("Receiver balance before: %s wei", balanceBefore.String())

	// Get chain ID
	chainID, err := client.ChainID(ctx)
	s.Require().NoError(err)
	s.T().Logf("Chain ID: %s", chainID.String())

	// Get nonce for sender
	nonce, err := client.PendingNonceAt(ctx, senderAddress)
	s.Require().NoError(err)
	s.T().Logf("Sender nonce: %d", nonce)

	// Get suggested gas price
	gasPrice, err := client.SuggestGasPrice(ctx)
	s.Require().NoError(err)
	s.T().Logf("Gas price: %s wei", gasPrice.String())

	// Create transaction to send 1000 wei
	amountToSend := big.NewInt(1000)
	gasLimit := uint64(21000) // Standard ETH transfer gas limit

	tx := types.NewTransaction(nonce, receiverAddress, amountToSend, gasLimit, gasPrice, nil)

	// Sign the transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), senderPrivateKey)
	s.Require().NoError(err)
	s.T().Logf("Transaction signed. Hash: %s", signedTx.Hash().Hex())

	// Send the transaction
	err = client.SendTransaction(ctx, signedTx)
	s.Require().NoError(err, "Failed to send transaction")
	s.T().Logf("Transaction sent: %s", signedTx.Hash().Hex())

	// Wait for transaction to be mined
	s.T().Log("Waiting for transaction to be mined...")
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var receipt *types.Receipt
	for {
		receipt, err = client.TransactionReceipt(timeoutCtx, signedTx.Hash())
		if err == nil {
			break
		}
		if timeoutCtx.Err() != nil {
			s.Require().FailNow("Transaction was not mined within timeout")
		}
		time.Sleep(1 * time.Second)
	}

	s.Require().Equal(uint64(1), receipt.Status, "Transaction should succeed")
	s.T().Logf("Transaction mined in block %d", receipt.BlockNumber.Uint64())

	// Get receiver balance after transaction
	balanceAfter, err := client.BalanceAt(ctx, receiverAddress, nil)
	s.Require().NoError(err)
	s.T().Logf("Receiver balance after: %s wei", balanceAfter.String())

	// Verify the balance change matches the amount sent
	// This is the equivalent of: assert w3.eth.get_balance(receiver) - balance == amt
	balanceChange := new(big.Int).Sub(balanceAfter, balanceBefore)
	s.Require().Equal(amountToSend, balanceChange,
		"Receiver balance should increase by exactly the amount sent")
	s.T().Logf("Balance verification successful: %s wei received", balanceChange.String())

	// Verify sender balance decreased by amount + gas
	senderBalanceAfter, err := client.BalanceAt(ctx, senderAddress, nil)
	s.Require().NoError(err)

	gasCost := new(big.Int).Mul(gasPrice, big.NewInt(int64(receipt.GasUsed)))
	expectedDecrease := new(big.Int).Add(amountToSend, gasCost)
	actualDecrease := new(big.Int).Sub(senderBalance, senderBalanceAfter)

	s.Require().Equal(expectedDecrease, actualDecrease,
		"Sender balance should decrease by amount sent + gas cost")
	s.T().Logf("Sender balance verification successful: %s wei deducted (amount + gas)", actualDecrease.String())
}

// TestValidatorStakeChange tests that validator stake changes on provider propagate to consumer
func (s *ProviderConsumerSuite) TestValidatorStakeChange() {
	// Update validator stake on provider and verify it propagates to consumer
	// Amount must be >= 1e18 (1 token in atto units) to register as voting power
	s.Require().NoError(s.Provider.UpdateAndVerifyStakeChange(s.GetContext(), s.Consumer, s.Relayer, 1_000_000_000_000_000_000, 0))
}

// TestIBCTransferMalformedCallbackMemo verifies that IBC transfers with malformed
// src_callback metadata in the memo are rejected at send time on the consumer chain.
//
// Before the fix, the callbacks middleware's SendPacket was not wired into the
// ICS4Wrapper chain, so malformed memos were accepted at send time but then
// blocked ack/timeout processing, permanently locking funds.
//
// After the fix (transfer → callbacks → ratelimit → channel), the same validation
// that runs at ack/timeout also runs at send time, so bad memos fail fast.
func (s *ProviderConsumerSuite) TestIBCTransferMalformedCallbackMemo() {
	ctx := s.GetContext()

	const senderKey = "ibcTestSender"

	// Create a fresh wallet and fund it from the pre-funded genesis account.
	senderWallet, err := s.Consumer.BuildWallet(ctx, senderKey, "")
	s.Require().NoError(err)

	_, _ = s.Consumer.BuildWallet(ctx, "prefundedTest", chainsuite.TestAccountMnemonic)
	err = s.Consumer.SendFunds(ctx, "prefundedTest", ibc.WalletAmount{
		Address: senderWallet.FormattedAddress(),
		Denom:   s.Consumer.Config().Denom,
		Amount:  sdkmath.NewInt(10_000_000_000_000_000), // enough for gas + several IBC sends
	})
	s.Require().NoError(err)

	// validAddr is only used as a string value inside memo fields, not as a tx signer.
	validAddr := s.Consumer.ValidatorWallets[0].Address

	tests := []struct {
		name        string
		memo        string
		expectError bool
	}{
		{
			name:        "no memo",
			memo:        "",
			expectError: false,
		},
		{
			name:        "memo without src_callback key",
			memo:        `{"ibc_memo": "plain transfer"}`,
			expectError: false,
		},
		{
			name:        "empty callback address",
			memo:        `{"src_callback": {"address": ""}}`,
			expectError: true,
		},
		{
			name:        "gas_limit as JSON number instead of string",
			memo:        `{"src_callback": {"address": "` + validAddr + `", "gas_limit": 500000}}`,
			expectError: true,
		},
		{
			name:        "calldata is not valid hex",
			memo:        `{"src_callback": {"address": "` + validAddr + `", "calldata": "not_hex!"}}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		s.Run(tt.name, func() {
			err := chainsuite.SendIBCTransferWithMemo(ctx, s.Consumer, s.Provider, s.Relayer, senderKey, tt.memo)
			if tt.expectError {
				s.Require().Error(err, "expected IBC transfer with memo %q to be rejected at send time", tt.memo)
			} else {
				s.Require().NoError(err, "IBC transfer with memo %q should succeed", tt.memo)
			}
		})
	}
}

// TestProviderInfo tests that consumer can query provider info correctly
func (s *ProviderConsumerSuite) TestProviderInfo() {
	providerInfo, err := s.Consumer.GetProviderInfo(s.GetContext())
	s.Require().NoError(err)

	// Verify provider chain ID
	s.Require().Equal(chainsuite.ProviderChainID, providerInfo.Provider.ChainID)

	// Verify connection is established
	s.Require().NotEmpty(providerInfo.Provider.ConnectionID)
	s.Require().NotEmpty(providerInfo.Provider.ClientID)
}
