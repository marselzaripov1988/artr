package app

// Исходящие переводы из контрактов.
//
// Контракт, собравшийся отдать монеты, выпускает BankMsg.Send, а wasmd
// по умолчанию превращает его в cosmos.bank.v1beta1.MsgSend. У Artery
// банк свой, службы сообщений cosmos.bank в сети нет, и маршрутизатор
// отвечает "can't route message" — контракт может принимать деньги, но
// не может их отдавать.
//
// Здесь тот же BankMsg превращается в MsgSend банка Artery. Правки
// wasmd не требуется: у кеепера есть точка расширения для своих
// кодировщиков, и подменяется ровно один — банковский, остальные
// остаются штатными.
//
// Важно, что перевод идёт именно СООБЩЕНИЕМ, а не прямым вызовом
// SendCoins. Сообщение проходит через обработчик банка Artery, а тот
// берёт комиссию за перевод. Прямой вызов её бы обошёл — и контракты
// стали бы путём вывода крупных сумм без комиссии, той самой дырой,
// которую в сети уже закрывают потолком. Контракт здесь не привилегия,
// а обычный отправитель.

import (
	"github.com/pkg/errors"

	wasmKeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	bank "github.com/arterynetwork/artr/x/bank/types"
)

// encodeBankMsg — банковский кодировщик для контрактов.
//
// Повторяет разбор штатного EncodeBankMsg, отличаясь единственным: тип
// итогового сообщения.
func encodeBankMsg(sender sdk.AccAddress, msg *wasmvmtypes.BankMsg) ([]sdk.Msg, error) {
	// Сжигание контрактом не поддерживается намеренно. Штатный кодировщик
	// wasmd тоже его не умеет, а у Artery сжигание — часть экономики
	// (комиссия, отзыв делегации), и отдавать его контрактам как обычную
	// операцию не стоит без отдельного решения.
	if msg.Burn != nil {
		return nil, errors.New("artery: contracts are not allowed to burn coins")
	}
	if msg.Send == nil {
		return nil, errors.New("artery: unknown variant of BankMsg")
	}
	// Пустой перевод — не ошибка: штатный кодировщик молча возвращает
	// пустой список, и контракты на это рассчитывают.
	if len(msg.Send.Amount) == 0 {
		return nil, nil
	}

	amount, err := wasmKeeper.ConvertWasmCoinsToSdkCoins(msg.Send.Amount)
	if err != nil {
		return nil, errors.Wrap(err, "cannot convert coins")
	}

	to, err := sdk.AccAddressFromBech32(msg.Send.ToAddress)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse recipient address")
	}

	return []sdk.Msg{bank.NewMsgSend(sender, to, amount)}, nil
}

// arteryBankEncoder — набор кодировщиков с подменённым банковским.
//
// Merge внутри wasmd заменяет только заполненные поля, поэтому остальные
// (IBC, wasm, gov) остаются штатными.
func arteryBankEncoder() *wasmKeeper.MessageEncoders {
	return &wasmKeeper.MessageEncoders{Bank: encodeBankMsg}
}
