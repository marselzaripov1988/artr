package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/arterynetwork/artr/x/schedule/types"
)

func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	bz := ctx.KVStore(k.storeKey).Get(types.ParamsKey)
	if bz == nil {
		return types.Params{}
	}
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

func (k Keeper) setParams(ctx sdk.Context, params types.Params) {
	ctx.KVStore(k.storeKey).Set(types.ParamsKey, k.cdc.MustMarshal(&params))
}
