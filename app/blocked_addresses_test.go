package app

import (
	"testing"

	anchoringprecompile "github.com/MANTRA-Chain/inveniam/x/anchoring/precompile"
	cosmosevmutils "github.com/cosmos/evm/utils"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	corevm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

func TestBlockedAddresses_BlocksAllPrecompiles(t *testing.T) {
	blockedAddrs := BlockedAddresses()

	for _, precompile := range evmtypes.AvailableStaticPrecompiles {
		bech32Addr := cosmosevmutils.Bech32StringFromHexAddress(precompile)
		require.Truef(t, blockedAddrs[bech32Addr], "expected cosmos/evm precompile %s to be blocked", precompile)
	}

	anchoringAddr := anchoringprecompile.AnchoringPrecompileAddress.Hex()
	anchoringBech32 := cosmosevmutils.Bech32StringFromHexAddress(anchoringAddr)
	require.Truef(t, blockedAddrs[anchoringBech32], "expected anchoring precompile %s to be blocked", anchoringAddr)

	for _, addr := range corevm.PrecompiledAddressesPrague {
		precompile := addr.Hex()
		bech32Addr := cosmosevmutils.Bech32StringFromHexAddress(precompile)
		require.Truef(t, blockedAddrs[bech32Addr], "expected go-ethereum precompile %s to be blocked", precompile)
	}
}
