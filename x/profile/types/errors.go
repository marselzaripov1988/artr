package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrNicknamePrefix       = errorsmod.Register(ModuleName, 1, "nickname cannot start with 'ARTR-' prefix")
	ErrNicknameAlreadyInUse = errorsmod.Register(ModuleName, 2, "nickname is already in use")
	ErrNotFound             = errorsmod.Register(ModuleName, 3, "profile not found")
	ErrAccountAlreadyExists = errorsmod.Register(ModuleName, 4, "account already exists")
	ErrNicknameTooShort     = errorsmod.Register(ModuleName, 5, "nickname is too short")
	ErrNicknameInvalidChars = errorsmod.Register(ModuleName, 6, "nickname contains invalid characters")
	ErrUnauthorized         = errorsmod.Register(ModuleName, 7, "sender is out of whitelist")
)
