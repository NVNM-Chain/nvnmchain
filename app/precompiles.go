package app

import (
	"github.com/MANTRA-Chain/inveniam/x/document/precompile"
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
