package types_test

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
)

// fakeTx satisfies sdk.Tx for the detection helpers (only GetMsgs is read).
type fakeTx struct{ msgs []sdk.Msg }

func (t fakeTx) GetMsgs() []sdk.Msg                    { return t.msgs }
func (t fakeTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }

func newEthTx(to *common.Address) sdk.Msg {
	return evmtypes.NewTx(&evmtypes.EvmTxArgs{
		Nonce: 1, GasLimit: 100_000, GasPrice: big.NewInt(1), To: to,
	})
}

func TestIsCosmosAnchoringTx(t *testing.T) {
	addRecord := &types.MsgAddRecord{Sender: "nvnm1xyz"}
	send := &banktypes.MsgSend{}

	tests := []struct {
		name string
		tx   sdk.Tx
		want bool
	}{
		{"single MsgAddRecord", fakeTx{msgs: []sdk.Msg{addRecord}}, true},
		{"nil tx", nil, false},
		{"empty msgs", fakeTx{}, false},
		{"multi-msg bundle with MsgAddRecord", fakeTx{msgs: []sdk.Msg{addRecord, send}}, false},
		{"single non-anchoring msg", fakeTx{msgs: []sdk.Msg{send}}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, types.IsCosmosAnchoringTx(tc.tx))
		})
	}
}

func TestIsEVMAnchoringTx(t *testing.T) {
	precompile := types.AnchoringPrecompileAddress
	other := common.HexToAddress("0x000000000000000000000000000000000000DEAD")

	tests := []struct {
		name string
		tx   sdk.Tx
		want bool
	}{
		{"to precompile", fakeTx{msgs: []sdk.Msg{newEthTx(&precompile)}}, true},
		{"to contract", fakeTx{msgs: []sdk.Msg{newEthTx(&other)}}, false},
		{"contract creation (nil to)", fakeTx{msgs: []sdk.Msg{newEthTx(nil)}}, false},
		{"multi-msg bundle with eth tx", fakeTx{msgs: []sdk.Msg{newEthTx(&precompile), &banktypes.MsgSend{}}}, false},
		{"nil tx", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, types.IsEVMAnchoringTx(tc.tx))
		})
	}
}
