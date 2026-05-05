package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	corevm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/gorilla/mux"
	"github.com/spf13/cast"

	// Force-load the tracer engines to trigger registration due to Go-Ethereum v1.10.15 changes
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"

	abci "github.com/cometbft/cometbft/abci/types"

	clienthelpers "cosmossdk.io/client/v2/helpers"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/circuit"
	circuitkeeper "cosmossdk.io/x/circuit/keeper"
	circuittypes "cosmossdk.io/x/circuit/types"
	"cosmossdk.io/x/evidence"
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/nft"
	nftkeeper "cosmossdk.io/x/nft/keeper"
	nftmodule "cosmossdk.io/x/nft/module"
	"cosmossdk.io/x/upgrade"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"

	"github.com/NVNM-Chain/nvnmchain/app/ante"
	appparams "github.com/NVNM-Chain/nvnmchain/app/params"
	"github.com/NVNM-Chain/nvnmchain/app/upgrades"
	_ "github.com/NVNM-Chain/nvnmchain/client/docs/statik"
	"github.com/NVNM-Chain/nvnmchain/client/docs/swagger"
	taxkeeper "github.com/NVNM-Chain/nvnmchain/x/tax/keeper"
	tax "github.com/NVNM-Chain/nvnmchain/x/tax/module"
	taxtypes "github.com/NVNM-Chain/nvnmchain/x/tax/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	sigtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	txmodule "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/mint"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	evmosencoding "github.com/cosmos/evm/encoding"
	ethcommon "github.com/ethereum/go-ethereum/common"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	reflectionv1 "cosmossdk.io/api/cosmos/reflection/v1"
	"cosmossdk.io/client/v2/autocli"
	anchoringkeeper "github.com/NVNM-Chain/nvnmchain/x/anchoring/keeper"
	anchoring "github.com/NVNM-Chain/nvnmchain/x/anchoring/module"
	anchoringprecompile "github.com/NVNM-Chain/nvnmchain/x/anchoring/precompile"
	anchoringtypes "github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
	chainante "github.com/cosmos/evm/ante"
	evmaddress "github.com/cosmos/evm/encoding/address"
	evmmempool "github.com/cosmos/evm/mempool"
	srvflags "github.com/cosmos/evm/server/flags"
	cosmosevmutils "github.com/cosmos/evm/utils"
	"github.com/cosmos/evm/x/erc20"
	erc20keeper "github.com/cosmos/evm/x/erc20/keeper"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	"github.com/cosmos/evm/x/feemarket"
	feemarketkeeper "github.com/cosmos/evm/x/feemarket/keeper"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	ibccallbackskeeper "github.com/cosmos/evm/x/ibc/callbacks/keeper"
	"github.com/cosmos/evm/x/vm"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/cosmos/gogoproto/proto"
	ratelimit "github.com/cosmos/ibc-apps/modules/rate-limiting/v10"
	ratelimitkeeper "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/keeper"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
	ibccallbacks "github.com/cosmos/ibc-go/v10/modules/apps/callbacks"
	ibctransfer "github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	transferkeeper "github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v10/modules/core"
	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channelv2types "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"

	consumer "github.com/cosmos/interchain-security/v7/x/ccv/consumer"
	consumerkeeper "github.com/cosmos/interchain-security/v7/x/ccv/consumer/keeper"
	consumertypes "github.com/cosmos/interchain-security/v7/x/ccv/consumer/types"
	ccvdistr "github.com/cosmos/interchain-security/v7/x/ccv/democracy/distribution"
	ccvstaking "github.com/cosmos/interchain-security/v7/x/ccv/democracy/staking"
	ccvtypes "github.com/cosmos/interchain-security/v7/x/ccv/types"
)

// defaultEvmCoinInfo seeds the EVM keeper's fallback (cosmos/evm#816).
func defaultEvmCoinInfo(homePath string) evmtypes.EvmCoinInfo {
	if info, ok := loadEvmCoinInfoFromGenesis(homePath); ok {
		return info
	}
	return evmtypes.EvmCoinInfo{
		Denom:         FutureStakingDenom,
		ExtendedDenom: FutureStakingDenom,
		DisplayDenom:  "nvnm",
		Decimals:      evmtypes.EighteenDecimals.Uint32(),
	}
}

// loadEvmCoinInfoFromGenesis token-streams evm.params + bank.denom_metadata,
// skipping bank.balances/supply. Returns evm_denom-only info if metadata
// is missing.
func loadEvmCoinInfoFromGenesis(homePath string) (evmtypes.EvmCoinInfo, bool) {
	if homePath == "" {
		return evmtypes.EvmCoinInfo{}, false
	}
	f, err := os.Open(filepath.Join(homePath, "config", "genesis.json"))
	if err != nil {
		return evmtypes.EvmCoinInfo{}, false
	}
	defer f.Close()

	dec := json.NewDecoder(bufio.NewReader(f))
	if !seekObjectKey(dec, "app_state") {
		return evmtypes.EvmCoinInfo{}, false
	}
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return evmtypes.EvmCoinInfo{}, false
	}

	var evm struct {
		Params struct {
			EvmDenom             string `json:"evm_denom"`
			ExtendedDenomOptions *struct {
				ExtendedDenom string `json:"extended_denom"`
			} `json:"extended_denom_options"`
		} `json:"params"`
	}
	var metadata []banktypes.Metadata
	var seenEvm, seenBank bool
	for dec.More() && (!seenEvm || !seenBank) {
		t, err := dec.Token()
		if err != nil {
			return evmtypes.EvmCoinInfo{}, false
		}
		key, _ := t.(string)
		switch key {
		case evmtypes.ModuleName:
			if err := dec.Decode(&evm); err != nil {
				return evmtypes.EvmCoinInfo{}, false
			}
			seenEvm = true
		case banktypes.ModuleName:
			if err := streamBankDenomMetadata(dec, &metadata); err != nil {
				return evmtypes.EvmCoinInfo{}, false
			}
			seenBank = true
		default:
			if err := skipValue(dec); err != nil {
				return evmtypes.EvmCoinInfo{}, false
			}
		}
	}

	if evm.Params.EvmDenom == "" {
		return evmtypes.EvmCoinInfo{}, false
	}

	info := evmtypes.EvmCoinInfo{
		Denom:         evm.Params.EvmDenom,
		ExtendedDenom: evm.Params.EvmDenom,
		DisplayDenom:  evm.Params.EvmDenom,
		Decimals:      evmtypes.EighteenDecimals.Uint32(),
	}
	for i := range metadata {
		if metadata[i].Base != evm.Params.EvmDenom {
			continue
		}
		info.DisplayDenom = metadata[i].Display
		for _, u := range metadata[i].DenomUnits {
			if u.Denom == metadata[i].Display {
				info.Decimals = u.Exponent
				break
			}
		}
		break
	}
	if info.Decimals != evmtypes.EighteenDecimals.Uint32() {
		if evm.Params.ExtendedDenomOptions == nil {
			return evmtypes.EvmCoinInfo{}, false
		}
		info.ExtendedDenom = evm.Params.ExtendedDenomOptions.ExtendedDenom
	}
	return info, true
}

// seekObjectKey advances dec to the named key's value, skipping siblings.
func seekObjectKey(dec *json.Decoder, key string) bool {
	t, err := dec.Token()
	if err != nil || t != json.Delim('{') {
		return false
	}
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return false
		}
		k, ok := t.(string)
		if !ok {
			return false
		}
		if k == key {
			return true
		}
		if err := skipValue(dec); err != nil {
			return false
		}
	}
	return false
}

// streamBankDenomMetadata decodes only the bank module's denom_metadata.
func streamBankDenomMetadata(dec *json.Decoder, out *[]banktypes.Metadata) error {
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if t != json.Delim('{') {
		return fmt.Errorf("bank: expected object, got %v", t)
	}
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return err
		}
		k, ok := t.(string)
		if !ok {
			return fmt.Errorf("bank: expected key string, got %v", t)
		}
		if k == "denom_metadata" {
			if err := dec.Decode(out); err != nil {
				return err
			}
			continue
		}
		if err := skipValue(dec); err != nil {
			return err
		}
	}
	_, err = dec.Token() // closing }
	return err
}

// skipValue consumes one JSON value via token reads (no buffering).
func skipValue(dec *json.Decoder) error {
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if t != json.Delim('{') && t != json.Delim('[') {
		return nil
	}
	depth := 1
	for depth > 0 {
		t, err := dec.Token()
		if err != nil {
			return err
		}
		switch t {
		case json.Delim('{'), json.Delim('['):
			depth++
		case json.Delim('}'), json.Delim(']'):
			depth--
		}
	}
	return nil
}

func init() {
	// Replace evmos defaults
	// manually update the power reduction by replacing micro (u) -> atto (a) nvnm
	sdk.DefaultPowerReduction = cosmosevmutils.AttoPowerReduction
	stakingtypes.DefaultMinCommissionRate = math.LegacyZeroDec()

	// DefaultNodeHome default home directories for nvnmchaind
	var err error
	DefaultNodeHome, err = clienthelpers.GetNodeHomeDirectory(NodeDir)
	if err != nil {
		panic(err)
	}
}

const appName = "App"

const (
	// ContractMemoryLimit is the memory limit of each contract execution (in MiB)
	// constant value so all nodes run with the same limit.
	ContractMemoryLimit = uint32(32)
)

// We pull these out so we can set them with LDFLAGS in the Makefile
var (
	NodeDir = ".nvnmchain"
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string
	// Authority address
	Authority = "nvnm1qkptlvhg2cw5scmw0rcwdvfe6k5y9050pymmm9"
)

// module account permissions
var maccPerms = map[string][]string{
	authtypes.FeeCollectorName:     nil,
	distrtypes.ModuleName:          nil,
	minttypes.ModuleName:           {authtypes.Minter},
	stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
	stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
	govtypes.ModuleName:            {authtypes.Burner},
	nft.ModuleName:                 nil,
	// non sdk modules
	ibctransfertypes.ModuleName: {authtypes.Minter, authtypes.Burner},
	ratelimittypes.ModuleName:   nil,
	taxtypes.ModuleName:         nil,

	// Cosmos EVM modules
	evmtypes.ModuleName:       {authtypes.Minter, authtypes.Burner},
	feemarkettypes.ModuleName: nil,
	erc20types.ModuleName:     {authtypes.Minter, authtypes.Burner},

	// Consumer
	consumertypes.ConsumerRedistributeName:     nil,
	consumertypes.ConsumerToSendToProviderName: nil,

	// NVNMChain Specific Modules
	anchoringtypes.ModuleName: nil,
}

var Upgrades = []upgrades.Upgrade{}

var (
	_ runtime.AppI            = (*App)(nil)
	_ servertypes.Application = (*App)(nil)
)

// App extended ABCI application
type App struct {
	*baseapp.BaseApp
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry types.InterfaceRegistry
	clientCtx         client.Context

	pendingTxListeners []chainante.PendingTxListener

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper         authkeeper.AccountKeeper
	BankKeeper            bankkeeper.BaseKeeper
	StakingKeeper         *stakingkeeper.Keeper
	SlashingKeeper        slashingkeeper.Keeper
	MintKeeper            mintkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	GovKeeper             govkeeper.Keeper
	UpgradeKeeper         *upgradekeeper.Keeper
	ParamsKeeper          paramskeeper.Keeper //nolint:staticcheck
	EvidenceKeeper        evidencekeeper.Keeper
	FeeGrantKeeper        feegrantkeeper.Keeper
	NFTKeeper             nftkeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper
	CircuitKeeper         circuitkeeper.Keeper

	// IBC
	IBCKeeper       *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	TransferKeeper  transferkeeper.Keeper
	RateLimitKeeper ratelimitkeeper.Keeper
	CallbackKeeper  ibccallbackskeeper.ContractKeeper

	// NVNMChain keepers
	TaxKeeper       taxkeeper.Keeper
	AnchoringKeeper anchoringkeeper.Keeper

	// Cosmos EVM keepers
	FeeMarketKeeper feemarketkeeper.Keeper
	EVMKeeper       *evmkeeper.Keeper
	Erc20Keeper     erc20keeper.Keeper
	EVMMempool      *evmmempool.ExperimentalEVMMempool

	// Consumer
	ConsumerKeeper consumerkeeper.Keeper

	// the module manager
	ModuleManager      *module.Manager
	BasicModuleManager module.BasicManager

	// simulation manager
	sm *module.SimulationManager

	// module configurator
	configurator module.Configurator
}

// New returns a reference to an initialized App.
func New(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *App {
	evmChainID := cast.ToUint64(appOpts.Get(srvflags.EVMChainID))
	encodingConfig := evmosencoding.MakeConfig(evmChainID)
	appCodec := encodingConfig.Codec
	legacyAmino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	bApp := baseapp.NewBaseApp(appName, logger, db, encodingConfig.TxConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)
	bApp.SetTxEncoder(encodingConfig.TxConfig.TxEncoder())

	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
		minttypes.StoreKey, distrtypes.StoreKey, slashingtypes.StoreKey,
		govtypes.StoreKey, paramstypes.StoreKey, consensusparamtypes.StoreKey, upgradetypes.StoreKey, feegrant.StoreKey,
		evidencetypes.StoreKey,
		circuittypes.StoreKey,
		nftkeeper.StoreKey,
		// non sdk store keys
		ibcexported.StoreKey, ibctransfertypes.StoreKey,
		ratelimittypes.StoreKey,
		taxtypes.StoreKey,
		consumertypes.StoreKey, anchoringtypes.StoreKey,

		// Cosmos EVM store keys
		evmtypes.StoreKey, feemarkettypes.StoreKey, erc20types.StoreKey,
	)

	tkeys := storetypes.NewTransientStoreKeys(paramstypes.TStoreKey, evmtypes.TransientKey, feemarkettypes.TransientKey)
	memKeys := storetypes.NewMemoryStoreKeys()

	// register streaming services
	if err := bApp.RegisterStreamingServices(appOpts, keys); err != nil {
		panic(err)
	}

	app := &App{
		BaseApp:           bApp,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}

	app.ParamsKeeper = initParamsKeeper(
		appCodec,
		legacyAmino,
		keys[paramstypes.StoreKey],
		tkeys[paramstypes.TStoreKey],
	)

	// set the BaseApp's parameter store
	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[consensusparamtypes.StoreKey]),
		Authority,
		runtime.EventService{},
	)
	bApp.SetParamStore(&app.ConsensusParamsKeeper.ParamsStore)

	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		Authority,
	)
	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		app.AccountKeeper,
		BlockedAddresses(),
		Authority,
		logger,
	)
	app.BankKeeper.AppendSendRestriction(app.blockUserSendsToCCVRewardsBuffer)

	app.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
		app.AccountKeeper,
		&app.BankKeeper,
		Authority,
		evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)

	app.MintKeeper = mintkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[minttypes.StoreKey]),
		app.StakingKeeper,
		app.AccountKeeper,
		&app.BankKeeper,
		authtypes.FeeCollectorName,
		Authority,
	)

	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[distrtypes.StoreKey]),
		app.AccountKeeper,
		&app.BankKeeper,
		app.StakingKeeper,
		consumertypes.ConsumerRedistributeName,
		Authority,
	)

	app.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		legacyAmino,
		runtime.NewKVStoreService(keys[slashingtypes.StoreKey]),
		app.StakingKeeper,
		Authority,
	)

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(keys[feegrant.StoreKey]), app.AccountKeeper)

	app.CircuitKeeper = circuitkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[circuittypes.StoreKey]),
		Authority,
		app.AccountKeeper.AddressCodec(),
	)
	app.SetCircuitBreaker(&app.CircuitKeeper)

	// get skipUpgradeHeights from the app options
	skipUpgradeHeights := map[int64]bool{}
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}
	homePath := cast.ToString(appOpts.Get(flags.FlagHome))
	// set the governance module account as the authority for conducting upgrades
	app.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		runtime.NewKVStoreService(keys[upgradetypes.StoreKey]),
		appCodec,
		homePath,
		app.BaseApp,
		Authority,
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	// NOTE: slashing hook was removed since it's only relevant for consumerKeeper
	app.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(app.DistrKeeper.Hooks()),
	)

	// pre-initialize ConsumerKeeper to satsfy ibckeeper.NewKeeper
	// which would panic on nil or zero keeper
	// ConsumerKeeper implements StakingKeeper but all function calls result in no-ops so this is safe
	// communication over IBC is not affected by these changes
	app.ConsumerKeeper = consumerkeeper.NewNonZeroKeeper(
		appCodec,
		keys[consumertypes.StoreKey],
	)

	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[ibcexported.StoreKey]),
		nil,
		app.UpgradeKeeper,
		Authority,
	)

	// Create CCV consumer and modules
	app.ConsumerKeeper = consumerkeeper.NewKeeper(
		appCodec,
		keys[consumertypes.StoreKey],
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ConnectionKeeper,
		app.IBCKeeper.ClientKeeper,
		app.SlashingKeeper,
		app.BankKeeper,
		app.AccountKeeper,
		&app.TransferKeeper,
		app.IBCKeeper,
		authtypes.FeeCollectorName,
		Authority,
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)

	// consumer keeper satisfies the staking keeper interface
	// of the slashing module
	app.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		legacyAmino,
		runtime.NewKVStoreService(keys[slashingtypes.StoreKey]),
		&app.ConsumerKeeper,
		Authority,
	)

	// register slashing module StakingHooks to the consumer keeper
	app.ConsumerKeeper = *app.ConsumerKeeper.SetHooks(app.SlashingKeeper.Hooks())
	consumerModule := consumer.NewAppModule(app.ConsumerKeeper, app.GetSubspace(consumertypes.ModuleName))

	app.AnchoringKeeper = anchoringkeeper.NewKeeper(
		appCodec,
		app.AccountKeeper.AddressCodec(),
		runtime.NewKVStoreService(keys[anchoringtypes.StoreKey]),
	)

	app.BankKeeper.BaseSendKeeper = app.BankKeeper.SetHooks(
		banktypes.NewMultiBankHooks())

	// Register the proposal types
	// Deprecated: Avoid adding new handlers, instead use the new proposal flow
	// by granting the governance module the right to execute the message.
	// See: https://docs.cosmos.network/main/modules/gov#proposal-messages
	govRouter := govv1beta1.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler)
	govConfig := govtypes.DefaultConfig()
	/*
		Example of setting gov params:
		govConfig.MaxMetadataLen = 10000
	*/
	govKeeper := govkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[govtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		app.DistrKeeper,
		app.MsgServiceRouter(),
		govConfig,
		Authority,
	)

	// Set legacy router for backwards compatibility with gov v1beta1
	govKeeper.SetLegacyRouter(govRouter)

	app.GovKeeper = *govKeeper.SetHooks(
		govtypes.NewMultiGovHooks(
		// register the governance hooks
		),
	)

	app.TaxKeeper = taxkeeper.NewKeeper(
		appCodec,
		app.AccountKeeper.AddressCodec(),
		runtime.NewKVStoreService(keys[taxtypes.StoreKey]),
		logger,
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.FeeCollectorName,
	)

	// Create RateLimit keeper
	clientKeeper := app.IBCKeeper.ClientKeeper
	app.RateLimitKeeper = *ratelimitkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[ratelimittypes.StoreKey]),
		app.GetSubspace(ratelimittypes.ModuleName),
		Authority,
		app.BankKeeper,
		app.IBCKeeper.ChannelKeeper,
		clientKeeper,
		app.IBCKeeper.ChannelKeeper,
	)

	app.NFTKeeper = nftkeeper.NewKeeper(
		runtime.NewKVStoreService(keys[nftkeeper.StoreKey]),
		appCodec,
		app.AccountKeeper,
		app.BankKeeper,
	)

	// create evidence keeper with router
	evidenceKeeper := evidencekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[evidencetypes.StoreKey]),
		&app.ConsumerKeeper,
		app.SlashingKeeper,
		app.AccountKeeper.AddressCodec(),
		runtime.ProvideCometInfoService(),
	)
	// If evidence needs to be handled for the app, set routes in router here and seal
	app.EvidenceKeeper = *evidenceKeeper

	// Cosmos EVM keepers
	app.FeeMarketKeeper = feemarketkeeper.NewKeeper(
		appCodec,
		sdk.MustAccAddressFromBech32(Authority),
		keys[feemarkettypes.StoreKey],
		tkeys[feemarkettypes.TransientKey],
	)

	// Set up EVM keeper
	tracer := cast.ToString(appOpts.Get(srvflags.EVMTracer))

	app.EVMKeeper = evmkeeper.NewKeeper(
		appCodec, keys[evmtypes.StoreKey], tkeys[evmtypes.TransientKey],
		keys,
		sdk.MustAccAddressFromBech32(Authority),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		app.FeeMarketKeeper,
		&app.ConsensusParamsKeeper,
		&app.Erc20Keeper,
		evmChainID,
		tracer,
	)
	app.EVMKeeper = app.EVMKeeper.WithDefaultEvmCoinInfo(defaultEvmCoinInfo(homePath))

	// ERC20 Keeper
	app.Erc20Keeper = erc20keeper.NewKeeper(
		keys[erc20types.StoreKey],
		appCodec,
		sdk.MustAccAddressFromBech32(Authority),
		app.AccountKeeper,
		app.BankKeeper,
		app.EVMKeeper,
		app.StakingKeeper,
		&app.TransferKeeper,
	)

	// instantiate IBC transfer keeper AFTER the ERC-20 keeper to use it in the instantiation
	app.TransferKeeper = transferkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[ibctransfertypes.StoreKey]),
		nil,
		app.IBCKeeper.ChannelKeeper, // ICS4Wrapper: overridden after transfer stack is built via WithICS4Wrapper
		app.IBCKeeper.ChannelKeeper,
		bApp.MsgServiceRouter(),
		app.AccountKeeper,
		app.BankKeeper,
		Authority,
	)
	app.TransferKeeper.SetAddressCodec(evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32AccountAddrPrefix()))

	/*
		Create Transfer Stack

		IBCModule stack (outermost to innermost, i.e. call-entry order for RecvPacket/OnAck/OnTimeout):
			- IBC RateLimit middleware
			- IBC Callbacks middleware (with EVM ContractKeeper)
			- IBC Transfer

		SendPacket ICS4Wrapper chain (transfer → core IBC):
			transfer.SendTransfer -> callbacks.SendPacket -> ratelimit.SendPacket -> channel.SendPacket

		The callbacks middleware sits in the ICS4Wrapper send path so that malformed src_callback
		metadata is rejected at send time. Without this, a sender could include a syntactically
		invalid src_callback memo that passes at send time but permanently blocks
		OnAcknowledgementPacket / OnTimeoutPacket, locking funds indefinitely.

		RecvPacket / OnAcknowledgementPacket / OnTimeoutPacket execution order:
			channel -> ratelimit (guard, aborts if rate exceeded) -> callbacks -> transfer
	*/

	// create IBC module from top to bottom of stack
	var transferStack porttypes.IBCModule

	transferStack = ibctransfer.NewIBCModule(app.TransferKeeper)
	maxCallbackGas := uint64(1_000_000)
	// transferStack = erc20.NewIBCMiddleware(app.Erc20Keeper, transferStack)
	app.CallbackKeeper = ibccallbackskeeper.NewKeeper(
		app.AccountKeeper,
		app.EVMKeeper,
		app.Erc20Keeper,
	)
	cbMiddleware := ibccallbacks.NewIBCMiddleware(transferStack, app.RateLimitKeeper, app.CallbackKeeper, maxCallbackGas)
	transferStack = ratelimit.NewIBCMiddleware(app.RateLimitKeeper, &cbMiddleware)

	// Wire the ICS4Wrapper send path: transfer -> callbacks -> ratelimit -> channel.
	app.TransferKeeper.WithICS4Wrapper(&cbMiddleware)

	// Create static IBC router, add app routes, then set and seal it
	ibcRouter := porttypes.NewRouter().
		AddRoute(ibctransfertypes.ModuleName, transferStack).
		AddRoute(ccvtypes.ConsumerPortID, consumerModule)

	app.IBCKeeper.SetRouter(ibcRouter)

	if err := app.configStaticPrecompiles(appCodec); err != nil {
		panic(err)
	}

	storeProvider := app.IBCKeeper.ClientKeeper.GetStoreProvider()
	tmLightClientModule := ibctm.NewLightClientModule(appCodec, storeProvider)
	clientKeeper.AddRoute(ibctm.ModuleName, &tmLightClientModule)

	// optional: enable sign mode textual by overwriting the default tx config (after setting the bank keeper)
	txConfigOpts := authtx.ConfigOptions{
		EnabledSignModes:           append(authtx.DefaultSignModes, sigtypes.SignMode_SIGN_MODE_TEXTUAL),
		TextualCoinMetadataQueryFn: txmodule.NewBankKeeperCoinMetadataQueryFn(app.BankKeeper),
	}
	txConfig, err := authtx.NewTxConfigWithOptions(
		appCodec,
		txConfigOpts,
	)
	if err != nil {
		panic(err)
	}
	app.txConfig = txConfig

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.ModuleManager = module.NewManager(
		genutil.NewAppModule(
			app.AccountKeeper,
			&app.ConsumerKeeper,
			app,
			txConfig,
		),
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, nil),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, nil),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, nil),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper, nil, nil),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, nil, app.interfaceRegistry),
		ccvdistr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, *app.StakingKeeper, authtypes.FeeCollectorName, app.GetSubspace(distrtypes.ModuleName)),
		ccvstaking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		upgrade.NewAppModule(app.UpgradeKeeper, app.AccountKeeper.AddressCodec()),
		evidence.NewAppModule(app.EvidenceKeeper),
		params.NewAppModule(app.ParamsKeeper), //nolint:staticcheck
		nftmodule.NewAppModule(appCodec, app.NFTKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),
		circuit.NewAppModule(appCodec, app.CircuitKeeper),
		consumerModule,
		// non sdk modules
		ibc.NewAppModule(app.IBCKeeper),
		ibctransfer.NewAppModule(app.TransferKeeper),
		ibctm.NewAppModule(tmLightClientModule),
		ratelimit.NewAppModule(appCodec, app.RateLimitKeeper),

		// nvnmchain modules
		tax.NewAppModule(appCodec, app.TaxKeeper),
		anchoring.NewAppModule(appCodec, app.AnchoringKeeper),

		// Cosmos EVM modules
		vm.NewAppModule(app.EVMKeeper, app.AccountKeeper, app.BankKeeper, app.AccountKeeper.AddressCodec()),
		feemarket.NewAppModule(app.FeeMarketKeeper),
		erc20.NewAppModule(app.Erc20Keeper, app.AccountKeeper),
	)

	// BasicModuleManager defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration and genesis verification.
	// By default it is composed of all the module from the module manager.
	// Additionally, app module basics can be overwritten by passing them as argument.
	app.BasicModuleManager = module.NewBasicManagerFromManager(
		app.ModuleManager,
		map[string]module.AppModuleBasic{
			genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
			govtypes.ModuleName: gov.NewAppModuleBasic(
				[]govclient.ProposalHandler{
					paramsclient.ProposalHandler,
				},
			),
			ibctransfertypes.ModuleName: ibctransfer.AppModuleBasic{},
		})
	app.BasicModuleManager.RegisterLegacyAminoCodec(legacyAmino)
	app.BasicModuleManager.RegisterInterfaces(interfaceRegistry)

	// NOTE: upgrade module is required to be prioritized
	app.ModuleManager.SetOrderPreBlockers(
		upgradetypes.ModuleName,
		authtypes.ModuleName,
		evmtypes.ModuleName,
	)
	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	app.ModuleManager.SetOrderBeginBlockers(
		// distribute first to distribute rewards from ConsumerRedistributeName
		// to governators and community pool
		distrtypes.ModuleName,
		minttypes.ModuleName,

		// IBC modules
		ibcexported.ModuleName, ibctransfertypes.ModuleName,

		// Cosmos EVM BeginBlockers
		erc20types.ModuleName, feemarkettypes.ModuleName,
		evmtypes.ModuleName, // NOTE: EVM BeginBlocker must come after FeeMarket BeginBlocker

		taxtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		genutiltypes.ModuleName,
		govtypes.ModuleName,
		// additional non simd modules
		ratelimittypes.ModuleName,
		consumertypes.ModuleName,
	)

	app.ModuleManager.SetOrderEndBlockers(
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		genutiltypes.ModuleName,
		// Cosmos EVM EndBlockers
		evmtypes.ModuleName,
		erc20types.ModuleName,
		feemarkettypes.ModuleName,
		feegrant.ModuleName,
		// tax before consumer to ensure tax is collected prior to
		// sending funds to provider and sending to redistribute account
		taxtypes.ModuleName,
		ibctransfertypes.ModuleName,
		ibcexported.ModuleName,
		ratelimittypes.ModuleName,
		consumertypes.ModuleName,
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: The genutils module must also occur after auth so that it can access the params from auth.
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	genesisModuleOrder := []string{
		// simd modules
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		minttypes.ModuleName,
		evidencetypes.ModuleName,
		feegrant.ModuleName,
		nft.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		consensusparamtypes.ModuleName,
		circuittypes.ModuleName,
		ibcexported.ModuleName,

		// Cosmos EVM modules
		//
		// NOTE: feemarket module needs to be initialized before genutil module:
		// gentx transactions use MinGasPriceDecorator.AnteHandle
		evmtypes.ModuleName,
		feemarkettypes.ModuleName,
		erc20types.ModuleName,

		// additional non simd modules
		ibctransfertypes.ModuleName,
		genutiltypes.ModuleName,
		ratelimittypes.ModuleName,
		taxtypes.ModuleName,
		consumertypes.ModuleName,
		anchoringtypes.ModuleName,
	}
	app.ModuleManager.SetOrderInitGenesis(genesisModuleOrder...)
	app.ModuleManager.SetOrderExportGenesis(genesisModuleOrder...)

	// Uncomment if you want to set a custom migration order here.
	// app.ModuleManager.SetOrderMigrations(custom order)

	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	err = app.ModuleManager.RegisterServices(app.configurator)
	if err != nil {
		panic(err)
	}

	autocliv1.RegisterQueryServer(app.GRPCQueryRouter(), runtimeservices.NewAutoCLIQueryService(app.ModuleManager.Modules))

	reflectionSvc, err := runtimeservices.NewReflectionService()
	if err != nil {
		panic(err)
	}
	reflectionv1.RegisterReflectionServiceServer(app.GRPCQueryRouter(), reflectionSvc)

	// add test gRPC service for testing gRPC queries in isolation
	// testdata_pulsar.RegisterQueryServer(app.GRPCQueryRouter(), testdata_pulsar.QueryImpl{})

	// create the simulation manager and define the order of the modules for deterministic simulations
	//
	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(app.appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, nil),
	}
	app.sm = module.NewSimulationManagerFromAppModules(app.ModuleManager.Modules, overrideModules)

	app.sm.RegisterStoreDecoders()

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	maxGasWanted := cast.ToUint64(appOpts.Get(srvflags.EVMMaxTxGasWanted))

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetPreBlocker(app.PreBlocker)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)

	// set denom resolver to test variant.
	// app.FeeMarketKeeper.SetDenomResolver(&ante.DenomResolverImpl{
	//	StakingKeeper: &app.StakingKeeper,
	// })

	// must be before Loading version
	// requires the snapshot store to be created and registered as a BaseAppOption
	// see cmd/nvnmchaind/root.go: 206 - 214 approx
	if manager := app.SnapshotManager(); manager != nil {
		err := manager.RegisterExtensions()
		if err != nil {
			panic(fmt.Errorf("failed to register snapshot extension: %s", err))
		}
	}

	app.setAnteHandler(txConfig, maxGasWanted)
	// app.setPostHandler()

	// configure mempool
	prepareProposalHandler, processProposalHandler := app.configureEVMMempool(appOpts, logger)
	app.SetPrepareProposal(prepareProposalHandler)
	app.SetProcessProposal(processProposalHandler)

	// Register any on-chain upgrades.
	app.setupUpgradeStoreLoaders()
	app.setupUpgradeHandlers()

	// At startup, after all modules have been registered, check that all proto
	// annotations are correct.
	protoFiles, err := proto.MergedRegistry()
	if err != nil {
		panic(err)
	}
	err = msgservice.ValidateProtoAnnotations(protoFiles)
	if err != nil {
		// Once we switch to using protoreflect-based antehandlers, we might
		// want to panic here instead of logging a warning.
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
	}

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			panic(fmt.Errorf("error loading last version: %w", err))
		}
	}

	return app
}

func (app *App) setAnteHandler(txConfig client.TxConfig, maxGasWanted uint64) {
	evmHandlerOpts := NewEVMAnteHandlerOptionsFromApp(app, txConfig, maxGasWanted)

	if err := evmHandlerOpts.Validate(); err != nil {
		panic(err)
	}

	handlerOpts := ante.HandlerOptions{
		EvmOptions:     evmHandlerOpts.Options(),
		IBCKeeper:      app.IBCKeeper,
		CircuitKeeper:  &app.CircuitKeeper,
		ConsumerKeeper: app.ConsumerKeeper,
	}

	if err := handlerOpts.Validate(); err != nil {
		panic(err)
	}

	// Set the AnteHandler for the app
	app.SetAnteHandler(ante.NewAnteHandler(handlerOpts))
}

// RegisterPendingTxListener is used by json-rpc server to listen to pending transactions callback.
func (app *App) RegisterPendingTxListener(listener func(ethcommon.Hash)) {
	app.pendingTxListeners = append(app.pendingTxListeners, listener)
}

func (app *App) onPendingTx(hash ethcommon.Hash) {
	for _, listener := range app.pendingTxListeners {
		listener(hash)
	}
}

// no post handler currently
// func (app *App) setPostHandler() {
// }

// Name returns the name of the App
func (app *App) Name() string { return app.BaseApp.Name() }

// PreBlocker application updates every pre block
func (app *App) PreBlocker(ctx sdk.Context, _ *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	return app.ModuleManager.PreBlock(ctx)
}

// BeginBlocker application updates every begin block
func (app *App) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	return app.ModuleManager.BeginBlock(ctx)
}

// EndBlocker application updates every end block
func (app *App) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	return app.ModuleManager.EndBlock(ctx)
}

func (app *App) Configurator() module.Configurator {
	return app.configurator
}

// InitChainer application update at chain initialization
func (app *App) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}
	err := app.UpgradeKeeper.SetModuleVersionMap(ctx, app.ModuleManager.GetVersionMap())
	if err != nil {
		panic(err)
	}
	response, err := app.ModuleManager.InitGenesis(ctx, app.appCodec, genesisState)
	return response, err
}

// LoadHeight loads a particular height
func (app *App) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// LegacyAmino returns legacy amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *App) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *App) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns App's InterfaceRegistry
func (app *App) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// TxConfig returns App's TxConfig
func (app *App) TxConfig() client.TxConfig {
	return app.txConfig
}

func (app *App) SetClientCtx(clientCtx client.Context) {
	app.clientCtx = clientCtx
}

func (app *App) GetMempool() sdkmempool.ExtMempool {
	return app.EVMMempool
}

func (app *App) Close() error {
	var err error
	if m, ok := app.GetMempool().(*evmmempool.ExperimentalEVMMempool); ok && m != nil {
		err = m.Close()
	}
	err = errors.Join(err, app.BaseApp.Close())
	msg := "Application gracefully shutdown"
	if err == nil {
		app.Logger().Info(msg)
	} else {
		app.Logger().Error(msg, "error", err)
	}
	return err
}

// AutoCliOpts returns the autocli options for the app.
func (app *App) AutoCliOpts() autocli.AppOptions {
	modules := make(map[string]appmodule.AppModule, 0)
	for _, m := range app.ModuleManager.Modules {
		if moduleWithName, ok := m.(module.HasName); ok {
			moduleName := moduleWithName.Name()
			if appModule, ok := moduleWithName.(appmodule.AppModule); ok {
				modules[moduleName] = appModule
			}
		}
	}

	return autocli.AppOptions{
		Modules:               modules,
		ModuleOptions:         runtimeservices.ExtractAutoCLIOptions(app.ModuleManager.Modules),
		AddressCodec:          evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		ValidatorAddressCodec: evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		ConsensusAddressCodec: evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	}
}

// DefaultGenesis returns a default genesis from the registered AppModuleBasic's.
func (app *App) DefaultGenesis() map[string]json.RawMessage {
	genesis := app.BasicModuleManager.DefaultGenesis(app.appCodec)

	// Add mint denom configuration
	mintGenState := minttypes.DefaultGenesisState()
	mintGenState.Params.MintDenom = appparams.DefaultBondDenom
	genesis[minttypes.ModuleName] = app.appCodec.MustMarshalJSON(mintGenState)

	// Add EVM genesis configuration
	evmGenState := evmtypes.DefaultGenesisState()
	evmGenState.Params.ActiveStaticPrecompiles = evmtypes.AvailableStaticPrecompiles
	genesis[evmtypes.ModuleName] = app.appCodec.MustMarshalJSON(evmGenState)

	// Add ERC20 genesis configuration
	erc20GenState := erc20types.DefaultGenesisState()
	erc20GenState.TokenPairs = ExampleTokenPairs
	erc20GenState.NativePrecompiles = append(erc20GenState.NativePrecompiles, WTokenContractMainnet)
	genesis[erc20types.ModuleName] = app.appCodec.MustMarshalJSON(erc20GenState)

	return genesis
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *App) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetStoreKeys returns all the stored store keys.
func (app *App) GetStoreKeys() []storetypes.StoreKey {
	keys := make([]storetypes.StoreKey, 0, len(app.keys))
	for _, key := range app.keys {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Name() < keys[j].Name()
	})
	return keys
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *App) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *App) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *App) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// SimulationManager implements the SimulationApp interface
func (app *App) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *App) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register new CometBFT queries routes from grpc-gateway.
	cmtservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register node gRPC service for grpc-gateway.
	nodeservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register grpc-gateway routes for all modules.
	app.BasicModuleManager.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if apiConfig.Swagger {
		RegisterSwaggerAPI(clientCtx, apiSvr.Router)
	}
}

// RegisterSwaggerAPI registers swagger route with API Server.
func RegisterSwaggerAPI(ctx client.Context, rtr *mux.Router) {
	staticServer := http.FileServer(swagger.FS)
	rtr.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticServer))
	rtr.PathPrefix("/swagger/").Handler(staticServer)
	rtr.PathPrefix("/openapi/").Handler(staticServer)
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *App) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.GRPCQueryRouter(), clientCtx, app.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *App) RegisterTendermintService(clientCtx client.Context) {
	cmtApp := server.NewCometABCIWrapper(app)
	cmtservice.RegisterTendermintService(
		clientCtx,
		app.GRPCQueryRouter(),
		app.interfaceRegistry,
		cmtApp.Query,
	)
}

func (app *App) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg)
}

// configure store loader that checks if version == upgradeHeight and applies store upgrades
func (app *App) setupUpgradeStoreLoaders() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	for _, upgrade := range Upgrades {
		if upgradeInfo.Name == upgrade.UpgradeName {
			storeUpgrades := upgrade.StoreUpgrades
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
		}
	}
}

func (app *App) setupUpgradeHandlers() {
	for _, upgrade := range Upgrades {
		app.UpgradeKeeper.SetUpgradeHandler(
			upgrade.UpgradeName,
			upgrade.CreateUpgradeHandler(
				app.ModuleManager,
				app.configurator,
				&upgrades.UpgradeKeepers{},
				app.keys,
			),
		)
	}
}

func (app *App) blockUserSendsToCCVRewardsBuffer(
	ctx context.Context,
	fromAddr, toAddr sdk.AccAddress,
	_ sdk.Coins,
) (sdk.AccAddress, error) {
	if !toAddr.Equals(authtypes.NewModuleAddress(consumertypes.ConsumerToSendToProviderName)) {
		return toAddr, nil
	}
	if fromAddr.Equals(authtypes.NewModuleAddress(authtypes.FeeCollectorName)) {
		return toAddr, nil
	}
	if app.isIBCRefundTx(ctx, fromAddr) {
		return toAddr, nil
	}
	return nil, fmt.Errorf("sending to %s is restricted", consumertypes.ConsumerToSendToProviderName)
}

func (app *App) isIBCRefundTx(ctx context.Context, fromAddr sdk.AccAddress) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	txBytes := sdkCtx.TxBytes()
	if len(txBytes) == 0 {
		return fromAddr.Equals(authtypes.NewModuleAddress(ibctransfertypes.ModuleName))
	}
	tx, err := app.txConfig.TxDecoder()(txBytes)
	if err != nil {
		return false
	}
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return false
	}
	hasRefund := false
	for _, msg := range msgs {
		switch msg.(type) {
		case *channeltypes.MsgTimeout, *channeltypes.MsgTimeoutOnClose, *channeltypes.MsgAcknowledgement:
			hasRefund = true
		case *channelv2types.MsgTimeout, *channelv2types.MsgAcknowledgement:
			hasRefund = true
		case *ibcclienttypes.MsgUpdateClient:
			continue
		case *channeltypes.MsgRecvPacket, *channelv2types.MsgRecvPacket:
			return false
		default:
			return false
		}
	}
	return hasRefund
}

// GetMaccPerms returns a copy of the module account permissions
//
// NOTE: This is solely to be used for testing purposes.
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}

	return dupMaccPerms
}

// BlockedAddresses returns all the app's blocked account addresses.
func BlockedAddresses() map[string]bool {
	blockedAddrs := make(map[string]bool)

	maccPerms := GetMaccPerms()
	accs := make([]string, 0, len(maccPerms))
	for acc := range maccPerms {
		accs = append(accs, acc)
	}
	sort.Strings(accs)

	for _, acc := range accs {
		blockedAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	// Remove the fee-pool from the group of blocked recipient addresses in bank
	// this is required for the consumer chain to be able to send tokens to
	// the provider chain
	delete(blockedAddrs, authtypes.NewModuleAddress(
		consumertypes.ConsumerToSendToProviderName).String())

	for _, precompile := range evmtypes.AvailableStaticPrecompiles {
		blockedAddrs[cosmosevmutils.Bech32StringFromHexAddress(precompile)] = true
	}

	blockedAddrs[cosmosevmutils.Bech32StringFromHexAddress(anchoringprecompile.AnchoringPrecompileAddress.Hex())] = true

	for _, addr := range corevm.PrecompiledAddressesPrague {
		precompile := addr.Hex()
		blockedAddrs[cosmosevmutils.Bech32StringFromHexAddress(precompile)] = true
	}

	return blockedAddrs
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey) paramskeeper.Keeper { //nolint:staticcheck
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey) //nolint:staticcheck
	keyTable := ibcclienttypes.ParamKeyTable()
	paramsKeeper.Subspace(ratelimittypes.ModuleName).WithKeyTable(ratelimittypes.ParamKeyTable())
	paramsKeeper.Subspace(ibcexported.ModuleName).WithKeyTable(keyTable)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName).WithKeyTable(ibctransfertypes.ParamKeyTable())

	return paramsKeeper
}
