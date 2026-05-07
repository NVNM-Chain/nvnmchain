package app

import (
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/precompile"
	"github.com/cosmos/cosmos-sdk/codec"
	precompiletypes "github.com/cosmos/evm/precompiles/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
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

	// remove default staking and distribution precompiles
	delete(corePrecompiles, common.HexToAddress(evmtypes.StakingPrecompileAddress))
	delete(corePrecompiles, common.HexToAddress(evmtypes.DistributionPrecompileAddress))

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
