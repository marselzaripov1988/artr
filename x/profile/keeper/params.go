package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/arterynetwork/artr/x/profile/types"
)

// GetParams returns the total set of subscription parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	if bz := ctx.KVStore(k.storeKey).Get(types.ParamsKey); bz != nil {
		k.cdc.MustUnmarshal(bz, &params)
	}
	return params
}

// SetParams sets the subscription parameters to the param space.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.Logger(ctx).Debug("SetParams", "params", params)
	// Множитель номера карты менять нельзя: по нему вычисляются уже
	// выданные номера. Раньше прежнее значение читалось отдельным ключом
	// из подпространства x/params, теперь берётся из набора в сторе модуля.
	//
	// Нулевое значение трактуется как "не задано" и заменяется прежним —
	// так было и раньше, поведение сохранено.
	if bz := ctx.KVStore(k.storeKey).Get(types.ParamsKey); bz != nil {
		var old types.Params
		k.cdc.MustUnmarshal(bz, &old)

		if params.CardMagic != old.CardMagic {
			if params.CardMagic == 0 {
				params.CardMagic = old.CardMagic
			} else {
				panic("card number magic must not be changed")
			}
		}
	}
	ctx.KVStore(k.storeKey).Set(types.ParamsKey, k.cdc.MustMarshal(&params))
}

func (k Keeper) AddFreeCreator(ctx sdk.Context, creator sdk.AccAddress) {
	params := k.GetParams(ctx)
	params.Creators = add(params.Creators, creator.String())
	k.SetParams(ctx, params)
}

func (k Keeper) RemoveFreeCreator(ctx sdk.Context, creator sdk.AccAddress) {
	params := k.GetParams(ctx)
	params.Creators = remove(params.Creators, creator.String())
	k.SetParams(ctx, params)
}

func (k Keeper) AddTokenRateSigner(ctx sdk.Context, signer sdk.AccAddress) {
	params := k.GetParams(ctx)
	params.TokenRateSigners = add(params.TokenRateSigners, signer.String())
	k.SetParams(ctx, params)
}

func (k Keeper) RemoveTokenRateSigner(ctx sdk.Context, signer sdk.AccAddress) {
	params := k.GetParams(ctx)
	params.TokenRateSigners = remove(params.TokenRateSigners, signer.String())
	k.SetParams(ctx, params)
}

func (k Keeper) AddVpnCurrentSigner(ctx sdk.Context, signer sdk.AccAddress) {
	p := k.GetParams(ctx)
	p.VpnSigners = add(p.VpnSigners, signer.String())
	k.SetParams(ctx, p)
}

func (k Keeper) RemoveVpnCurrentSigner(ctx sdk.Context, signer sdk.AccAddress) {
	p := k.GetParams(ctx)
	p.VpnSigners = remove(p.VpnSigners, signer.String())
	k.SetParams(ctx, p)
}

func (k Keeper) AddStorageCurrentSigner(ctx sdk.Context, signer sdk.AccAddress) {
	p := k.GetParams(ctx)
	p.StorageSigners = add(p.StorageSigners, signer.String())
	k.SetParams(ctx, p)
}

func (k Keeper) RemoveStorageCurrentSigner(ctx sdk.Context, signer sdk.AccAddress) {
	p := k.GetParams(ctx)
	p.StorageSigners = remove(p.StorageSigners, signer.String())
	k.SetParams(ctx, p)
}

func add(arr []string, item string) []string {
	for _, x := range arr {
		if x == item {
			return arr
		}
	}
	return append(arr, item)
}

func remove(arr []string, item string) []string {
	idx := -1
	for i, x := range arr {
		if x == item {
			idx = i
			break
		}
	}
	if idx < 0 {
		return arr
	}
	if idx != len(arr)-1 {
		arr[idx] = arr[len(arr)-1]
	}
	return arr[:len(arr)-1]
}
