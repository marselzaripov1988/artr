package keeper

import (
	"github.com/arterynetwork/artr/x/delegating/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetParams returns the total set of delegating parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.Logger(ctx).Debug("GetParams")
	if bz := ctx.KVStore(k.mainStoreKey).Get(types.ParamsKey); bz != nil {
		k.cdc.MustUnmarshal(bz, &params)
	}
	k.Logger(ctx).Debug("GetParams", "params", params)
	return params
}

// SetParams sets the delegating parameters to the param space.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.Logger(ctx).Debug("SetParams", "params", params)
	ctx.KVStore(k.mainStoreKey).Set(types.ParamsKey, k.cdc.MustMarshal(&params))
}
