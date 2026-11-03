package v1_3

import (
	"cosmossdk.io/store/types"
	"github.com/NVNM-Chain/nvnmchain/app/upgrades"
)

const (
	// UpgradeName defines the on-chain upgrade name.
	UpgradeName = "v1.3.0"
	// UpgradeNameV131 is the plan name scheduled on nvnm-1 at height 4390900.
	// It runs the same handler; v1.3.0 stays registered for nvnm-testnet-1.
	UpgradeNameV131 = "v1.3.1"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: types.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{},
	},
}

var UpgradeV131 = upgrades.Upgrade{
	UpgradeName:          UpgradeNameV131,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades:        Upgrade.StoreUpgrades,
}
