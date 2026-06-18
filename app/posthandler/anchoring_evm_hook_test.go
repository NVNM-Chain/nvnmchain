package posthandler_test

import (
	"math/big"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/NVNM-Chain/nvnmchain/app/posthandler"
	testkeeper "github.com/NVNM-Chain/nvnmchain/testutil/keeper"
	anchoringtypes "github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
)

func setupHook(t *testing.T) (sdk.Context, *mockBank, posthandler.AnchoringEVMRefundHook) {
	t.Helper()
	k, ctx, _ := testkeeper.AnchoringKeeper(t)
	params, err := k.Params.Get(ctx)
	require.NoError(t, err)
	params.AnchoringFee = sdk.NewCoin(capDenom, math.NewIntWithDecimal(1, 16)) // 1¢
	require.NoError(t, k.Params.Set(ctx, params))

	bk := &mockBank{}
	return ctx, bk, posthandler.NewAnchoringEVMRefundHook(k, bk, authtypes.FeeCollectorName)
}

// gasPriceForFee returns the per-gas price such that gasUsed × price == feeWei.
func gasPriceForFee(feeWei *big.Int, gasUsed uint64) *big.Int {
	return new(big.Int).Quo(feeWei, new(big.Int).SetUint64(gasUsed))
}

func TestAnchoringEVMRefundHook(t *testing.T) {
	precompile := anchoringtypes.AnchoringPrecompileAddress
	wrapper := common.HexToAddress("0x000000000000000000000000000000000000C0DE")
	sender := common.HexToAddress("0x000000000000000000000000000000000000BEEF")
	cap1c := math.NewIntWithDecimal(1, 16).BigInt()
	fee5c := math.NewIntWithDecimal(5, 16).BigInt()

	tests := []struct {
		name       string
		to         *common.Address
		gasUsed    uint64
		gasPrice   *big.Int
		receipt    *ethtypes.Receipt
		wantRefund sdk.Coins
	}{
		{
			name:       "direct precompile, overage refunded",
			to:         &precompile,
			gasUsed:    100_000,
			gasPrice:   gasPriceForFee(fee5c, 100_000),
			receipt:    &ethtypes.Receipt{Status: ethtypes.ReceiptStatusSuccessful, GasUsed: 100_000},
			wantRefund: sdk.NewCoins(sdk.NewCoin(capDenom, math.NewIntWithDecimal(4, 16))),
		},
		{
			name:     "natural below cap no refund",
			to:       &precompile,
			gasUsed:  50_000,
			gasPrice: gasPriceForFee(new(big.Int).Quo(cap1c, big.NewInt(2)), 50_000), // 0.5¢
			receipt:  &ethtypes.Receipt{Status: ethtypes.ReceiptStatusSuccessful, GasUsed: 50_000},
		},
		{
			name:     "contract-wrapped precompile skipped",
			to:       &wrapper,
			gasPrice: big.NewInt(1_000_000_000_000),
			receipt:  &ethtypes.Receipt{Status: ethtypes.ReceiptStatusSuccessful, GasUsed: 150_000},
		},
		{
			name:     "contract creation skipped",
			to:       nil,
			gasPrice: big.NewInt(1_000_000_000_000),
			receipt:  &ethtypes.Receipt{Status: ethtypes.ReceiptStatusSuccessful, GasUsed: 150_000},
		},
		{
			name:     "failed receipt no refund",
			to:       &precompile,
			gasPrice: big.NewInt(1_000_000_000_000),
			receipt:  &ethtypes.Receipt{Status: ethtypes.ReceiptStatusFailed, GasUsed: 150_000},
		},
		{
			name:     "nil receipt no refund",
			to:       &precompile,
			gasPrice: big.NewInt(1_000_000_000_000),
			receipt:  nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, bk, hook := setupHook(t)
			msg := core.Message{To: tc.to, From: sender, GasPrice: tc.gasPrice}
			require.NoError(t, hook.PostTxProcessing(ctx, sender, msg, tc.receipt))

			if len(tc.wantRefund) == 0 {
				require.Empty(t, bk.calls)
				return
			}
			require.Len(t, bk.calls, 1)
			require.Equal(t, tc.wantRefund, bk.calls[0].coins)
		})
	}
}
