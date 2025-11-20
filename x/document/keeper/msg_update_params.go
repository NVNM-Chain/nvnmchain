package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/MANTRA-Chain/inveniam/x/document/types"
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
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid sender; expected admin %s, got %s", params.Admin, req.Authority)
	}

	updateParams, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	if req.Admin != "" {
		updateParams.Admin = req.Admin
	}

	if err := updateParams.Validate(); err != nil {
		return nil, err
	}

	if err := k.Params.Set(ctx, updateParams); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
