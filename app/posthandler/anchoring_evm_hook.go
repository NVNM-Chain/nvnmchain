package posthandler

import (
	"context"
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	anchoringkeeper "github.com/NVNM-Chain/nvnmchain/x/anchoring/keeper"
	anchoringtypes "github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
)

// AnchoringEVMRefundHook caps direct EVM precompile calls at
// Params.AnchoringFee, refunding the overage. EvmHook is used (not the SDK
// PostHandler) because baseapp gives PostHandler a fresh EventManager —
// receipt-derived data is unreachable from there.
type AnchoringEVMRefundHook struct {
	anchoringKeeper anchoringkeeper.Keeper
	bankKeeper      BankKeeper
	feeCollector    string
}

func NewAnchoringEVMRefundHook(
	ak anchoringkeeper.Keeper,
	bk BankKeeper,
	feeCollector string,
) AnchoringEVMRefundHook {
	return AnchoringEVMRefundHook{
		anchoringKeeper: ak,
		bankKeeper:      bk,
		feeCollector:    feeCollector,
	}
}

// PostTxProcessing implements evmtypes.EvmHooks. msg.GasPrice is the
// effectiveGasPrice (cosmos/evm sets it via AsMessage), so net fee is
// receipt.GasUsed × msg.GasPrice.
func (h AnchoringEVMRefundHook) PostTxProcessing(
	ctx sdk.Context,
	sender common.Address,
	msg core.Message,
	receipt *ethtypes.Receipt,
) error {
	if receipt == nil || receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return nil
	}
	if msg.To == nil || *msg.To != anchoringtypes.AnchoringPrecompileAddress {
		return nil
	}
	if msg.GasPrice == nil || msg.GasPrice.Sign() <= 0 {
		return nil
	}

	netPaid := new(big.Int).Mul(new(big.Int).SetUint64(receipt.GasUsed), msg.GasPrice)

	params, err := h.anchoringKeeper.Params.Get(ctx)
	if err != nil {
		return fmt.Errorf("get anchoring params: %w", err)
	}
	if !params.IsAnchoringFeeUsable() {
		return nil
	}
	cap := params.AnchoringFee
	capAmt := cap.Amount.BigInt()
	if netPaid.Cmp(capAmt) <= 0 {
		return nil
	}

	overage := math.NewIntFromBigInt(new(big.Int).Sub(netPaid, capAmt))
	refund := sdk.NewCoin(cap.Denom, overage)
	payer := sdk.AccAddress(sender.Bytes())

	if err := h.bankKeeper.SendCoinsFromModuleToAccount(
		context.Context(ctx),
		h.feeCollector,
		payer,
		sdk.NewCoins(refund),
	); err != nil {
		return fmt.Errorf("refund anchoring fee overage: %w", err)
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		anchoringtypes.EventTypeAnchoringFeeRefunded,
		sdk.NewAttribute(anchoringtypes.AttributeKeyFeePayer, payer.String()),
		sdk.NewAttribute(anchoringtypes.AttributeKeyRefundAmount, refund.String()),
	))
	return nil
}
