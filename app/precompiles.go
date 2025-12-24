package app

import (
	"github.com/MANTRA-Chain/inveniam/x/anchoring/precompile"
	"github.com/cosmos/cosmos-sdk/codec"
	precompiletypes "github.com/cosmos/evm/precompiles/types"
)

func (app *App) configStaticPrecompiles(appCodec codec.Codec) error {
	corePrecompiles := precompiletypes.DefaultStaticPrecompiles(
		*app.StakingKeeper,
		app.DistrKeeper,
		app.BankKeeper,
		&app.Erc20Keeper,
		&app.TransferKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.GovKeeper,
		app.SlashingKeeper,
		appCodec,
	)

	anchoringPrecompile, err := precompile.NewPrecompile(app.AnchoringKeeper)
	if err != nil {
		return err
	}
	corePrecompiles[anchoringPrecompile.Address()] = anchoringPrecompile

	app.EVMKeeper.WithStaticPrecompiles(
		corePrecompiles,
	)

	return nil
}
