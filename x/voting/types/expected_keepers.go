package types

import (
	"context"
	"time"

	"cosmossdk.io/math"
	upgrade "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	params "github.com/cosmos/cosmos-sdk/x/params/types"

	bank "github.com/arterynetwork/artr/x/bank/types"
	"github.com/arterynetwork/artr/x/delegating"
	noding "github.com/arterynetwork/artr/x/noding/types"
	profile "github.com/arterynetwork/artr/x/profile/types"
	referral "github.com/arterynetwork/artr/x/referral/types"
)

// ParamSubspace defines the expected Subspace interfacace
type ParamSubspace interface {
	WithKeyTable(table params.KeyTable) params.Subspace
	Get(ctx sdk.Context, key []byte, ptr interface{})
	GetParamSet(ctx sdk.Context, ps params.ParamSet)
	SetParamSet(ctx sdk.Context, ps params.ParamSet)
}

type ScheduleKeeper interface {
	ScheduleTask(ctx sdk.Context, time time.Time, event string, data []byte)
	DeleteAll(ctx sdk.Context, time time.Time, event string)
}

type UprgadeKeeper interface {
	ScheduleUpgrade(ctx context.Context, plan upgrade.Plan) error
	ClearUpgradePlan(ctx context.Context) error
}

type NodingKeeper interface {
	AddToStaff(ctx sdk.Context, acc sdk.AccAddress) error
	RemoveFromStaff(ctx sdk.Context, acc sdk.AccAddress) error

	GetParams(ctx sdk.Context) (params noding.Params)
	SetParams(ctx sdk.Context, params noding.Params)

	GeneralAmnesty(ctx sdk.Context)

	IsQualified(ctx sdk.Context, accAddr sdk.AccAddress) (result bool, delegation math.Int, reason noding.Reason, err error)
}

type DelegatingKeeper interface {
	GetParams(ctx sdk.Context) (params delegating.Params)
	SetParams(ctx sdk.Context, params delegating.Params)
}

type ReferralKeeper interface {
	GetParams(ctx sdk.Context) (params referral.Params)
	SetParams(ctx sdk.Context, params referral.Params)

	Get(ctx sdk.Context, acc string) (referral.Info, error)
}

type ProfileKeeper interface {
	GetParams(ctx sdk.Context) profile.Params
	SetParams(ctx sdk.Context, params profile.Params)

	AddFreeCreator(ctx sdk.Context, creator sdk.AccAddress)
	RemoveFreeCreator(ctx sdk.Context, creator sdk.AccAddress)
	AddTokenRateSigner(ctx sdk.Context, address sdk.AccAddress)
	RemoveTokenRateSigner(ctx sdk.Context, address sdk.AccAddress)
	AddVpnCurrentSigner(ctx sdk.Context, address sdk.AccAddress)
	RemoveVpnCurrentSigner(ctx sdk.Context, address sdk.AccAddress)
	AddStorageCurrentSigner(ctx sdk.Context, address sdk.AccAddress)
	RemoveStorageCurrentSigner(ctx sdk.Context, address sdk.AccAddress)
}

type signersKeeper interface {
	AddSigner(ctx sdk.Context, address sdk.AccAddress)
	RemoveSigner(ctx sdk.Context, address sdk.AccAddress)
}
type EarningKeeper signersKeeper

type BankKeeper interface {
	GetParams(ctx sdk.Context) bank.Params
	SetParams(ctx sdk.Context, params bank.Params)
	AddBlockedSender(ctx sdk.Context, acc sdk.AccAddress)
	RemoveBlockedSender(ctx sdk.Context, acc sdk.AccAddress)
}
