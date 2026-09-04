package util

import (
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cryptoTypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

// Консенсусные ключи валидаторов Artery хранятся в bech32 c префиксом
// artrvalconspub — и в генезисе, и в сторе x/noding, и в сообщениях.
//
// До сих пор это делал пакет types/bech32/legacybech32 из SDK, но он
// объявлен устаревшим и в 0.50 удалён. Формат при этом остаётся прежним:
// bech32 поверх amino-маршалинга ключа. Менять его нельзя — это формат
// состояния работающей сети, — поэтому кодировку берём себе, повторяя
// удаляемый пакет один в один.
//
// Совпадение с оригиналом закреплено тестом: пока обе реализации
// существуют, он сверяет их на одних и тех же ключах.

// ParseConsPubKey разбирает консенсусный ключ валидатора из bech32.
func ParseConsPubKey(value string) (cryptoTypes.PubKey, error) {
	bz, err := sdk.GetFromBech32(value, sdk.GetConfig().GetBech32ConsensusPubPrefix())
	if err != nil {
		return nil, err
	}
	return legacy.PubKeyFromBytes(bz)
}

// MustParseConsPubKey разбирает ключ и паникует при ошибке.
//
// Нужна там, где ключ уже проверен по месту приёма и повторная обработка
// ошибки только загромождает код: в тестах и при чтении собственного
// стора, куда непроверенный ключ попасть не может.
func MustParseConsPubKey(value string) cryptoTypes.PubKey {
	pk, err := ParseConsPubKey(value)
	if err != nil {
		panic(err)
	}
	return pk
}

// FormatConsPubKey кодирует консенсусный ключ валидатора в bech32.
func FormatConsPubKey(pk cryptoTypes.PubKey) (string, error) {
	return bech32.ConvertAndEncode(
		sdk.GetConfig().GetBech32ConsensusPubPrefix(),
		legacy.Cdc.MustMarshal(pk),
	)
}

// MustFormatConsPubKey кодирует ключ и паникует при ошибке.
func MustFormatConsPubKey(pk cryptoTypes.PubKey) string {
	s, err := FormatConsPubKey(pk)
	if err != nil {
		panic(err)
	}
	return s
}
