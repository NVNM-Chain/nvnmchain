package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/NVNM-Chain/nvnmchain/x/tax/types"
)

func (k msgServer) UpdateParams(ctx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if _, err := k.addressCodec.StringToBytes(req.Authority); err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	if req.Authority != params.TaxAddress {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid sender; expected taxAddress %s, got %s", params.TaxAddress, req.Authority)
	}

	updateParams, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	if req.Tax != "" {
		updateParams.Tax, err = math.LegacyNewDecFromStr(req.Tax)
		if err != nil {
			return nil, err
		}
		// Check against MaxTax
		if updateParams.Tax.GT(types.MaxTax) {
			return nil, fmt.Errorf("tax %s cannot exceed maximum of %s", updateParams.Tax, types.MaxTax)
		}
	}

	if req.TaxAddress != "" {
		taxAddr, err := k.addressCodec.StringToBytes(req.TaxAddress)
		if err != nil {
			return nil, errorsmod.Wrap(err, "invalid tax address")
		}
		if k.bankKeeper.BlockedAddr(taxAddr) {
			return nil, fmt.Errorf("tax address %s is blocked", req.TaxAddress)
		}
		updateParams.TaxAddress = req.TaxAddress
	}

	if err := updateParams.Validate(); err != nil {
		return nil, err
	}

	if err := k.Params.Set(ctx, updateParams); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
