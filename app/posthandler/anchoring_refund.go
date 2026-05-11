package posthandler

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	anchoringkeeper "github.com/NVNM-Chain/nvnmchain/x/anchoring/keeper"
	anchoringtypes "github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
)

// BankKeeper is the narrow bank slice used here so unit tests can mock it.
type BankKeeper interface {
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// AnchoringRefundDecorator caps single-msg MsgAddRecord fees at
// Params.AnchoringFee, refunding the overage on success. EVM direct
// precompile calls are handled separately in AnchoringEVMRefundHook.
type AnchoringRefundDecorator struct {
	anchoringKeeper anchoringkeeper.Keeper
	bankKeeper      BankKeeper
	feeCollector    string
}

func NewAnchoringRefundDecorator(
	ak anchoringkeeper.Keeper,
	bk BankKeeper,
	feeCollector string,
) AnchoringRefundDecorator {
	return AnchoringRefundDecorator{
		anchoringKeeper: ak,
		bankKeeper:      bk,
		feeCollector:    feeCollector,
	}
}

func (d AnchoringRefundDecorator) PostHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	success bool,
	next sdk.PostHandler,
) (sdk.Context, error) {
	if simulate || !success {
		return next(ctx, tx, simulate, success)
	}
	if anchoringtypes.IsCosmosAnchoringTx(tx) {
		if err := d.refundCosmos(ctx, tx); err != nil {
			return ctx, err
		}
	}
	return next(ctx, tx, simulate, success)
}

// FeePayer falls back to the first signer when AuthInfo.Fee.Payer is empty
// (see cosmos-sdk x/auth/tx/builder.go FeePayer); empty payer here means a
// truly unsignable tx, which we silently skip.
func (d AnchoringRefundDecorator) refundCosmos(ctx sdk.Context, tx sdk.Tx) error {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return nil
	}
	paid := feeTx.GetFee()
	payer := feeTx.FeePayer()
	if len(payer) == 0 {
		return nil
	}

	params, err := d.anchoringKeeper.Params.Get(ctx)
	if err != nil {
		return fmt.Errorf("get anchoring params: %w", err)
	}
	cap := params.AnchoringFee

	paidInCapDenom := paid.AmountOf(cap.Denom)
	if !paidInCapDenom.GT(cap.Amount) {
		return nil
	}

	refund := sdk.NewCoin(cap.Denom, paidInCapDenom.Sub(cap.Amount))
	if err := d.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		d.feeCollector,
		payer,
		sdk.NewCoins(refund),
	); err != nil {
		return fmt.Errorf("refund anchoring fee overage: %w", err)
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		anchoringtypes.EventTypeAnchoringFeeRefunded,
		sdk.NewAttribute(anchoringtypes.AttributeKeyFeePayer, sdk.AccAddress(payer).String()),
		sdk.NewAttribute(anchoringtypes.AttributeKeyRefundAmount, refund.String()),
	))
	return nil
}
