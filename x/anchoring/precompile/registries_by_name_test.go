//go:build uint256

// External test package: the keeper test helper imports app, which imports
// this precompile, so the full-path test cannot live in package precompile.
package precompile_test

import (
	"path/filepath"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"

	appparams "github.com/NVNM-Chain/nvnmchain/app/params"
	keepertest "github.com/NVNM-Chain/nvnmchain/testutil/keeper"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/nameindex"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/precompile"
	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
)

// eoaStateDB reports every address as code-less, i.e. an EOA.
type eoaStateDB struct{ vm.StateDB }

func (eoaStateDB) GetCode(common.Address) []byte { return nil }

// TestRegistriesByName_Execute runs the full precompile path against a real
// index: selector dispatch, ABI decode, keeper query, gas, ABI encode, and the
// gate on both sides of the query/transaction line.
func TestRegistriesByName_Execute(t *testing.T) {
	appparams.SetAddressPrefixes()
	k, ctx, _ := keepertest.AnchoringKeeper(t)
	keepertest.MustCreateAnchoringRegistry(t, k, ctx, keepertest.TestSenderAddr, "fund-documents")
	keepertest.MustCreateAnchoringRegistry(t, k, ctx, keepertest.TestSenderAddr, "Fund-Archive")
	keepertest.MustCreateAnchoringRegistry(t, k, ctx, keepertest.TestSenderAddr, "unrelated")

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	store, err := nameindex.Open(filepath.Join(t.TempDir(), "index.db"), cdc)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	k.NameIndex = store
	require.NoError(t, k.BackfillNameIndex(ctx))

	p, err := precompile.NewPrecompile(k)
	require.NoError(t, err)

	eoa := common.HexToAddress("0x00000000000000000000000000000000000000e1")
	input, err := precompile.RegistriesByNameCall{
		Name:      "fund",
		MatchMode: uint8(types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_PREFIX),
	}.Encode()
	require.NoError(t, err)
	newCall := func() *vm.Contract {
		contract := vm.NewContract(eoa, precompile.AnchoringPrecompileAddress, nil, 1_000_000, nil)
		contract.Input = append(append([]byte{}, precompile.RegistriesByNameSelector[:]...), input...)
		return contract
	}

	// As a query: served, case-folded, and metered.
	qctx := ctx.WithIsCheckTx(true).WithGasMeter(storetypes.NewGasMeter(1_000_000))
	out, err := p.Execute(qctx, eoaStateDB{}, newCall(), false, eoa)
	require.NoError(t, err)
	var ret precompile.RegistriesByNameReturn
	_, err = ret.Decode(out)
	require.NoError(t, err)
	require.Len(t, ret.Registries, 2)
	require.ElementsMatch(t,
		[]string{"fund-documents", "Fund-Archive"},
		[]string{ret.Registries[0].Name, ret.Registries[1].Name})
	require.Positive(t, qctx.GasMeter().GasConsumed())

	// Byte-for-byte the same input in block execution: reverted before the
	// index is touched, with the reason in the revert data.
	out, err = p.Execute(ctx.WithIsCheckTx(false), eoaStateDB{}, newCall(), false, eoa)
	require.ErrorIs(t, err, vm.ErrExecutionReverted)
	require.Contains(t, string(out), precompile.ErrRegistriesByNameInTx.Error())
}
