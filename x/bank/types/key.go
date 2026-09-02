package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
)

const (
	// module name
	ModuleName   = "artrbank"
	QuerierRoute = ModuleName
	StoreKey     = ModuleName
)

// KVStore keys
var (
	BalancesPrefix      = []byte("balances")
	SupplyKey           = []byte{0x00}
	DenomMetadataPrefix = []byte{0x1}

	// ParamsKey — параметры модуля, перенесённые из подпространства x/params.
	// Значение 0x02 свободно: 0x00 занят предложением монет, 0x01 —
	// метаданными деномов, балансы лежат под своим строковым префиксом.
	ParamsKey = []byte{0x02}
)

// DenomMetadataKey returns the denomination metadata key.
func DenomMetadataKey(denom string) []byte {
	d := []byte(denom)
	return append(DenomMetadataPrefix, d...)
}

// AddressFromBalancesStore returns an account address from a balances prefix
// store. The key must not contain the perfix BalancesPrefix as the prefix store
// iterator discards the actual prefix.
func AddressFromBalancesStore(key []byte) sdk.AccAddress {
	// SDK 0.43 убрал sdk.AddrLen — адреса стали переменной длины. У Artery
	// они по-прежнему ровно 20 байт (см. app/config.go), что и есть address.Len.
	addr := key[:address.Len]
	if len(addr) != address.Len {
		panic(fmt.Sprintf("unexpected account address key length; got: %d, expected: %d", len(addr), address.Len))
	}

	return sdk.AccAddress(addr)
}
