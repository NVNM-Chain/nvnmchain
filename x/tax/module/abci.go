package tax

import (
	"github.com/NVNM-Chain/nvnmchain/x/tax/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/tax/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EndBlocker is called at the end of each block. It allocates the tax to the tax module account.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, telemetry.Now(), telemetry.MetricKeyEndBlocker)

	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}
	// if the tax is zero, no need to continue
	if params.Tax.IsZero() {
		return nil
	}

	// only allocate rewards if the block height is greater than 1
	if ctx.BlockHeight() > 1 {
		TaxAddress, err := sdk.AccAddressFromBech32(params.TaxAddress)
		if err != nil {
			return err
		}
		if err := k.AllocateTax(ctx, params.Tax, TaxAddress); err != nil {
			return err
		}
	}

	return nil
}
