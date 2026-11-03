package cmd

import (
	"bytes"
	"strings"
	"testing"

	"cosmossdk.io/log"
	"github.com/NVNM-Chain/nvnmchain/app"
	dbm "github.com/cosmos/cosmos-db"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"
)

func TestUpgradeHandlersCommand(t *testing.T) {
	cmd := upgradeHandlersCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{}) // otherwise cobra reads the test binary's os.Args
	require.NoError(t, cmd.Execute())

	// the command lists app.Upgrades, so every entry must reach the keeper
	a := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, false, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
	var want []string
	for _, u := range app.Upgrades {
		require.True(t, a.UpgradeKeeper.HasHandler(u.UpgradeName), u.UpgradeName)
		want = append(want, u.UpgradeName)
	}
	require.Equal(t, want, strings.Fields(out.String()))
}
