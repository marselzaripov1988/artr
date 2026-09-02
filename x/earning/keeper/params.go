package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/arterynetwork/artr/x/earning/types"
)

// GetParams returns the total set of earning parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	bz := ctx.KVStore(k.storeKey).Get(types.ParamsKey)
	if bz == nil {
		return types.Params{}
	}
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams sets the earning parameters to the param space.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.Logger(ctx).Debug("SetParams", "params", params)
	ctx.KVStore(k.storeKey).Set(types.ParamsKey, k.cdc.MustMarshal(&params))
}

func (k Keeper) AddSigner(ctx sdk.Context, address sdk.AccAddress) {
	p := k.GetParams(ctx)
	p.Signers = append(p.Signers, address.String())
	k.SetParams(ctx, p)
}

func (k Keeper) RemoveSigner(ctx sdk.Context, address sdk.AccAddress) {
	p := k.GetParams(ctx)
	bech32 := address.String()
	for i, signer := range p.Signers {
		if signer == bech32 {
			last := len(p.Signers) - 1
			if i != last {
				p.Signers[i] = p.Signers[last]
			}
			p.Signers = p.Signers[:last]
			k.SetParams(ctx, p)
			return
		}
	}
}
