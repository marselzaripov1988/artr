package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrNotQualified      = errorsmod.Register(ModuleName, 1, "account is not qualified for noding")
	ErrPubkeyBusy        = errorsmod.Register(ModuleName, 2, "node with this public key is already validator")
	ErrNotFound          = errorsmod.Register(ModuleName, 3, "cannot find account data")
	ErrNotJailed         = errorsmod.Register(ModuleName, 4, "validator is not jailed")
	ErrJailPeriodNotOver = errorsmod.Register(ModuleName, 5, "jail period is not finished yet")
	ErrBannedForLifetime = errorsmod.Register(ModuleName, 6, "validator is banned for a lifetime")
	ErrAlreadyOn         = errorsmod.Register(ModuleName, 7, "noding is already on")
)
