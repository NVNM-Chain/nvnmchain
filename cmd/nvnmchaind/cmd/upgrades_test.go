package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/NVNM-Chain/nvnmchain/app"
	"github.com/stretchr/testify/require"
)

func TestUpgradeHandlersCommand(t *testing.T) {
	cmd := upgradeHandlersCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	require.NoError(t, cmd.Execute())

	var want []string
	for _, u := range app.Upgrades {
		want = append(want, u.UpgradeName)
	}
	require.Equal(t, want, strings.Fields(out.String()))
}
