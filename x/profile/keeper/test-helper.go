//go:build testing
// +build testing

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DeleteProfile убирает профиль счёта. Только для проверок.
//
// Нужен, чтобы воспроизвести счёт, заведённый в обход MsgCreateAccount:
// такой есть в балансах и в реферальном дереве, но профиля у него нет.
// Именно на таком счёте начисление разыменовывало пустой указатель и
// молча умирало — см. TestAccrueSurvivesMissingProfile.
//
// Побочные указатели (по прозвищу, по номеру карты) остаются висеть, и
// это здесь верно: у счёта, которому профиль никогда не заводили, их
// тоже нет.
func (k Keeper) DeleteProfile(ctx sdk.Context, addr sdk.AccAddress) {
	k.profileStore(ctx).Delete(addr)
}
