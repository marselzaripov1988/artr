package app

import (
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	transferTypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

// Соответствие обоим интерфейсам проверяется компилятором, а не на слово.
var (
	_ authTypes.BankKeeper     = bankAdapter{}
	_ transferTypes.BankKeeper = bankAdapter{}
)
