package keeper

import (
	"github.com/arterynetwork/artr/x/voting/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetParams returns the total set of voting parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	if bz := ctx.KVStore(k.storeKey).Get(types.KeyParams); bz != nil {
		k.cdc.MustUnmarshal(bz, &params)
	}
	return params
}

// SetParams sets the voting parameters to the param space.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	ctx.KVStore(k.storeKey).Set(types.KeyParams, k.cdc.MustMarshal(&params))
}
