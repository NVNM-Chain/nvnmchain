package keeper_test

import (
	"testing"

	appparams "github.com/MANTRA-Chain/inveniam/app/params"
	keepertest "github.com/MANTRA-Chain/inveniam/testutil/keeper"
	"github.com/stretchr/testify/require"
)

func TestCreateDenom(t *testing.T) {
	appparams.SetAddressPrefixes()

	testcases := []struct {
		address        string
		subdenom       string
		expectError    bool
		expectNewDenom string
	}{
		{
			address:        "inveniam13jq8lj4l2hjt9zs2wjpwzvfftmxtuzx7823eag",
			subdenom:       "test",
			expectError:    false,
			expectNewDenom: "",
		},
	}

	for _, tc := range testcases {
		bk := keepertest.MockBankKeeper{
			HasDenomSupply: false,
			DenomExists:    false,
		}
		ek := keepertest.MockERC20Keeper{
			IsRegistered: false,
		}
		k, ctx := keepertest.TokenFactoryKeeper(t, bk, ek)
		newTokenDenom, err := k.CreateDenom(ctx, tc.address, tc.subdenom)
		if tc.expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.NotEmpty(t, newTokenDenom)
		}
	}
}

func TestUpdateDenomWithERC20(t *testing.T) {
	appparams.SetAddressPrefixes()

	testcases := []struct {
		name                string
		denom               string
		denomExists         bool
		erc20AlreadyReg     bool
		expectError         bool
		expectedErrorString string
	}{
		{
			name:            "successful update of existing denom",
			denom:           "factory/inveniam13jq8lj4l2hjt9zs2wjpwzvfftmxtuzx7823eag/test",
			denomExists:     true,
			erc20AlreadyReg: false,
			expectError:     false,
		},
		{
			name:                "fail when denom does not exist",
			denom:               "factory/inveniam13jq8lj4l2hjt9zs2wjpwzvfftmxtuzx7823eag/nonexistent",
			denomExists:         false,
			erc20AlreadyReg:     false,
			expectError:         true,
			expectedErrorString: "denom does not exist",
		},
		{
			name:            "fail when ERC20 is already registered",
			denom:           "factory/inveniam13jq8lj4l2hjt9zs2wjpwzvfftmxtuzx7823eag/duplicate",
			denomExists:     true,
			erc20AlreadyReg: true,
			expectError:     false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			bk := keepertest.MockBankKeeper{
				HasDenomSupply: false,
				DenomExists:    tc.denomExists,
			}
			ek := keepertest.MockERC20Keeper{
				IsRegistered: tc.erc20AlreadyReg,
			}
			k, ctx := keepertest.TokenFactoryKeeper(t, bk, ek)

			err := k.UpdateDenomWithERC20(ctx, tc.denom)

			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErrorString)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
