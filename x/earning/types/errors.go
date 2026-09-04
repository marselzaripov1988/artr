package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrAlreadyListed = errorsmod.Register(ModuleName, 1, "account is already in list")
	ErrTooLate       = errorsmod.Register(ModuleName, 2, "too late, cannot schedule for the past")
	ErrLocked        = errorsmod.Register(ModuleName, 3, "earner list is locked")
	ErrNotLocked     = errorsmod.Register(ModuleName, 4, "earner list is not locked")
	ErrNoMoney       = errorsmod.Register(ModuleName, 5, "there are no coins in VPN&storage module accounts")
)
