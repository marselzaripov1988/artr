package main

// Команда `artrd query account`.
//
// В SDK 0.50 штатную authcmd.GetAccountCmd убрали: запрос счёта
// переведён на autocli, а он здесь не подключён. Команда пропала, и
// посмотреть номер счёта с последовательностью стало нечем — только
// через REST или gRPC, то есть curl'ом.
//
// Само подписание транзакций от этого не страдает: клиент берёт номер и
// последовательность из службы счетов напрямую, не через CLI. Команда
// нужна человеку — разобраться, почему транзакция не принята, или
// свериться перед отправкой вручную.
//
// Подключать autocli ради одной команды не стали: он добавил бы
// вдобавок команды стандартных модулей SDK, которых у Artery нет или
// которые заменены своими — например cosmos.bank. Путаницы от этого
// больше, чем пользы.

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func cmdQueryAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account <адрес>",
		Short: "Показать счёт: номер, последовательность и открытый ключ",
		Long: "Показать счёт по адресу.\n\n" +
			"Номер счёта и последовательность нужны для подписи транзакции. " +
			"Обычно клиент подставляет их сам; указывать вручную через " +
			"--account-number и --sequence приходится разве что при офлайновой " +
			"подписи. Неверный номер даёт отказ вида " +
			"\"signature verification failed\", а не внятное сообщение о номере.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			// Разбираем адрес до запроса: иначе пустой или испорченный
			// адрес уйдёт на узел и вернётся оттуда невнятным
			// "Address cannot be empty".
			addr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			res, err := authtypes.NewQueryClient(clientCtx).Account(
				cmd.Context(),
				&authtypes.QueryAccountRequest{Address: addr.String()},
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res.Account)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
