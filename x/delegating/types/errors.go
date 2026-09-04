package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrNothingDelegated = errorsmod.Register(ModuleName, 1, "nothing's delegated")
	ErrLessThanMinimum  = errorsmod.Register(ModuleName, 2, "delegation is lass than minimum")
)
