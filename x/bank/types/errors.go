package types

import (
	errorsmod "cosmossdk.io/errors"
)

// x/bank module sentinel errors
var (
	ErrNoInputs            = errorsmod.Register(ModuleName, 1, "no inputs to send transaction")
	ErrNoOutputs           = errorsmod.Register(ModuleName, 2, "no outputs to send transaction")
	ErrInputOutputMismatch = errorsmod.Register(ModuleName, 3, "sum inputs != sum outputs")
	ErrSendDisabled        = errorsmod.Register(ModuleName, 4, "send transactions are disabled")
)
