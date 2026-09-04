package main

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"cosmossdk.io/log"
	tmCfg "github.com/cometbft/cometbft/config"
	dbm "github.com/cosmos/cosmos-db"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	cryptoCodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptoTypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server"
	serverCmd "github.com/cosmos/cosmos-sdk/server/cmd"
	serverTypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"

	"github.com/arterynetwork/artr/app"
	"github.com/arterynetwork/artr/x/bank"
	bankcmd "github.com/arterynetwork/artr/x/bank/client/cli"
)

const flagInvCheckPeriod = "inv-check-period"

var invCheckPeriod uint

func main() {
	app.InitConfig()
	ec := app.NewEncodingConfig()

	app.ModuleBasics.RegisterInterfaces(ec.InterfaceRegistry)

	// Регистрация всех криптотипов SDK, включая ed25519. Раньше здесь стоял
	// только secp256k1, и на 0.42 это сходило с рук. В 0.45 команды вроде
	// `tendermint show-validator` маршалят ключ через реестр интерфейсов, и
	// консенсусный ключ валидатора (он ed25519) без этой регистрации падает
	// с "unable to resolve type URL /cosmos.crypto.ed25519.PubKey".
	cryptoCodec.RegisterInterfaces(ec.InterfaceRegistry)

	ec.InterfaceRegistry.RegisterInterface("tendermint.crypto.PubKey", (*cryptoTypes.PubKey)(nil), &secp256k1.PubKey{})
	// В 0.50 транзакция регистрируется под sdk.HasMsgs, а не sdk.Tx: у
	// последнего появились методы, которых у *tx.Tx нет, и регистрация под
	// ним падает с "doesn't actually implement interface". Так же делает и
	// сам SDK в types/tx.
	ec.InterfaceRegistry.RegisterInterface("cosmos.tx.v1beta1.Tx", (*sdk.HasMsgs)(nil), &tx.Tx{})

	clientCtx := ec.BuildClientContext().
		WithInput(os.Stdin).
		WithViper("")

	cobra.EnableCommandSorting = false
	rootCmd := &cobra.Command{
		Use:   "artrd",
		Short: "Artery Blockchain node (server + client)",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// SDK 0.43 убрал client.ReadHomeFlag: --home читается вместе с
			// остальными постоянными флагами через ReadPersistentCommandFlags.
			clientCtx, err := client.ReadPersistentCommandFlags(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			clientCtx, err = config.ReadFromClientConfig(clientCtx)
			if err != nil {
				return err
			}

			if err := client.SetCmdClientContextHandler(clientCtx, cmd); err != nil {
				return err
			}

			// Пустые шаблон и конфиг означают значения по умолчанию.
			// В 0.46 добавлен четвёртый аргумент — конфиг Tendermint.
			return server.InterceptConfigsPreRunHandler(cmd, "", nil, tmCfg.DefaultConfig())
		},
	}

	rootCmd.AddCommand(
		genutilcli.InitCmd(app.ModuleBasics, app.DefaultNodeHome),
		genutilcli.ValidateGenesisCmd(app.ModuleBasics),
		debug.Cmd(),
	)
	server.AddCommands(rootCmd, app.DefaultNodeHome, newApp(ec), exportAppState(ec), addModuleInitFlags)
	rootCmd.AddCommand(
		// StatusCommand переехал из client/rpc в server; keys.Commands
		// больше не принимает домашний каталог — он берётся из флагов.
		server.StatusCommand(),
		queryCmd(),
		txCmd(),
		keys.Commands(),
	)
	// Команды config в SDK 0.50 нет: она вынесена в отдельный инструмент
	// confix. Тянуть его сюда незачем — в этой сборке config chain-id
	// значение всё равно не сохранял, и девнет пишет client.toml сам.

	if err := serverCmd.Execute(rootCmd, "ARTR", app.DefaultNodeHome); err != nil {
		// server.ErrorCode из SDK 0.50 убран: код возврата больше не
		// передаётся через тип ошибки.
		{
			os.Exit(1)
		}
	}
}

func newApp(ec app.EncodingConfig) serverTypes.AppCreator {
	return func(logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts serverTypes.AppOptions) serverTypes.Application {
		// Настройки берутся штатным набором SDK из appOpts, а не из глобального
		// viper: он в этом месте пуст, и часть значений молча терялась.
		//
		// Отдельно важен SetChainID: с 0.47 приложение сверяет свой chain-id
		// с генезисом в InitChain и падает при расхождении. Без него был
		// "invalid chain-id on InitChain; expected: , got: ...".
		return app.NewArteryApp(
			logger, db, traceStore, true, invCheckPeriod, ec,
			server.DefaultBaseappOptions(appOpts)...,
		)
	}
}

func exportAppState(ec app.EncodingConfig) serverTypes.AppExporter {
	return func(
		logger log.Logger, db dbm.DB, traceStore io.Writer, height int64, forZeroHeight bool, jailWhiteList []string,
		_ serverTypes.AppOptions,
		// SDK 0.47 добавил последним аргументом список модулей для выборочного
		// экспорта. Artery экспортирует состояние целиком, поэтому не используется.
		modulesToExport []string,
	) (serverTypes.ExportedApp, error) {

		if height != -1 {
			aApp := app.NewArteryApp(logger, db, traceStore, false, uint(1), ec)
			err := aApp.LoadHeight(height)
			if err != nil {
				return serverTypes.ExportedApp{}, err
			}
			return aApp.ExportAppStateAndValidators(forZeroHeight, jailWhiteList, modulesToExport)
		}

		aApp := app.NewArteryApp(logger, db, traceStore, true, uint(1), ec)

		return aApp.ExportAppStateAndValidators(forZeroHeight, jailWhiteList, modulesToExport)
	}
}

func addModuleInitFlags(_ *cobra.Command) {}

func queryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:     "query",
		Aliases: []string{"q"},
		Short:   "Querying subcommands",
	}

	queryCmd.AddCommand(
		// authcmd.GetAccountCmd из SDK 0.50 убран: запрос счёта переведён
		// на autocli, который здесь не подключён. Команда query account
		// временно недоступна — счёт читается через REST или gRPC.
		flags.LineBreak,
		// Своя tendermint-validator-set держалась на rpc.GetValidators,
		// которого в 0.50 нет. Штатная команда делает то же самое.
		rpc.ValidatorCommand(),
		server.QueryBlockCmd(),
		authcmd.QueryTxsByEventsCmd(),
		authcmd.QueryTxCmd(),
		flags.LineBreak,
	)

	// add modules' query commands
	app.ModuleBasics.AddQueryCommands(queryCmd)

	return queryCmd
}

func txCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:   "tx",
		Short: "Transactions subcommands",
	}

	txCmd.AddCommand(
		bankcmd.CmdSend(),
		bankcmd.CmdBurn(),
		flags.LineBreak,
		authcmd.GetSignCommand(),
		authcmd.GetMultiSignCommand(),
		flags.LineBreak,
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
		flags.LineBreak,
	)

	// add modules' tx commands
	app.ModuleBasics.AddTxCommands(txCmd)

	// remove auth and bank commands as they're mounted under the root tx command
	var cmdsToRemove []*cobra.Command

	for _, cmd := range txCmd.Commands() {
		if cmd.Use == authtypes.ModuleName || cmd.Use == bank.ModuleName {
			cmdsToRemove = append(cmdsToRemove, cmd)
		}
	}

	txCmd.RemoveCommand(cmdsToRemove...)

	return txCmd
}
