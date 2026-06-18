package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) UpdateParams(ctx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if _, err := k.addressCodec.StringToBytes(req.Authority); err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	if req.Authority != params.Admin {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "invalid sender; expected admin")
	}

	updateParams, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	if req.Admin != "" {
		updateParams.Admin = req.Admin
	}

	if req.AnchoringFee != nil {
		if req.AnchoringFee.Denom == "" {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "anchoring fee denom cannot be empty")
		}
		updateParams.AnchoringFee = *req.AnchoringFee
	}

	// Auto-heal the stored sentinel from the EVM denom so Admin-only updates
	// on legacy state aren't blocked. Runs before Validate so the validator
	// sees the final value.
	updateParams.ResolveAnchoringFeeDenom()

	if err := updateParams.Validate(); err != nil {
		return nil, err
	}

	if err := k.Params.Set(ctx, updateParams); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
