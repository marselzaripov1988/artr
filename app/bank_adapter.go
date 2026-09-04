package app

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/arterynetwork/artr/x/bank"
)

// bankForAnte переводит банковский кеепер Artery под интерфейс, которого
// ждёт обработчик подписи из x/auth.
//
// С SDK 0.50 кееперы принимают context.Context, а модули Artery написаны
// на sdk.Context и работают с ним внутри. Переписывать весь x/bank ради
// трёх методов, которые нужны одному только ante-обработчику, значило бы
// тронуть сотни мест ради чужого требования.
//
// Поэтому здесь переходник: он разворачивает контекст обратно и зовёт
// прежние методы. Граница проходит ровно там, где кончается SDK и
// начинается Artery.
type bankForAnte struct {
	k bank.Keeper
}

func (b bankForAnte) IsSendEnabledCoins(ctx context.Context, coins ...sdk.Coin) error {
	return b.k.IsSendEnabledCoins(sdk.UnwrapSDKContext(ctx), coins...)
}

func (b bankForAnte) SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
	return b.k.SendCoins(sdk.UnwrapSDKContext(ctx), from, to, amt)
}

func (b bankForAnte) SendCoinsFromAccountToModule(
	ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins,
) error {
	return b.k.SendCoinsFromAccountToModule(sdk.UnwrapSDKContext(ctx), senderAddr, recipientModule, amt)
}
