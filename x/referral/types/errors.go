package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrParentNil          = errorsmod.Register(ModuleName, 1, "parentAcc cannot be nil")
	ErrRegistrationClosed = errorsmod.Register(ModuleName, 2, "referrer is inactive for too long")
	ErrNotFound           = errorsmod.Register(ModuleName, 3, "account is out of the referral structure")
)
