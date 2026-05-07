package e2e

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *IntegrationTestSuite) testBankTokenTransfer() {
	s.Run("send_tokens_between_accounts", func() {
		var (
			err           error
			valIdx        = 0
			c             = s.chainA
			chainEndpoint = fmt.Sprintf("http://%s", s.valResources[c.id][valIdx].GetHostPort("1317/tcp"))
		)

		// define one sender and two recipient accounts
		alice, _ := c.genesisAccounts[1].keyInfo.GetAddress()
		bob, _ := c.genesisAccounts[2].keyInfo.GetAddress()
		charlie, _ := c.genesisAccounts[3].keyInfo.GetAddress()

		var beforeAliceUomBalance,
			beforeBobUomBalance,
			beforeCharlieUomBalance,
			afterAliceUomBalance,
			afterBobUomBalance,
			afterCharlieUomBalance sdk.Coin

		// get balances of sender and recipient accounts
		s.Require().Eventually(
			func() bool {
				beforeAliceUomBalance, err = getSpecificBalance(chainEndpoint, alice.String(), anvnmDenom)
				s.Require().NoError(err)

				beforeBobUomBalance, err = getSpecificBalance(chainEndpoint, bob.String(), anvnmDenom)
				s.Require().NoError(err)

				beforeCharlieUomBalance, err = getSpecificBalance(chainEndpoint, charlie.String(), anvnmDenom)
				s.Require().NoError(err)

				return beforeAliceUomBalance.IsValid() && beforeBobUomBalance.IsValid() && beforeCharlieUomBalance.IsValid()
			},
			10*time.Second,
			5*time.Second,
		)

		// alice sends tokens to bob
		s.execBankSend(s.chainA, valIdx, alice.String(), bob.String(), tokenAmount.String(), standardFees.String(), false)

		// check that the transfer was successful
		s.Require().Eventually(
			func() bool {
				afterAliceUomBalance, err = getSpecificBalance(chainEndpoint, alice.String(), anvnmDenom)
				s.Require().NoError(err)

				afterBobUomBalance, err = getSpecificBalance(chainEndpoint, bob.String(), anvnmDenom)
				s.Require().NoError(err)

				expectedAliceUomBalance := beforeAliceUomBalance.Sub(tokenAmount).Sub(standardFees)
				expectedBobUomBalance := beforeBobUomBalance.Add(tokenAmount)
				decremented := expectedAliceUomBalance.Equal(afterAliceUomBalance)
				incremented := expectedBobUomBalance.Equal(afterBobUomBalance)

				return decremented && incremented
			},
			10*time.Second,
			5*time.Second,
		)

		// save the updated account balances of alice and bob
		beforeAliceUomBalance, beforeBobUomBalance = afterAliceUomBalance, afterBobUomBalance

		// alice sends tokens to bob and charlie, at once
		s.execBankMultiSend(s.chainA, valIdx, alice.String(), []string{bob.String(), charlie.String()}, tokenAmount.String(), standardFees.String(), false)

		s.Require().Eventually(
			func() bool {
				afterAliceUomBalance, err = getSpecificBalance(chainEndpoint, alice.String(), anvnmDenom)
				s.Require().NoError(err)

				afterBobUomBalance, err = getSpecificBalance(chainEndpoint, bob.String(), anvnmDenom)
				s.Require().NoError(err)

				afterCharlieUomBalance, err = getSpecificBalance(chainEndpoint, charlie.String(), anvnmDenom)
				s.Require().NoError(err)

				expectedAliceUomBalance := beforeAliceUomBalance.Sub(tokenAmount).Sub(tokenAmount).Sub(standardFees)
				expectedBobUomBalance := beforeBobUomBalance.Add(tokenAmount)
				expectedCharlieUomBalance := beforeCharlieUomBalance.Add(tokenAmount)
				decremented := expectedAliceUomBalance.Equal(afterAliceUomBalance)
				incremented := expectedBobUomBalance.Equal(afterBobUomBalance) &&
					expectedCharlieUomBalance.Equal(afterCharlieUomBalance)

				return decremented && incremented
			},
			10*time.Second,
			5*time.Second,
		)
	})
}
