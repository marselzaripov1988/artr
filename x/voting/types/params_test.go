//go:build testing
// +build testing

package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/arterynetwork/artr/x/voting/types"
)

// Срок голосования по умолчанию измеряется в часах.
//
// Проверка нужна потому, что единицу тут уже теряли: в константу было
// подставлено число блоков (util.BlocksOneDay = 2880), а Keeper.Propose
// умножает это значение на time.Hour. Сутки превращались в 120 дней, и
// заметить это можно было, только не дождавшись конца голосования.
//
// Сеть, поднятая из генезиса без параметров voting, оказалась бы
// неуправляемой: предложение об остановке — тот самый механизм переноса —
// висело бы четыре месяца.

// TestDefaultVotingPeriodIsOneDay — умолчание даёт ровно сутки.
//
// Считается ровно так же, как в Keeper.Propose: значение умножается на
// time.Hour. Если в константу опять попадёт число блоков, арифметика
// разойдётся здесь, а не в работающей сети.
//
// Сутки — не произвольная величина: столько стоит в мейннете (24 в
// выгрузке состояния), и на это опирается порядок переноса — предложение
// об остановке подаётся за сутки до назначенной высоты.
func TestDefaultVotingPeriodIsOneDay(t *testing.T) {
	p := types.DefaultParams()

	require.Equal(t, 24*time.Hour, time.Duration(p.VotingPeriod)*time.Hour,
		"срок голосования по умолчанию должен быть сутки")
	require.Equal(t, 24*time.Hour, time.Duration(p.PollPeriod)*time.Hour,
		"срок опроса по умолчанию должен быть сутки")
}

// TestDefaultParamsAreValid — умолчание проходит собственную проверку.
//
// Params.Validate зовётся при разборе генезиса. Умолчание, которое её не
// проходит, роняло бы сеть на старте.
func TestDefaultParamsAreValid(t *testing.T) {
	require.NoError(t, types.DefaultParams().Validate())
}
