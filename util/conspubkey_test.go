//go:build testing
// +build testing

package util_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	legacybech32 "github.com/cosmos/cosmos-sdk/types/bech32/legacybech32"

	// Префикс artrvalconspub ставится в init() пакета app.
	_ "github.com/arterynetwork/artr/app"

	"github.com/arterynetwork/artr/util"
)

// TestMatchesLegacyBech32 — страховка на время переезда.
//
// Кодировка консенсусных ключей взята из SDK себе: пакет legacybech32
// помечен устаревшим и однажды исчезнет. Формат при этом менять нельзя —
// им записаны генезис и стор работающей сети.
//
// Пока обе реализации существуют рядом, сверяем их на случайных ключах.
// Когда пакет уберут, эта проверка уйдёт вместе с ним, а привязка к
// известному ключу ниже останется.
func TestMatchesLegacyBech32(t *testing.T) {
	for i := 0; i < 100; i++ {
		pk := ed25519.GenPrivKey().PubKey()

		want, err := legacybech32.MarshalPubKey(legacybech32.ConsPK, pk)
		require.NoError(t, err)

		got, err := util.FormatConsPubKey(pk)
		require.NoError(t, err)
		require.Equal(t, want, got, "кодирование разошлось с legacybech32")

		back, err := util.ParseConsPubKey(want)
		require.NoError(t, err)
		require.True(t, pk.Equals(back), "разбор не вернул исходный ключ")
	}
}

// TestKnownKey закрепляет конкретное значение из тестового генезиса.
//
// Проверка выше сверяет две реализации между собой и переживёт их общую
// ошибку. Эта — привязка к записи, которая уже лежит в сети: если
// кодировка поедет, она заметит это и без второй реализации.
func TestKnownKey(t *testing.T) {
	const known = "artrvalconspub1zcjduepqpme87trszw7awc62ra2de9edwr40v7xy7yfhvpvds96fncagm04qxu308e"

	pk, err := util.ParseConsPubKey(known)
	require.NoError(t, err)
	require.Equal(t, known, util.MustFormatConsPubKey(pk), "круговой путь изменил ключ")
}

// TestRejectsGarbage: неверная строка должна давать ошибку, а не ключ.
func TestRejectsGarbage(t *testing.T) {
	for _, bad := range []string{
		"",
		"not-a-key",
		// Верный bech32, но чужой префикс.
		"artr1d4ezqdj03uachct8hum0z9zlfftzdq2f6yzvhj",
		// Верный префикс, испорченная контрольная сумма.
		"artrvalconspub1zcjduepqpme87trszw7awc62ra2de9edwr40v7xy7yfhvpvds96fncagm04qxu308f",
	} {
		_, err := util.ParseConsPubKey(bad)
		require.Error(t, err, "принят негодный ключ: %q", bad)
	}
}
