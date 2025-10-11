package ante

import (
	"errors"

	circuitante "cosmossdk.io/x/circuit/ante"
	circuitkeeper "cosmossdk.io/x/circuit/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	cosmosante "github.com/cosmos/evm/ante/cosmos"
	evmante "github.com/cosmos/evm/ante/evm"
	ibcante "github.com/cosmos/ibc-go/v10/modules/core/ante"
	"github.com/cosmos/ibc-go/v10/modules/core/keeper"
	consumerante "github.com/cosmos/interchain-security/v7/app/consumer/ante"
	ibcconsumerkeeper "github.com/cosmos/interchain-security/v7/x/ccv/consumer/keeper"
)

// HandlerOptions extend the SDK's AnteHandler options by r	equiring the IBC
// channel keeper.
type HandlerOptions struct {
	EvmOptions     EVMHandlerOptions
	IBCKeeper      *keeper.Keeper
	CircuitKeeper  *circuitkeeper.Keeper
	ConsumerKeeper ibcconsumerkeeper.Keeper
}

// Validate checks if the keepers are defined
func (options HandlerOptions) Validate() error {
	if options.EvmOptions.Validate() != nil {
		return options.EvmOptions.Validate()
	}
	if options.IBCKeeper == nil {
		return errors.New("ibc keeper is required for ante builder")
	}
	if options.CircuitKeeper == nil {
		return errors.New("circuit keeper is required for ante builder")
	}
	return nil
}

// newCosmosAnteHandler constructor
func newCosmosAnteHandler(options HandlerOptions) sdk.AnteHandler {
	anteDecorators := []sdk.AnteDecorator{
		cosmosante.NewRejectMessagesDecorator(), // reject MsgEthereumTxs
		consumerante.NewMsgFilterDecorator(options.ConsumerKeeper),
		consumerante.NewDisabledModulesDecorator("/cosmos.evidence", "/cosmos.slashing"),
		ante.NewSetUpContextDecorator(),
		circuitante.NewCircuitBreakerDecorator(options.CircuitKeeper),
		ante.NewExtensionOptionsDecorator(options.EvmOptions.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.EvmOptions.AccountKeeper),
		cosmosante.NewMinGasPriceDecorator(options.EvmOptions.FeeMarketKeeper, options.EvmOptions.EvmKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.EvmOptions.AccountKeeper),
		ante.NewDeductFeeDecorator(
			options.EvmOptions.AccountKeeper,
			options.EvmOptions.BankKeeper,
			options.EvmOptions.FeegrantKeeper,
			options.EvmOptions.TxFeeChecker,
		),
		// SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewSetPubKeyDecorator(options.EvmOptions.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.EvmOptions.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.EvmOptions.AccountKeeper, options.EvmOptions.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.EvmOptions.AccountKeeper, options.EvmOptions.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.EvmOptions.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
		evmante.NewGasWantedDecorator(options.EvmOptions.EvmKeeper, options.EvmOptions.FeeMarketKeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...)
}
