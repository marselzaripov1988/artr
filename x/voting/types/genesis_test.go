//go:build testing
// +build testing

package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	// Разбор адресов идёт через глобальный конфиг SDK, а префикс artr
	// ставится в init() пакета app. Без этого импорта любой адрес сети не
	// проходит разбор, и проверки падают не по той причине, по которой
	// писались.
	_ "github.com/arterynetwork/artr/app"

	"github.com/arterynetwork/artr/x/voting/types"
)

// Адреса из списка government мейннета: проверке нужен разбираемый
// bech32, выдуманный не подойдёт — контрольная сумма не сойдётся.
const (
	addr1 = "artr1pzndpwevc3vmrfwz6uszlggfgcqnm6txnx22cl"
	addr2 = "artr1vrqsp6lf05twm750309lpzng9mk8nq82t5nxkp"
)

func validParams() types.Params {
	return types.Params{VotingPeriod: 24, PollPeriod: 24}
}

// TestEmptyAgreedFromExport проверяет, что выгрузка сети проходит
// собственную проверку.
//
// Проверка «голосов нет, пока нет открытой заявки» была написана как
// сравнение с nil. Экспорт пишет пустой список как [], JSON разбирает его
// в слайс нулевой длины — не nil, — и генезис, выгруженный самой сетью,
// её же validate-genesis отвергал. На выгрузке мейннета это и вылезло.
func TestEmptyAgreedFromExport(t *testing.T) {
	gs := types.GenesisState{
		Params:     validParams(),
		Government: []string{addr1, addr2},
		Agreed:     []string{},
		Disagreed:  []string{},
	}

	require.NoError(t, types.ValidateGenesis(gs))
}

// TestVotesWithoutProposalRejected — обратная сторона: само правило
// должно остаться в силе, непустой список голосов без открытой заявки
// по-прежнему недопустим.
func TestVotesWithoutProposalRejected(t *testing.T) {
	gs := types.GenesisState{
		Params:     validParams(),
		Government: []string{addr1, addr2},
		Agreed:     []string{addr1},
	}
	require.Error(t, types.ValidateGenesis(gs))

	gs.Agreed = nil
	gs.Disagreed = []string{addr1}
	require.Error(t, types.ValidateGenesis(gs))
}
