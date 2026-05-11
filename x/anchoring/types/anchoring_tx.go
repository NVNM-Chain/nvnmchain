package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
)

// AnchoringPrecompileAddress is declared here (no build tag) so non-EVM
// packages can reference it without depending on the build-tagged precompile.
var AnchoringPrecompileAddress = common.HexToAddress("0x0000000000000000000000000000000000000a00")

// IsCosmosAnchoringTx returns true for single-msg MsgAddRecord txs only.
// The single-msg check prevents bundles like [MsgAddRecord, MsgSend] from
// inheriting the fee cap.
func IsCosmosAnchoringTx(tx sdk.Tx) bool {
	if tx == nil {
		return false
	}
	msgs := tx.GetMsgs()
	if len(msgs) != 1 {
		return false
	}
	if _, ok := msgs[0].(*MsgAddRecord); ok {
		return true
	}
	return false
}

// IsEVMAnchoringTx returns true when the outer tx.To is the precompile.
// Contract-mediated calls have a different outer To and are excluded.
func IsEVMAnchoringTx(tx sdk.Tx) bool {
	if tx == nil {
		return false
	}
	msgs := tx.GetMsgs()
	if len(msgs) != 1 {
		return false
	}
	ethMsg, ok := msgs[0].(*evmtypes.MsgEthereumTx)
	if !ok {
		return false
	}
	ethTx := ethMsg.AsTransaction()
	if ethTx == nil {
		return false
	}
	to := ethTx.To()
	return to != nil && *to == AnchoringPrecompileAddress
}
