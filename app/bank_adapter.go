package app

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/arterynetwork/artr/x/bank"
)

// bankAdapter переводит банковский кеепер Artery под интерфейсы, которых
// ждут от него чужие модули: обработчик подписи из x/auth и модуль
// переводов IBC.
//
// С SDK 0.50 кееперы принимают context.Context, а модули Artery написаны
// на sdk.Context и работают с ним внутри. Переписывать весь x/bank ради
// чужих требований незачем — переходник разворачивает контекст обратно.
// Граница проходит ровно там, где кончается SDK и начинается Artery.
//
// Часть методов отличается не только контекстом: у Artery GetBalance
// возвращает сразу все монеты счёта, а SpendableCoins — набор, тогда как
// IBC спрашивает по одному деному. Это тоже сводится здесь, а не правкой
// самого банка.
type bankAdapter struct {
	k bank.Keeper
}

// --- Общее для x/auth и IBC ---

func (b bankAdapter) SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
	return b.k.SendCoins(sdk.UnwrapSDKContext(ctx), from, to, amt)
}

func (b bankAdapter) SendCoinsFromAccountToModule(
	ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins,
) error {
	return b.k.SendCoinsFromAccountToModule(sdk.UnwrapSDKContext(ctx), senderAddr, recipientModule, amt)
}

func (b bankAdapter) IsSendEnabledCoins(ctx context.Context, coins ...sdk.Coin) error {
	return b.k.IsSendEnabledCoins(sdk.UnwrapSDKContext(ctx), coins...)
}

// --- Требуется модулю переводов IBC ---

func (b bankAdapter) MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return b.k.MintCoins(sdk.UnwrapSDKContext(ctx), moduleName, amt)
}

func (b bankAdapter) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return b.k.BurnCoins(sdk.UnwrapSDKContext(ctx), moduleName, amt)
}

func (b bankAdapter) SendCoinsFromModuleToAccount(
	ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins,
) error {
	return b.k.SendCoinsFromModuleToAccount(sdk.UnwrapSDKContext(ctx), senderModule, recipientAddr, amt)
}

func (b bankAdapter) BlockedAddr(addr sdk.AccAddress) bool {
	return b.k.BlockedAddr(addr)
}

func (b bankAdapter) HasDenomMetaData(ctx context.Context, denom string) bool {
	return b.k.HasDenomMetaData(sdk.UnwrapSDKContext(ctx), denom)
}

func (b bankAdapter) SetDenomMetaData(ctx context.Context, md bankTypes.Metadata) {
	b.k.SetDenomMetaData(sdk.UnwrapSDKContext(ctx), md)
}

// SpendableCoin — доступная сумма в одном деноме.
//
// У Artery такого метода нет: SpendableCoins отдаёт весь набор. Выбираем
// нужный деном здесь.
func (b bankAdapter) SpendableCoin(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	coins := b.k.SpendableCoins(sdk.UnwrapSDKContext(ctx), addr)
	return sdk.NewCoin(denom, coins.AmountOf(denom))
}

// GetAllBalances — у Artery это GetBalance: он и так возвращает все
// монеты счёта, а не одну.
func (b bankAdapter) GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	return b.k.GetBalance(sdk.UnwrapSDKContext(ctx), addr)
}
