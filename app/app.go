package app

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rakyll/statik/fs"

	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	tmjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/libs/log"
	tmos "github.com/cometbft/cometbft/libs/os"

	bam "github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	nodeService "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	codecTypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptoTypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	config2 "github.com/cosmos/cosmos-sdk/server/config"
	serverTypes "github.com/cosmos/cosmos-sdk/server/types"
	storeTypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authKeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	consensusKeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusTypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradeKeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradeTypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	_ "github.com/arterynetwork/artr/client/docs/statik"
	"github.com/arterynetwork/artr/util"
	"github.com/arterynetwork/artr/x/bank"
	"github.com/arterynetwork/artr/x/delegating"
	"github.com/arterynetwork/artr/x/earning"
	earningKeeper "github.com/arterynetwork/artr/x/earning/keeper"
	earningTypes "github.com/arterynetwork/artr/x/earning/types"
	"github.com/arterynetwork/artr/x/noding"
	nodingKeeper "github.com/arterynetwork/artr/x/noding/keeper"
	nodingTypes "github.com/arterynetwork/artr/x/noding/types"
	"github.com/arterynetwork/artr/x/profile"
	profileKeeper "github.com/arterynetwork/artr/x/profile/keeper"
	profileTypes "github.com/arterynetwork/artr/x/profile/types"
	"github.com/arterynetwork/artr/x/referral"
	"github.com/arterynetwork/artr/x/schedule"
	scheduleKeeper "github.com/arterynetwork/artr/x/schedule/keeper"
	scheduleTypes "github.com/arterynetwork/artr/x/schedule/types"
	"github.com/arterynetwork/artr/x/voting"
	votingKeeper "github.com/arterynetwork/artr/x/voting/keeper"
	votingTypes "github.com/arterynetwork/artr/x/voting/types"
)

const appName = "artery"

var (
	// DefaultCLIHome default home directories for the application CLI
	DefaultCLIHome = os.ExpandEnv("$HOME/.artrcli")

	// DefaultNodeHome sets the folder where the applcation data and configuration will be stored
	DefaultNodeHome = os.ExpandEnv("$HOME/.artrd")

	// ModuleBasics The module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		bank.AppModuleBasic{},
		referral.AppModuleBasic{},
		profile.AppModuleBasic{},
		schedule.AppModuleBasic{},
		delegating.AppModuleBasic{},
		voting.AppModuleBasic{},
		noding.AppModuleBasic{},
		earning.AppModuleBasic{},
	)

	// module account permissions
	maccPerms = map[string][]string{
		authTypes.FeeCollectorName:      nil,
		util.SplittableFeeCollectorName: nil,
		noding.ModuleName:               nil,
		earning.ModuleName:              nil,
		earning.VpnCollectorName:        nil,
		earning.StorageCollectorName:    nil,
	}
)

// ArteryApp extended ABCI application
type ArteryApp struct {
	*bam.BaseApp
	ec             EncodingConfig
	invCheckPeriod uint

	// keys to access the substores
	keys map[string]*storeTypes.KVStoreKey

	// keepers
	accountKeeper    authKeeper.AccountKeeper
	bankKeeper       bank.Keeper
	upgradeKeeper    *upgradeKeeper.Keeper
	consensusKeeper  consensusKeeper.Keeper
	referralKeeper   *referral.Keeper
	profileKeeper    profileKeeper.Keeper
	scheduleKeeper   scheduleKeeper.Keeper
	delegatingKeeper *delegating.Keeper
	votingKeeper     votingKeeper.Keeper
	nodingKeeper     noding.Keeper
	earningKeeper    earning.Keeper

	// Module Manager
	mm *module.Manager

	// simulation manager
	sm *module.SimulationManager
}

// verify app interface at compile time
var _ serverTypes.Application = (*ArteryApp)(nil)

// NewArteryApp is a constructor function for ArteryApp
func NewArteryApp(
	logger log.Logger, db dbm.DB, traceStore io.Writer, loadLatest bool,
	invCheckPeriod uint, ec EncodingConfig, baseAppOptions ...func(*bam.BaseApp),
) *ArteryApp {
	// BaseApp handles interactions with Tendermint through the ABCI protocol
	bApp := bam.NewBaseApp(appName, logger, db, ec.TxConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(ec.InterfaceRegistry)

	keys := newKVStoreKeys(authTypes.StoreKey, bank.StoreKey, upgradeTypes.StoreKey,
		profileTypes.StoreKey, profileTypes.AliasStoreKey, profileTypes.CardStoreKey,
		scheduleTypes.StoreKey, referral.StoreKey, referral.IndexStoreKey, delegating.MainStoreKey,
		votingTypes.StoreKey, noding.StoreKey, noding.IdxStoreKey,
		earning.StoreKey,
		// SDK 0.47 вынес параметры консенсуса в отдельный модуль x/consensus
		// со своим стором.
		consensusTypes.StoreKey)

	//TODO: pass `ec.Marshaller` to all modules properly and use it properly in

	// Here you initialize your application with the store keys it requires
	var app = &ArteryApp{
		BaseApp:        bApp,
		ec:             ec,
		invCheckPeriod: invCheckPeriod,
		keys:           keys,
	}

	// С 0.47 параметры консенсуса живут в x/consensus, а не в подпространстве
	// x/params. Право менять их отдано модулю upgrade: своего гова у Artery
	// нет, а адрес модуля никому не принадлежит.
	app.consensusKeeper = consensusKeeper.NewKeeper(
		ec.Marshaler,
		keys[consensusTypes.StoreKey],
		authTypes.NewModuleAddress(upgradeTypes.ModuleName).String(),
	)
	bApp.SetParamStore(&app.consensusKeeper)

	// Scheduler handles block height based tasks
	app.scheduleKeeper = scheduleKeeper.NewKeeper(
		ec.Marshaler,
		keys[scheduleTypes.StoreKey],
	)

	//app.scheduleKeeper.AddHook("event-test", func(ctx sdk.Context, data []byte) {
	//	ctx.Logger().Error("test event called")
	//	addr, _ := sdk.AccAddressFromBech32("cosmos1ey3aa0uxndvdrvgyvsd0afyt69uet9avw7cseq")
	//	coins := sdk.NewCoins(sdk.NewCoin("artr", sdk.NewInt(1000)))
	//	app.bankKeeper.AddCoins(ctx, addr, coins)
	//})

	// The AccountKeeper handles address -> account lookups
	app.accountKeeper = authKeeper.NewAccountKeeper(
		ec.Marshaler,
		keys[authTypes.StoreKey],
		authTypes.ProtoBaseAccount,
		map[string][]string{
			authTypes.FeeCollectorName:        {},
			util.SplittableFeeCollectorName:   {},
			earningTypes.VpnCollectorName:     {},
			earningTypes.StorageCollectorName: {},
			earningTypes.ModuleName:           {},
		},
		// SDK 0.46 требует bech32-префикс аккаунтов; 0.47 добавил authority.
		Bech32PrefixAccAddr,
		authTypes.NewModuleAddress(upgradeTypes.ModuleName).String(),
	)

	// The BankKeeper allows you perform sdk.Coins interactions
	app.bankKeeper = bank.NewBaseKeeper(
		ec.Marshaler,
		keys[bank.StoreKey],
		app.accountKeeper,
		make(map[string]bool, 0),
	)

	//app.bankKeeper.AddHook("SetCoins", "test-event", func(ctx sdk.Context, acc authexported.Account) {
	//	logger.Error("Set coins hook", acc)
	//})

	app.referralKeeper = referral.NewKeeper(
		ec.Marshaler,
		keys[referral.StoreKey],
		keys[referral.IndexStoreKey],
		app.accountKeeper,
		app.scheduleKeeper,
		app.bankKeeper,
		app.bankKeeper,
	)

	app.profileKeeper = profileKeeper.NewKeeper(
		ec.Marshaler,
		keys[profileTypes.StoreKey],
		keys[profileTypes.AliasStoreKey],
		keys[profileTypes.CardStoreKey],
		app.accountKeeper,
		app.bankKeeper,
		app.referralKeeper,
		app.scheduleKeeper,
	)

	app.delegatingKeeper = delegating.NewKeeper(
		ec.Marshaler,
		keys[delegating.MainStoreKey],
		app.accountKeeper,
		app.scheduleKeeper,
		app.profileKeeper,
		app.bankKeeper,
		app.referralKeeper,
	)

	app.upgradeKeeper = upgradeKeeper.NewKeeper(
		map[int64]bool{},
		keys[upgradeTypes.StoreKey],
		ec.Marshaler,
		"",
		// SDK 0.43 добавил пятым аргументом ProtocolVersionSetter — через него
		// модуль апгрейда проставляет версию протокола в BaseApp.
		app.BaseApp,
		// SDK 0.46 добавил authority — адрес, которому разрешён апгрейд
		// сообщением MsgSoftwareUpgrade. У Artery апгрейды идут через x/voting,
		// а не сообщением, поэтому ставим адрес самого модуля: он никому не
		// принадлежит, и подставиться под него извне нельзя.
		authTypes.NewModuleAddress(upgradeTypes.ModuleName).String(),
	)

	app.nodingKeeper = nodingKeeper.NewKeeper(
		ec.Marshaler,
		keys[nodingTypes.StoreKey],
		keys[nodingTypes.IdxStoreKey],
		app.referralKeeper,
		app.accountKeeper,
		app.bankKeeper,
		authTypes.FeeCollectorName,
		util.SplittableFeeCollectorName,
	)

	app.earningKeeper = earningKeeper.NewKeeper(
		ec.Marshaler,
		keys[earningTypes.StoreKey],
		app.accountKeeper,
		app.bankKeeper,
		app.scheduleKeeper,
	)

	app.votingKeeper = votingKeeper.NewKeeper(
		ec.Marshaler,
		keys[votingTypes.StoreKey],
		app.scheduleKeeper,
		app.upgradeKeeper,
		app.nodingKeeper,
		app.delegatingKeeper,
		app.referralKeeper,
		app.profileKeeper,
		app.earningKeeper,
		app.bankKeeper,
	)

	app.referralKeeper.SetKeepers(app.nodingKeeper)
	app.delegatingKeeper.SetKeepers(app.nodingKeeper, app.earningKeeper)

	app.bankKeeper.AddHook("SetCoins", "update-referral",
		func(ctx sdk.Context, addr sdk.AccAddress) error {
			if err := app.referralKeeper.OnBalanceChanged(ctx, addr.String()); err != nil {
				return errors.Wrap(err, "update-referral hook error")
			}

			return nil
		})

	app.scheduleKeeper.AddHook(referral.StatusDowngradeHookName, app.referralKeeper.PerformDowngrade)
	app.scheduleKeeper.AddHook(referral.CompressionHookName, app.referralKeeper.PerformCompression)
	app.scheduleKeeper.AddHook(referral.TransitionTimeoutHookName, app.referralKeeper.PerformTransitionTimeout)
	app.scheduleKeeper.AddHook(profileTypes.RefreshHookName, app.profileKeeper.HandleRenewHook)
	app.scheduleKeeper.AddHook(profileTypes.RefreshImHookName, app.profileKeeper.HandleRenewImHook)
	app.scheduleKeeper.AddHook(votingTypes.VoteHookName, app.votingKeeper.ProcessSchedule)
	app.scheduleKeeper.AddHook(votingTypes.PollHookName, app.votingKeeper.EndPollHandler)
	app.scheduleKeeper.AddHook(delegating.RevokeHookName, app.delegatingKeeper.MustPerformRevoking)
	app.scheduleKeeper.AddHook(delegating.AccrueHookName, app.delegatingKeeper.MustPerformAccrue)
	app.scheduleKeeper.AddHook(referral.BanishHookName, app.referralKeeper.PerformBanish)

	app.referralKeeper.AddHook(referral.StatusUpdatedCallback, app.nodingKeeper.OnStatusUpdate)
	app.referralKeeper.AddHook(referral.StakeChangedCallback, app.nodingKeeper.OnStakeChanged)
	app.referralKeeper.AddHook(referral.BanishedCallback, app.delegatingKeeper.OnBanished)

	// Исторические обработчики апгрейдов (2.0.1 ... 2.5.8) удалены намеренно.
	//
	// Переход на новую версию SDK делается способом, который команда уже
	// применяла для v1 -> v2: экспорт состояния, офлайн-переписывание генезиса
	// (см. patch-genesis/) и запуск цепочки заново с высоты 1. Историю сеть
	// при этом не проигрывает, поэтому обработчики прошлых апгрейдов
	// становятся мёртвым кодом.
	//
	// Новые обработчики регистрируются здесь по мере появления.

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(
		schedule.NewAppModule(app.scheduleKeeper),
		auth.NewAppModule(ec.Marshaler, app.accountKeeper, nil, nil),
		bank.NewAppModule(app.bankKeeper, app.accountKeeper),
		upgrade.NewAppModule(app.upgradeKeeper),
		profile.NewAppModule(app.profileKeeper, app.accountKeeper),
		referral.NewAppModule(
			*app.referralKeeper, app.accountKeeper, app.scheduleKeeper, app.bankKeeper, app.bankKeeper,
		),
		delegating.NewAppModule(
			*app.delegatingKeeper, app.accountKeeper, app.scheduleKeeper, app.bankKeeper, app.profileKeeper,
			*app.referralKeeper,
		),
		noding.NewAppModule(
			app.nodingKeeper, *app.referralKeeper, app.accountKeeper, app.bankKeeper,
		),
		earning.NewAppModule(app.earningKeeper, app.bankKeeper, app.scheduleKeeper),
		voting.NewAppModule(
			app.votingKeeper, app.scheduleKeeper, app.upgradeKeeper, app.nodingKeeper, app.delegatingKeeper,
			*app.referralKeeper, app.profileKeeper, app.earningKeeper,
		),
	)

	app.RegisterInterfaces(ec.InterfaceRegistry)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.

	// SDK 0.45 требует, чтобы в списках порядка были перечислены ВСЕ модули,
	// а не только те, у которых есть блочные хуки. Раньше менеджер вызывал
	// хуки строго по списку, и недостающие модули просто пропускались.
	//
	// Дополнение списков поведения не меняет: непустой BeginBlock есть только
	// у noding и schedule (и у upgrade из SDK), и все трое стояли в списке
	// раньше — их взаимный порядок сохранён. У остальных тела пустые.
	app.mm.SetOrderBeginBlockers(
		upgradeTypes.ModuleName,
		noding.ModuleName,
		referral.ModuleName,
		delegating.ModuleName,
		scheduleTypes.ModuleName,
		// ниже — модули с пустым BeginBlock, порядок между ними безразличен
		authTypes.ModuleName,
		bank.ModuleName,
		profileTypes.ModuleName,
		votingTypes.ModuleName,
		earning.ModuleName,
	)
	// Непустой EndBlock только у noding — он и остаётся первым.
	app.mm.SetOrderEndBlockers(
		noding.ModuleName,
		upgradeTypes.ModuleName,
		referral.ModuleName,
		delegating.ModuleName,
		scheduleTypes.ModuleName,
		authTypes.ModuleName,
		bank.ModuleName,
		profileTypes.ModuleName,
		votingTypes.ModuleName,
		earning.ModuleName,
	)

	// Sets the order of Genesis - Order matters, genutil is to always come last
	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// Порядок здесь несёт смысл — модули читают состояние друг друга. Он
	// сохранён как был; upgrade дописан в конец, его InitGenesis ни от чего
	// не зависит и ни на что не влияет.
	app.mm.SetOrderInitGenesis(
		scheduleTypes.ModuleName,
		authTypes.ModuleName,
		referral.ModuleName,
		bank.ModuleName,
		profileTypes.ModuleName,
		delegating.ModuleName,
		noding.ModuleName,
		votingTypes.ModuleName,
		earning.ModuleName,
		upgradeTypes.ModuleName,
	)

	// Маршрутизация сообщений и запросов — только через сервисы:
	// legacy-роутеры (Router/QueryRouter) в 0.47 удалены.
	app.mm.RegisterServices(module.NewConfigurator(ec.Marshaler, app.MsgServiceRouter(), app.GRPCQueryRouter()))

	// The initChainer handles translating the genesis.json file into initial state for the network
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)

	// The AnteHandler handles signature verification and transaction pre-processing
	//
	// В SDK 0.43 позиционные аргументы заменены на структуру HandlerOptions,
	// а конструктор стал возвращать ошибку. FeegrantKeeper оставлен пустым:
	// модуля feegrant в Artery нет, и NewDeductFeeDecorator это допускает.
	anteHandler, err := ante.NewAnteHandler(
		ante.HandlerOptions{
			AccountKeeper:   app.accountKeeper,
			BankKeeper:      app.bankKeeper,
			SignModeHandler: ec.TxConfig.SignModeHandler(),
			SigGasConsumer:  ante.DefaultSigVerificationGasConsumer,
		},
	)
	if err != nil {
		panic(errors.Wrap(err, "cannot build ante handler"))
	}
	app.SetAnteHandler(anteHandler)

	// initialize stores
	app.MountKVStores(keys)

	if loadLatest {
		err := app.LoadLatestVersion()
		if err != nil {
			tmos.Exit(err.Error())
		}
	}

	return app
}

func (app *ArteryApp) RegisterInterfaces(registry codecTypes.InterfaceRegistry) {
	for _, am := range app.mm.Modules {
		// С 0.47 Manager.Modules хранит interface{}, поэтому нужна проверка.
		if basic, ok := am.(module.AppModuleBasic); ok {
			basic.RegisterInterfaces(registry)
		}
	}
	registry.RegisterInterface("tendermint.crypto.PubKey", (*cryptoTypes.PubKey)(nil), &secp256k1.PubKey{})
}

// GenesisState represents chain state at the start of the chain. Any initial state (account balances) are stored here.
type GenesisState map[string]json.RawMessage

// NewDefaultGenesisState generates the default state for the application.
func NewDefaultGenesisState(mrshl codec.JSONCodec) GenesisState {
	return ModuleBasics.DefaultGenesis(mrshl)
}

// InitChainer application update at chain initialization
func (app *ArteryApp) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState

	if err := tmjson.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

	// Карта версий модулей — основа системы миграций x/upgrade начиная с
	// SDK 0.43: менеджер сравнивает записанную версию с ConsensusVersion
	// модуля и по расхождению запускает миграцию. Без этого вызова карта
	// не записывается, стор модуля upgrade остаётся пустым, а пустое
	// IAVL-дерево с 0.46 ломает любой запрос состояния.
	app.upgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())

	return app.mm.InitGenesis(ctx, app.ec.Marshaler, genesisState)
}

// BeginBlocker application updates every begin block
func (app *ArteryApp) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return app.mm.BeginBlock(ctx, req)
}

// EndBlocker application updates every end block
func (app *ArteryApp) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.mm.EndBlock(ctx, req)
}

// LoadHeight loads a particular height
func (app *ArteryApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// Codec returns the application's sealed codec.
func (app *ArteryApp) Codec() codec.BinaryCodec {
	return app.ec.Marshaler
}

// SimulationManager implements the SimulationApp interface
func (app *ArteryApp) SimulationManager() *module.SimulationManager {
	return app.sm
}

// GetMaccPerms returns a mapping of the application's module account permissions.
func GetMaccPerms() map[string][]string {
	modAccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		modAccPerms[k] = v
	}
	return modAccPerms
}

func (app *ArteryApp) RegisterAPIRoutes(server *api.Server, apiConfig config2.APIConfig) {
	clientCtx := server.ClientCtx
	// Маршруты tx через grpc-gateway. Legacy REST (rpc.RegisterRoutes)
	// удалён из SDK в 0.46.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, server.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, server.GRPCGatewayRouter)

	// Register grpc-gateway routes for all modules.
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, server.GRPCGatewayRouter)

	if apiConfig.Swagger {
		RegisterSwaggerAPI(server.Router)
	}
}

func (app *ArteryApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.ec.InterfaceRegistry)
}

// RegisterNodeService — требование интерфейса server/types.Application
// начиная с SDK 0.47.
func (app *ArteryApp) RegisterNodeService(clientCtx client.Context) {
	nodeService.RegisterNodeService(clientCtx, app.GRPCQueryRouter())
}

func (app *ArteryApp) RegisterTendermintService(clientCtx client.Context) {
	// В 0.46 порядок аргументов изменён и добавлена функция ABCI-запроса.
	tmservice.RegisterTendermintService(
		clientCtx,
		app.BaseApp.GRPCQueryRouter(),
		app.ec.InterfaceRegistry,
		app.BaseApp.Query,
	)
}

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(rtr *mux.Router) {
	statikFS, err := fs.NewWithNamespace("swagger")
	if err != nil {
		panic(err)
	}

	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
}

// newKVStoreKeys повторяет sdk.NewKVStoreKeys, но без проверки assertNoPrefix,
// добавленной в SDK 0.45.
//
// Проверка запрещает имена сторов, где одно является строковым префиксом
// другого. У Artery таких пар три: noding / noding-index,
// referral / referral-index и profile / profileAliases, profileCards.
//
// Реальной коллизии за этим нет. Сторы монтируются в корневом multistore под
// префиксом "s/k:<имя>/", то есть сравниваются "s/k:noding/" и
// "s/k:noding-index/" — на позиции после имени стоят '/' и '-', диапазоны
// итераторов не пересекаются. Проверка в SDK сделана с запасом, по голой
// строке, без учёта завершающего разделителя.
//
// Переименовать сторы значило бы сменить раскладку состояния сразу в трёх
// модулях. Имена — часть формата существующей сети, поэтому сохраняем их,
// а от проверки отказываемся сознательно.
func newKVStoreKeys(names ...string) map[string]*storeTypes.KVStoreKey {
	keys := make(map[string]*storeTypes.KVStoreKey, len(names))
	for _, name := range names {
		keys[name] = sdk.NewKVStoreKey(name)
	}
	return keys
}
