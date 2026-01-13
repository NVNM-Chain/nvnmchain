package chainsuite

import (
	"cosmossdk.io/x/upgrade"

	"github.com/cosmos/ibc-go/modules/capability"
	transfer "github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	ibccore "github.com/cosmos/ibc-go/v10/modules/core"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ccvprovider "github.com/cosmos/interchain-security/v7/x/ccv/provider"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/cosmos/cosmos-sdk/x/mint"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"
)

// EVMEncoding returns a TestEncodingConfig that includes EVM crypto types.
// This is needed to decode transactions from chains that use ethsecp256k1 keys.
func EVMEncoding() *testutil.TestEncodingConfig {
	cfg := testutil.MakeTestEncodingConfig(
		auth.AppModuleBasic{},
		genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic(
			[]govclient.ProposalHandler{
				paramsclient.ProposalHandler,
			},
		),
		params.AppModuleBasic{},
		slashing.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		consensus.AppModuleBasic{},
		transfer.AppModuleBasic{},
		ibccore.AppModuleBasic{},
		ibctm.AppModuleBasic{},
		ccvprovider.AppModuleBasic{},
	)

	// Register EVM crypto types (ethsecp256k1)
	registerEVMCryptoInterfaces(cfg.InterfaceRegistry)

	return &cfg
}

// registerEVMCryptoInterfaces registers the EVM crypto key types.
func registerEVMCryptoInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*cryptotypes.PubKey)(nil), &EthSecp256k1PubKey{})
	registry.RegisterImplementations((*cryptotypes.PrivKey)(nil), &EthSecp256k1PrivKey{})
}
