package posthandler_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	"github.com/NVNM-Chain/nvnmchain/app/posthandler"
	testkeeper "github.com/NVNM-Chain/nvnmchain/testutil/keeper"
	anchoringtypes "github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
)

const capDenom = "anvnm"

type bankCall struct {
	from   string
	toAddr sdk.AccAddress
	coins  sdk.Coins
}

type mockBank struct{ calls []bankCall }

func (m *mockBank) SendCoinsFromModuleToAccount(_ context.Context, from string, to sdk.AccAddress, coins sdk.Coins) error {
	m.calls = append(m.calls, bankCall{from: from, toAddr: to, coins: coins})
	return nil
}

type fakeFeeTx struct {
	msgs  []sdk.Msg
	fee   sdk.Coins
	payer sdk.AccAddress
}

func (t fakeFeeTx) GetMsgs() []sdk.Msg                    { return t.msgs }
func (t fakeFeeTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }
func (t fakeFeeTx) GetGas() uint64                        { return 200_000 }
func (t fakeFeeTx) GetFee() sdk.Coins                     { return t.fee }
func (t fakeFeeTx) FeePayer() []byte                      { return t.payer }
func (t fakeFeeTx) FeeGranter() []byte                    { return nil }

func setupRefund(t *testing.T) (sdk.Context, *mockBank, posthandler.AnchoringRefundDecorator) {
	t.Helper()
	k, ctx, _ := testkeeper.AnchoringKeeper(t)
	params, err := k.Params.Get(ctx)
	require.NoError(t, err)
	params.AnchoringFee = sdk.NewCoin(capDenom, math.NewIntWithDecimal(1, 16)) // 1¢
	require.NoError(t, k.Params.Set(ctx, params))

	bk := &mockBank{}
	return ctx, bk, posthandler.NewAnchoringRefundDecorator(k, bk, authtypes.FeeCollectorName)
}

func runPost(t *testing.T, dec posthandler.AnchoringRefundDecorator, ctx sdk.Context, tx sdk.Tx, success bool) {
	t.Helper()
	_, err := dec.PostHandle(ctx, tx, false, success, func(c sdk.Context, _ sdk.Tx, _, _ bool) (sdk.Context, error) {
		return c, nil
	})
	require.NoError(t, err)
}

func TestAnchoringRefundDecorator(t *testing.T) {
	payer := sdk.AccAddress([]byte("payer_______________"))
	addRecord := &anchoringtypes.MsgAddRecord{Sender: payer.String()}
	paid5c := sdk.NewCoins(sdk.NewCoin(capDenom, math.NewIntWithDecimal(5, 16))) // 5¢

	tests := []struct {
		name       string
		fee        sdk.Coins
		msgs       []sdk.Msg
		success    bool
		wantRefund sdk.Coins // nil/empty == no refund expected
	}{
		{
			name:       "overage refunded",
			fee:        paid5c,
			msgs:       []sdk.Msg{addRecord},
			success:    true,
			wantRefund: sdk.NewCoins(sdk.NewCoin(capDenom, math.NewIntWithDecimal(4, 16))),
		},
		{
			name:    "below cap no refund",
			fee:     sdk.NewCoins(sdk.NewCoin(capDenom, math.NewIntWithDecimal(5, 15))), // 0.5¢
			msgs:    []sdk.Msg{addRecord},
			success: true,
		},
		{
			name:    "equal cap no refund",
			fee:     sdk.NewCoins(sdk.NewCoin(capDenom, math.NewIntWithDecimal(1, 16))),
			msgs:    []sdk.Msg{addRecord},
			success: true,
		},
		{
			name:    "multi-msg bundle skipped",
			fee:     paid5c,
			msgs:    []sdk.Msg{addRecord, &banktypes.MsgSend{}},
			success: true,
		},
		{
			name:    "failed tx no refund",
			fee:     paid5c,
			msgs:    []sdk.Msg{addRecord},
			success: false,
		},
		{
			name: "multi-denom: only cap-denom overage refunded",
			fee: sdk.NewCoins(
				sdk.NewCoin(capDenom, math.NewIntWithDecimal(5, 16)),
				sdk.NewCoin("uatom", math.NewInt(7_000_000)),
			),
			msgs:       []sdk.Msg{addRecord},
			success:    true,
			wantRefund: sdk.NewCoins(sdk.NewCoin(capDenom, math.NewIntWithDecimal(4, 16))),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, bk, dec := setupRefund(t)
			runPost(t, dec, ctx, fakeFeeTx{msgs: tc.msgs, fee: tc.fee, payer: payer}, tc.success)

			if len(tc.wantRefund) == 0 {
				require.Empty(t, bk.calls)
				return
			}
			require.Len(t, bk.calls, 1)
			require.Equal(t, authtypes.FeeCollectorName, bk.calls[0].from)
			require.Equal(t, payer, bk.calls[0].toAddr)
			require.Equal(t, tc.wantRefund, bk.calls[0].coins)
		})
	}
}
