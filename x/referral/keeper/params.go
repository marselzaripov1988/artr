package keeper

import (
	"github.com/arterynetwork/artr/x/referral/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetParams returns the total set of referral parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	if bz := ctx.KVStore(k.storeKey).Get(types.ParamsKey); bz != nil {
		k.cdc.MustUnmarshal(bz, &params)
	}
	return params
}

// SetParams sets the referral parameters to the param space.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.Logger(ctx).Debug("SetParams", "params", params)
	ctx.KVStore(k.storeKey).Set(types.ParamsKey, k.cdc.MustMarshal(&params))
}
