//go:build testing
// +build testing

package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/arterynetwork/artr/app"
	"github.com/arterynetwork/artr/x/bank/types"
)

// voucher — описание денома, пришедшего по IBC. Именно такое заводит
// модуль переводов при первом поступлении извне.
func voucher(base string) bankTypes.Metadata {
	return bankTypes.Metadata{
		Base:        base,
		Display:     base,
		Name:        base,
		Symbol:      base,
		Description: "IBC voucher",
		DenomUnits: []*bankTypes.DenomUnit{
			{Denom: base, Exponent: 0},
		},
	}
}

func TestDenomMetadataRoundTrip(t *testing.T) {
	a, cleanup, ctx := app.NewAppFromGenesis(nil)
	defer cleanup()
	k := a.GetBankKeeper()

	const denom = "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"

	require.False(t, k.HasDenomMetaData(ctx, denom), "метаданных быть не должно до записи")
	_, found := k.GetDenomMetaData(ctx, denom)
	require.False(t, found)

	md := voucher(denom)
	k.SetDenomMetaData(ctx, md)

	require.True(t, k.HasDenomMetaData(ctx, denom))
	got, found := k.GetDenomMetaData(ctx, denom)
	require.True(t, found)
	require.Equal(t, md, got, "метаданные изменились при круговом пути")
}

// TestDenomMetadataIsOutsideBalances проверяет само правило, а не его
// следствие: метаданные не должны попадать в диапазон, по которому идут
// обходы балансов.
//
// Проверка структурная и потому надёжнее поведенческой: она падает сразу,
// как только ключи разъедутся, независимо от того, проявилось это уже на
// каких-то данных или ещё нет. В этой кодовой базе разъезд префиксов
// случался четырежды.
func TestDenomMetadataIsOutsideBalances(t *testing.T) {
	require.NotEmpty(t, types.DenomMetadataPrefix)
	require.NotEmpty(t, types.BalancesPrefix)

	key := types.DenomMetadataKey("uartr")
	require.False(t, hasPrefix(key, types.BalancesPrefix),
		"ключ метаданных лежит внутри диапазона балансов")
	require.False(t, hasPrefix(types.BalancesPrefix, types.DenomMetadataPrefix),
		"префикс балансов лежит внутри диапазона метаданных")
	require.False(t, hasPrefix(types.ParamsKey, types.DenomMetadataPrefix),
		"ключ параметров лежит внутри диапазона метаданных")
}

func hasPrefix(key, prefix []byte) bool {
	if len(key) < len(prefix) {
		return false
	}
	for i := range prefix {
		if key[i] != prefix[i] {
			return false
		}
	}
	return true
}

func TestIterateAllDenomMetaData(t *testing.T) {
	a, cleanup, ctx := app.NewAppFromGenesis(nil)
	defer cleanup()
	k := a.GetBankKeeper()

	want := map[string]bool{"ibc/AAA": false, "ibc/BBB": false, "ibc/CCC": false}
	for d := range want {
		k.SetDenomMetaData(ctx, voucher(d))
	}

	var seen int
	k.IterateAllDenomMetaData(ctx, func(md bankTypes.Metadata) bool {
		if _, ok := want[md.Base]; ok {
			want[md.Base] = true
		}
		seen++
		return false
	})

	require.GreaterOrEqual(t, seen, len(want))
	for d, found := range want {
		require.True(t, found, "обход пропустил %s", d)
	}
}

// TestIterateStopsOnRequest: возврат true прерывает обход.
func TestIterateStopsOnRequest(t *testing.T) {
	a, cleanup, ctx := app.NewAppFromGenesis(nil)
	defer cleanup()
	k := a.GetBankKeeper()

	for _, d := range []string{"ibc/AAA", "ibc/BBB", "ibc/CCC"} {
		k.SetDenomMetaData(ctx, voucher(d))
	}

	var seen int
	k.IterateAllDenomMetaData(ctx, func(bankTypes.Metadata) bool {
		seen++
		return true
	})
	require.Equal(t, 1, seen, "обход не остановился по требованию")
}
