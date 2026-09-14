package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	addressCodec "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"

	bankTypes "github.com/arterynetwork/artr/x/bank/types"
	delegatingTypes "github.com/arterynetwork/artr/x/delegating/types"
	profileTypes "github.com/arterynetwork/artr/x/profile/types"
	votingTypes "github.com/arterynetwork/artr/x/voting/types"
)

// Подписант сообщений.
//
// С SDK 0.50 подписанта определяет x/tx по опции cosmos.msg.v1.signer в
// самом proto, а не по методу GetSigners(). Протоколы Artery писались
// раньше этого требования, и до их разметки на 0.53 не отправлялась ни
// одна транзакция: artrd tx падал с "no cosmos.msg.v1.signer option
// found" ещё до подписи.
//
// Поймать это было некому. Тесты зовут обработчики напрямую и путь
// подписания не трогают вовсе, а стенды считались рабочими по одному
// факту идущих блоков — транзакций на них никто не слал. Поломка дожила
// до сервера. Эти проверки закрывают ту щель.

// TestEverySignerIsDeclared — ни одно сообщение не осталось без опции.
//
// Validate() обходит все сервисы Msg в связанных описаниях и пытается
// получить подписанта у каждого входного сообщения. Отдельного списка
// вести не нужно: новый модуль попадёт под проверку сам.
//
// Оговорка, из-за которой проверка едва не вышла пустой: Validate()
// пропускает сервисы без аннотации cosmos.msg.v1.service. Пока её не
// было ни у одного сервиса Artery, этот тест проходил бы, не проверив
// ничего. Если аннотацию когда-нибудь снимут, TestMsgServicesAreAnnotated
// ниже это заметит.
func TestEverySignerIsDeclared(t *testing.T) {
	ec := NewEncodingConfig()
	require.NoError(t, ec.InterfaceRegistry.SigningContext().Validate())
}

// TestMsgServicesAreAnnotated — сервисы Msg помечены как таковые.
//
// Сторож для теста выше: без этой аннотации Validate() молча
// пропускает сервис, и отсутствие подписанта остаётся незамеченным.
func TestMsgServicesAreAnnotated(t *testing.T) {
	ec := NewEncodingConfig()
	// Если хоть один сервис Artery потеряет аннотацию, число services,
	// которые Validate() обходит, упадёт — но сам он об этом не скажет.
	// Поэтому спрашиваем подписанта у образца из каждого модуля: это
	// проходит через ту же машинерию, что и настоящая подпись.
	for _, msg := range []sdk.Msg{
		&bankTypes.MsgSend{FromAddress: sample, ToAddress: sample},
		&delegatingTypes.MsgDelegate{Address: sample},
		&profileTypes.MsgSetStorageCurrent{Sender: sample, Address: other},
		&votingTypes.MsgPropose{Proposal: votingTypes.Proposal{Author: sample}},
	} {
		_, _, err := ec.Marshaler.GetMsgV1Signers(msg)
		require.NoErrorf(t, err, "%T: подписант не выводится", msg)
	}
}

const (
	sample = "artr1qqqdgnsem9prsxnvayd8ju0h2p4d4mvk4fatjg"
	other  = "artr1yhy6d3m4utltdml7w7zte7mqx5wyuskq9rr5vg"
)

// TestSignerIsTheRightField — подписант берётся из того поля, из
// которого брал GetSigners().
//
// Это важнее отсутствия опции. Опции нет — транзакция не собирается, и
// это видно сразу. Опция указывает не на то поле — транзакция
// собирается, требует подпись не того счёта, и выясняется это на живой
// сети. В profile есть пары sender/address, где подписывает не тот, о
// ком сообщение, а MsgPropose держит подписанта во вложенном Proposal.
func TestSignerIsTheRightField(t *testing.T) {
	ec := NewEncodingConfig()

	// Разбираем тем же кодеком, что отдан реестру, а не через
	// sdk.AccAddressFromBech32: тот смотрит в глобальный конфиг, который
	// в голом тестовом двоичном файле ещё не настроен на префикс artr.
	want, err := addressCodec.NewBech32Codec(Bech32PrefixAccAddr).StringToBytes(sample)
	require.NoError(t, err)

	for name, msg := range map[string]sdk.Msg{
		// Обычный случай: подписант — единственный адрес отправителя.
		"MsgSend": &bankTypes.MsgSend{FromAddress: sample, ToAddress: other},

		// Подписывает sender, а речь в сообщении про address. Указать
		// здесь address значило бы потребовать подпись у того, чьи
		// показания записывают.
		"MsgSetStorageCurrent": &profileTypes.MsgSetStorageCurrent{Sender: sample, Address: other},
		"MsgSetVpnCurrent":     &profileTypes.MsgSetVpnCurrent{Sender: sample, Address: other},

		// Счёт создаёт не тот, кому он достанется.
		"MsgCreateAccount": &profileTypes.MsgCreateAccount{Creator: sample, Address: other, Referrer: other},

		// Подписант лежит во вложенном сообщении: опция указывает на
		// поле-сообщение, а спуск внутрь делает уже x/tx.
		"MsgPropose":   &votingTypes.MsgPropose{Proposal: votingTypes.Proposal{Author: sample}},
		"MsgStartPoll": &votingTypes.MsgStartPoll{Poll: votingTypes.Poll{Author: sample}},
	} {
		t.Run(name, func(t *testing.T) {
			signers, _, err := ec.Marshaler.GetMsgV1Signers(msg)
			require.NoError(t, err)
			require.Len(t, signers, 1, "подписант должен быть ровно один")
			require.Equal(t, want, signers[0], "подписант взят не из того поля")
		})
	}
}
