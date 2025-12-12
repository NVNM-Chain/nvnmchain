package app

import (
	"github.com/MANTRA-Chain/inveniam/x/document/precompile"
	"github.com/cosmos/evm/evmd"
)

func (app *App) configStaticPrecompiles() error {
	corePrecompiles := evmd.NewAvailableStaticPrecompiles(
		*app.StakingKeeper,
		app.DistrKeeper,
		app.BankKeeper,
		app.Erc20Keeper,
		app.TransferKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.EVMKeeper,
		app.GovKeeper,
		app.SlashingKeeper,
		app.AppCodec(),
	)

	docPrecompile, err := precompile.NewPrecompile(app.DocumentKeeper)
	if err != nil {
		return err
	}
	corePrecompiles[docPrecompile.Address()] = docPrecompile

	app.EVMKeeper.WithStaticPrecompiles(
		corePrecompiles,
	)

	return nil
}
