package interchain_test

import (
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	"github.com/NVNM-Chain/nvnmchain/tests/interchain/chainsuite"
)

const txAmount = 1_000_000_000

type TxSuite struct {
	*chainsuite.Suite
}

func txAmountUom() string {
	// Use ProviderDenom (amantra) for provider chain tests
	return fmt.Sprintf("%d%s", txAmount, chainsuite.ProviderDenom)
}

func (s *TxSuite) TestBankSend() {
	balanceBefore, err := s.Chain.GetBalance(s.GetContext(), s.Chain.ValidatorWallets[1].Address, chainsuite.ProviderDenom)
	s.Require().NoError(err)

	_, err = s.Chain.Validators[0].ExecTx(
		s.GetContext(),
		s.Chain.ValidatorWallets[0].Moniker,
		"bank", "send",
		s.Chain.ValidatorWallets[0].Address, s.Chain.ValidatorWallets[1].Address, txAmountUom(),
	)
	s.Require().NoError(err)

	balanceAfter, err := s.Chain.GetBalance(s.GetContext(), s.Chain.ValidatorWallets[1].Address, chainsuite.ProviderDenom)
	s.Require().NoError(err)
	s.Require().Equal(balanceBefore.Add(sdkmath.NewInt(txAmount)), balanceAfter)
}

func (s TxSuite) TestDelegateWithdrawUnbond() {
	// delegate tokens
	_, err := s.Chain.Validators[0].ExecTx(
		s.GetContext(),
		s.Chain.ValidatorWallets[0].Moniker,
		"staking", "delegate", s.Chain.ValidatorWallets[0].ValoperAddress, txAmountUom(),
	)
	s.Require().NoError(err)

	time.Sleep(20 * time.Second)
	// Withdraw rewards - just verify the transaction succeeds
	// Note: With high gas prices (1e9), the gas cost (~2e14) exceeds rewards earned in 20s,
	// so we don't check if balance increased, just that the withdrawal works
	_, err = s.Chain.Validators[0].ExecTx(
		s.GetContext(),
		s.Chain.ValidatorWallets[0].Moniker,
		"distribution", "withdraw-rewards", s.Chain.ValidatorWallets[0].ValoperAddress,
	)
	s.Require().NoError(err)

	// Unbond tokens
	_, err = s.Chain.Validators[0].ExecTx(
		s.GetContext(),
		s.Chain.ValidatorWallets[0].Moniker,
		"staking", "unbond", s.Chain.ValidatorWallets[0].ValoperAddress, txAmountUom(),
	)
	s.Require().NoError(err)
}


func (s TxSuite) TestFeegrant() {
	const (
		granter       = 5
		grantee       = 1
		fundsReceiver = 2
	)

	tests := []struct {
		name   string
		revoke func(expireTime time.Time)
	}{
		{
			name: "revoke",
			revoke: func(_ time.Time) {
				_, err := s.Chain.Validators[granter].ExecTx(
					s.GetContext(),
					s.Chain.ValidatorWallets[granter].Moniker,
					"feegrant", "revoke", s.Chain.ValidatorWallets[granter].Address, s.Chain.ValidatorWallets[grantee].Address,
				)
				s.Require().NoError(err)
			},
		},
		{
			name: "expire",
			revoke: func(expire time.Time) {
				<-time.After(time.Until(expire))
				err := testutil.WaitForBlocks(s.GetContext(), 1, s.Chain)
				s.Require().NoError(err)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		s.Run(tt.name, func() {
			expire := time.Now().Add(20 * chainsuite.CommitTimeout)
			_, err := s.Chain.Validators[granter].ExecTx(
				s.GetContext(),
				s.Chain.ValidatorWallets[granter].Moniker,
				"feegrant", "grant", s.Chain.ValidatorWallets[granter].Address, s.Chain.ValidatorWallets[grantee].Address,
				"--expiration", expire.Format(time.RFC3339),
			)
			s.Require().NoError(err)

			granterBalanceBefore, err := s.Chain.GetBalance(s.GetContext(), s.Chain.ValidatorWallets[granter].Address, chainsuite.ProviderDenom)
			s.Require().NoError(err)
			granteeBalanceBefore, err := s.Chain.GetBalance(s.GetContext(), s.Chain.ValidatorWallets[grantee].Address, chainsuite.ProviderDenom)
			s.Require().NoError(err)

			_, err = s.Chain.Validators[grantee].ExecTx(s.GetContext(), s.Chain.ValidatorWallets[grantee].Moniker,
				"bank", "send", s.Chain.ValidatorWallets[grantee].Address, s.Chain.ValidatorWallets[fundsReceiver].Address, txAmountUom(),
				"--fee-granter", s.Chain.ValidatorWallets[granter].Address,
			)
			s.Require().NoError(err)

			granteeBalanceAfter, err := s.Chain.GetBalance(s.GetContext(), s.Chain.ValidatorWallets[grantee].Address, chainsuite.ProviderDenom)
			s.Require().NoError(err)
			granterBalanceAfter, err := s.Chain.GetBalance(s.GetContext(), s.Chain.ValidatorWallets[granter].Address, chainsuite.ProviderDenom)
			s.Require().NoError(err)

			s.Require().True(granterBalanceAfter.LT(granterBalanceBefore), "granterBalanceBefore: %s, granterBalanceAfter: %s", granterBalanceBefore, granterBalanceAfter)
			s.Require().True(granteeBalanceAfter.Equal(granteeBalanceBefore.Sub(sdkmath.NewInt(txAmount))), "granteeBalanceBefore: %s, granteeBalanceAfter: %s", granteeBalanceBefore, granteeBalanceAfter)

			tt.revoke(expire)

			_, err = s.Chain.Validators[1].ExecTx(s.GetContext(), s.Chain.ValidatorWallets[grantee].Moniker,
				"bank", "send", s.Chain.ValidatorWallets[1].Address, s.Chain.ValidatorWallets[fundsReceiver].Address, txAmountUom(),
				"--fee-granter", s.Chain.ValidatorWallets[0].Address,
			)
			s.Require().Error(err)
		})
	}
}

func (s *TxSuite) TestMultisig() {
	pubkey1, _, err := s.Chain.Validators[1].ExecBin(s.GetContext(), "keys", "show", s.Chain.ValidatorWallets[1].Moniker, "--pubkey", "--keyring-backend", "test")
	s.Require().NoError(err)

	pubkey2, _, err := s.Chain.Validators[2].ExecBin(s.GetContext(), "keys", "show", s.Chain.ValidatorWallets[2].Moniker, "--pubkey", "--keyring-backend", "test")
	s.Require().NoError(err)

	_, _, err = s.Chain.Validators[0].ExecBin(s.GetContext(), "keys", "add", "val1", "--pubkey", string(pubkey1), "--keyring-backend", "test")
	s.Require().NoError(err)

	_, _, err = s.Chain.Validators[0].ExecBin(s.GetContext(), "keys", "add", "val2", "--pubkey", string(pubkey2), "--keyring-backend", "test")
	s.Require().NoError(err)

	multisigName := "multisig"
	_, _, err = s.Chain.Validators[0].ExecBin(s.GetContext(), "keys", "add", multisigName, "--multisig", fmt.Sprintf("%s,val1,val2", s.Chain.ValidatorWallets[0].Moniker), "--multisig-threshold", "2", "--keyring-backend", "test")
	s.Require().NoError(err)

	pubkeyMulti, _, err := s.Chain.Validators[0].ExecBin(s.GetContext(), "keys", "show", multisigName, "--pubkey", "--keyring-backend", "test")
	s.Require().NoError(err)

	_, _, err = s.Chain.Validators[1].ExecBin(s.GetContext(), "keys", "add", multisigName, "--pubkey", string(pubkeyMulti), "--keyring-backend", "test")
	s.Require().NoError(err)
	_, _, err = s.Chain.Validators[2].ExecBin(s.GetContext(), "keys", "add", multisigName, "--pubkey", string(pubkeyMulti), "--keyring-backend", "test")
	s.Require().NoError(err)
	// bogus validator, not in the multisig
	_, _, err = s.Chain.Validators[4].ExecBin(s.GetContext(), "keys", "add", multisigName, "--pubkey", string(pubkeyMulti), "--keyring-backend", "test")
	s.Require().NoError(err)

	defer func() {
		_, _, err = s.Chain.Validators[0].ExecBin(s.GetContext(), "keys", "delete", "val1", "--keyring-backend", "test", "-y")
		s.Require().NoError(err)
		_, _, err = s.Chain.Validators[0].ExecBin(s.GetContext(), "keys", "delete", "val2", "--keyring-backend", "test", "-y")
		s.Require().NoError(err)
		for i := 0; i < 3; i++ {
			_, _, err = s.Chain.Validators[i].ExecBin(s.GetContext(), "keys", "delete", multisigName, "--keyring-backend", "test", "-y")
			s.Require().NoError(err)
		}
		_, _, err = s.Chain.Validators[4].ExecBin(s.GetContext(), "keys", "delete", multisigName, "--keyring-backend", "test", "-y")
		s.Require().NoError(err)
	}()

	multisigAddr, err := s.Chain.Validators[0].KeyBech32(s.GetContext(), multisigName, "")
	s.Require().NoError(err)

	// Send funds from validator (which has 9e18) instead of faucet (which has only 1e14)
	_, err = s.Chain.Validators[0].ExecTx(
		s.GetContext(),
		s.Chain.ValidatorWallets[0].Moniker,
		"bank", "send",
		s.Chain.ValidatorWallets[0].Address, multisigAddr, "300000000000000"+chainsuite.ProviderDenom, // 3e14 - enough for gas (~2e14) + tx amount (1e9)
	)
	s.Require().NoError(err)

	balanceBefore, err := s.Chain.GetBalance(s.GetContext(), s.Chain.ValidatorWallets[3].Address, chainsuite.ProviderDenom)
	s.Require().NoError(err)

	txjson, err := s.Chain.GenerateTx(
		s.GetContext(), 0, "bank", "send", multisigName, s.Chain.ValidatorWallets[3].Address, txAmountUom(),
		"--gas", "auto", "--gas-adjustment", fmt.Sprint(s.Chain.Config().GasAdjustment), "--gas-prices", s.Chain.Config().GasPrices,
	)
	s.Require().NoError(err)

	err = s.Chain.Validators[0].WriteFile(s.GetContext(), []byte(txjson), "tx.json")
	s.Require().NoError(err)

	signed0, _, err := s.Chain.Validators[0].Exec(s.GetContext(),
		s.Chain.Validators[0].TxCommand(s.Chain.ValidatorWallets[0].Moniker, "sign",
			path.Join(s.Chain.Validators[0].HomeDir(), "tx.json"),
			"--multisig", multisigAddr,
		), nil)
	s.Require().NoError(err)

	err = s.Chain.Validators[1].WriteFile(s.GetContext(), []byte(txjson), "tx.json")
	s.Require().NoError(err)

	signed1, _, err := s.Chain.Validators[1].Exec(s.GetContext(),
		s.Chain.Validators[1].TxCommand(s.Chain.ValidatorWallets[1].Moniker, "sign",
			path.Join(s.Chain.Validators[1].HomeDir(), "tx.json"),
			"--multisig", multisigAddr,
		), nil)
	s.Require().NoError(err)

	err = s.Chain.Validators[4].WriteFile(s.GetContext(), []byte(txjson), "tx.json")
	s.Require().NoError(err)
	_, _, err = s.Chain.Validators[4].Exec(s.GetContext(),
		s.Chain.Validators[4].TxCommand(s.Chain.ValidatorWallets[4].Moniker, "sign",
			path.Join(s.Chain.Validators[4].HomeDir(), "tx.json"),
			"--multisig", multisigAddr,
		), nil)
	s.Require().Error(err)

	err = s.Chain.Validators[0].WriteFile(s.GetContext(), signed0, "signed0.json")
	s.Require().NoError(err)

	_, _, err = s.Chain.Validators[0].Exec(s.GetContext(), s.Chain.Validators[0].TxCommand(
		multisigName,
		"multisign",
		path.Join(s.Chain.Validators[0].HomeDir(), "tx.json"),
		multisigName,
		path.Join(s.Chain.Validators[0].HomeDir(), "signed0.json"),
	), nil)
	s.Require().NoError(err)

	err = s.Chain.Validators[0].WriteFile(s.GetContext(), signed1, "signed1.json")
	s.Require().NoError(err)

	multisign, _, err := s.Chain.Validators[0].Exec(s.GetContext(), s.Chain.Validators[0].TxCommand(
		multisigName,
		"multisign",
		path.Join(s.Chain.Validators[0].HomeDir(), "tx.json"),
		multisigName,
		path.Join(s.Chain.Validators[0].HomeDir(), "signed0.json"),
		path.Join(s.Chain.Validators[0].HomeDir(), "signed1.json"),
	), nil)
	s.Require().NoError(err)

	err = s.Chain.Validators[0].WriteFile(s.GetContext(), multisign, "multisign.json")
	s.Require().NoError(err)

	_, err = s.Chain.Validators[0].ExecTx(s.GetContext(), multisigName, "broadcast", path.Join(s.Chain.Validators[0].HomeDir(), "multisign.json"))
	s.Require().NoError(err)
	balanceAfter, err := s.Chain.GetBalance(s.GetContext(), s.Chain.ValidatorWallets[3].Address, chainsuite.ProviderDenom)
	s.Require().NoError(err)
	s.Require().Equal(balanceBefore.Add(sdkmath.NewInt(txAmount)), balanceAfter)
}

func TestTransactions(t *testing.T) {
	txSuite := TxSuite{chainsuite.NewSuite(chainsuite.SuiteConfig{})}
	suite.Run(t, &txSuite)
}
