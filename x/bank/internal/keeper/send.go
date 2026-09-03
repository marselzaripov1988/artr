package keeper

import (
	"github.com/pkg/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	storeTypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/arterynetwork/artr/util"
	"github.com/arterynetwork/artr/x/bank/types"
)

// SendKeeper defines a module interface that facilitates the transfer of coins
// between accounts without the possibility of creating coins.
type SendKeeper interface {
	ViewKeeper

	InputOutputCoins(ctx sdk.Context, inputs []types.Input, outputs []types.Output) error
	SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error

	SubtractCoins(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coins) error
	AddCoins(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coins) error

	SetBalance(ctx sdk.Context, addr sdk.AccAddress, balance sdk.Coins) error

	GetParams(ctx sdk.Context) types.Params
	SetParams(ctx sdk.Context, params types.Params)

	AddBlockedSender(ctx sdk.Context, acc sdk.AccAddress)
	RemoveBlockedSender(ctx sdk.Context, acc sdk.AccAddress)

	BlockedAddr(addr sdk.AccAddress) bool

	// Требование интерфейса x/auth/types.BankKeeper начиная с SDK 0.47.
	IsSendEnabledCoins(ctx sdk.Context, coins ...sdk.Coin) error

	AddHook(event string, name string, hook func(ctx sdk.Context, addr sdk.AccAddress) error)
}

var _ SendKeeper = (*BaseSendKeeper)(nil)

// BaseSendKeeper only allows transfers between accounts without the possibility of
// creating coins. It implements the SendKeeper interface.
type BaseSendKeeper struct {
	BaseViewKeeper

	cdc        codec.BinaryCodec
	ak         types.AccountKeeper
	storeKey   storeTypes.StoreKey

	// list of addresses that are restricted from receiving transactions
	blockedAddrs map[string]bool

	// hooks to call
	setCoinHooks map[string]func(ctx sdk.Context, addr sdk.AccAddress) error
}

func NewBaseSendKeeper(
	cdc codec.BinaryCodec, storeKey storeTypes.StoreKey, ak types.AccountKeeper, blockedAddrs map[string]bool,
) BaseSendKeeper {

	return BaseSendKeeper{
		BaseViewKeeper: NewBaseViewKeeper(cdc, storeKey, ak),
		cdc:            cdc,
		ak:             ak,
		storeKey:       storeKey,
		blockedAddrs:   blockedAddrs,
		setCoinHooks:   make(map[string]func(ctx sdk.Context, addr sdk.AccAddress) error),
	}
}

// InputOutputCoins handles a list of inputs and outputs. It does not emit any events, so a caller MUST emit them.
func (keeper BaseSendKeeper) InputOutputCoins(ctx sdk.Context, inputs []types.Input, outputs []types.Output) error {
	// Safety check ensuring that when sending coins the keeper must maintain the
	// Check supply invariant and validity of Coins.
	if err := types.ValidateInputsOutputs(inputs, outputs); err != nil {
		return err
	}

	for _, in := range inputs {
		if err := keeper.SubtractCoins(ctx, in.Address, in.Coins); err != nil {
			return err
		}
	}

	for _, out := range outputs {
		if err := keeper.AddCoins(ctx, out.Address, out.Coins); err != nil {
			return err
		}
	}

	return nil
}

// SendCoins moves coins from one account to another
func (keeper BaseSendKeeper) SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	util.EmitEvent(ctx,
		&types.EventTransfer{
			Sender:    fromAddr.String(),
			Recipient: toAddr.String(),
			Amount:    amt,
		},
	)

	if err := keeper.SubtractCoins(ctx, fromAddr, amt); err != nil {
		return err
	}

	if err := keeper.AddCoins(ctx, toAddr, amt); err != nil {
		return err
	}

	return nil
}

// SubtractCoins subtracts amt from the coins at the addr.
//
// CONTRACT: If the account is a vesting account, the amount has to be spendable.
func (k BaseSendKeeper) SubtractCoins(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coins) error {
	if !amt.IsValid() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, amt.String())
	}

	balance := k.GetBalance(ctx, addr)
	spendable := k.SpendableCoins(ctx, addr)

	_, hasNeg := spendable.SafeSub(amt...)
	if hasNeg {
		return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "%s is smaller than %s", spendable, amt)
	}

	return errors.Wrap(k.SetBalance(ctx, addr, sdk.NewCoins(balance.Sub(amt...)...)), "cannot set balance")
}

// AddCoins adds amt to the coins at the addr.
func (k BaseSendKeeper) AddCoins(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coins) error {
	if !amt.IsValid() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, amt.String())
	}

	balance := k.GetBalance(ctx, addr)
	return errors.Wrap(k.SetBalance(ctx, addr, sdk.NewCoins(balance.Add(amt...)...)), "cannot set balance")
}

func (keeper BaseSendKeeper) AddHook(event string, name string, hook func(ctx sdk.Context, addr sdk.AccAddress) error) {
	switch event {
	case "SetCoins":
		keeper.setCoinHooks[name] = hook
	default:
		panic(errors.Errorf("unknown event: %s", event))

	}
}

// SetBalance sets the balance (multiple coins) for an account by address. An error is returned upon failure.
func (k BaseSendKeeper) SetBalance(ctx sdk.Context, addr sdk.AccAddress, balance sdk.Coins) error {
	if !balance.IsValid() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, balance.String())
	}

	store := ctx.KVStore(k.storeKey)
	key := make([]byte, len(types.BalancesPrefix)+len(addr.Bytes()))
	copy(key, types.BalancesPrefix)
	copy(key[len(types.BalancesPrefix):], addr.Bytes())

	if balance.Empty() {
		store.Delete(key)
	} else {
		bz := k.cdc.MustMarshal(&types.Balance{Coins: balance})
		store.Set(key, bz)
	}

	if err := k.fireSetCoins(ctx, addr); err != nil {
		return errors.Wrap(err, "hook failed")
	}
	return nil
}

func (k BaseSendKeeper) fireSetCoins(ctx sdk.Context, addr sdk.AccAddress) error {
	if len(k.setCoinHooks) == 0 {
		return nil
	}

	for _, hook := range k.setCoinHooks {
		if err := hook(ctx, addr); err != nil {
			return err
		}
	}
	return nil
}

// GetMinSend возвращает минимальную сумму перевода.
//
// Раньше значение читалось отдельным ключом из подпространства x/params.
// После переноса параметров в стор модуля отдельного доступа к одному
// значению нет — оно часть общего набора.
func (keeper BaseSendKeeper) GetMinSend(ctx sdk.Context) int64 {
	return keeper.GetParams(ctx).MinSend
}

// SetMinSend задаёт минимальную сумму перевода.
func (keeper BaseSendKeeper) SetMinSend(ctx sdk.Context, minSend int64) {
	params := keeper.GetParams(ctx)
	params.MinSend = minSend
	keeper.SetParams(ctx, params)
}

// IsSendEnabledCoins сообщает, разрешена ли отправка указанных монет.
//
// Требование интерфейса x/auth/types.BankKeeper начиная с SDK 0.47. У Artery
// нет понятия денома, запрещённого к отправке: ограничения выражены иначе —
// минимальной суммой перевода и чёрным списком адресов отправителей. Поэтому
// здесь разрешено всё, а проверки выполняются в SendCoins.
func (keeper BaseSendKeeper) IsSendEnabledCoins(ctx sdk.Context, coins ...sdk.Coin) error {
	return nil
}

// BlockedSenderAddr checks if a given address is blacklisted (i.e restricted from
// sending funds)
func (keeper BaseSendKeeper) BlockedSenderAddr(ctx sdk.Context, addr sdk.AccAddress) bool {
	bech32 := addr.String()
	for _, v := range keeper.GetParams(ctx).BlockedSenders {
		if v == bech32 {
			return true
		}
	}
	return false
}

// BlacklistedAddr checks if a given address is blacklisted (i.e restricted from
// receiving funds)
func (keeper BaseSendKeeper) BlockedAddr(addr sdk.AccAddress) bool {
	return keeper.blockedAddrs[addr.String()]
}

// GetParams returns the total set of bank parameters.
func (k BaseSendKeeper) GetParams(ctx sdk.Context) (params types.Params) {
	if bz := ctx.KVStore(k.storeKey).Get(types.ParamsKey); bz != nil {
		k.cdc.MustUnmarshal(bz, &params)
	}
	return params
}

// SetParams sets the total set of bank parameters.
func (k BaseSendKeeper) SetParams(ctx sdk.Context, params types.Params) {
	ctx.KVStore(k.storeKey).Set(types.ParamsKey, k.cdc.MustMarshal(&params))
}

func (k BaseSendKeeper) AddBlockedSender(ctx sdk.Context, blockedSender sdk.AccAddress) {
	params := k.GetParams(ctx)
	util.AddStringOntoEnd(&params.BlockedSenders, blockedSender.String())
	k.SetParams(ctx, params)
}

func (k BaseSendKeeper) RemoveBlockedSender(ctx sdk.Context, blockedSender sdk.AccAddress) {
	params := k.GetParams(ctx)
	util.RemoveStringFast(&params.BlockedSenders, blockedSender.String())
	k.SetParams(ctx, params)
}
