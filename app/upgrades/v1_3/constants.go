package v1_3

import (
	"cosmossdk.io/store/types"
	"github.com/NVNM-Chain/nvnmchain/app/upgrades"
)

const (
	// UpgradeName defines the on-chain upgrade name.
	UpgradeName = "v1.3.0"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: types.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{},
	},
}
