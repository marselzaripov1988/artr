package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// module name
	ModuleName   = "artrbank"
	QuerierRoute = ModuleName
	StoreKey     = ModuleName
)

// addrLen — длина адреса аккаунта в байтах, должна совпадать с app.AddrLen.
const addrLen = 20

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
	// SDK 0.43 убрал sdk.AddrLen — в SDK адреса стали переменной длины.
	//
	// Подставлять вместо него address.Len нельзя: это 32 байта (длина
	// SHA-256 из ADR-028), а у Artery адреса двадцатибайтовые. Значение
	// продублировано здесь, потому что app.AddrLen отсюда недоступен —
	// пакет app импортирует этот, и получилась бы циклическая зависимость.
	if len(key) < addrLen {
		panic(fmt.Sprintf("unexpected account address key length; got: %d, expected: %d", len(key), addrLen))
	}
	addr := key[:addrLen]

	return sdk.AccAddress(addr)
}
