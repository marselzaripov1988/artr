//go:build testing
// +build testing

package types_test

import (
	"testing"
	"time"

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

// legacyUpgrade — заявка на обновление эпохи 1.1.x: срок задан высотой,
// времени нет. Подать такую сегодня нельзя, но в истории их одиннадцать
// штук из ста пятнадцати.
func legacyUpgrade() types.Proposal {
	return types.Proposal{
		Name:   "1.1.1 patch",
		Author: addr1,
		Type:   types.PROPOSAL_TYPE_SOFTWARE_UPGRADE,
		Args: &types.Proposal_SoftwareUpgrade{
			SoftwareUpgrade: &types.SoftwareUpgradeArgs{
				Name:   "1.1.1",
				Height: 23770,
			},
		},
	}
}

// TestLegacyUpgradeInHistoryAccepted проверяет, что прошлое сети не
// объявляется недействительным сегодняшним правилом.
func TestLegacyUpgradeInHistoryAccepted(t *testing.T) {
	gs := types.GenesisState{
		Params:     validParams(),
		Government: []string{addr1, addr2},
		History: []types.ProposalHistoryRecord{{
			Proposal:   legacyUpgrade(),
			Government: []string{addr1, addr2},
			Agreed:     []string{addr1, addr2},
			Started:    23000,
			Finished:   23100,
		}},
	}

	require.NoError(t, types.ValidateGenesis(gs))
}

// TestLegacyUpgradeRejectedAsNew — правило про будущие заявки сохраняется:
// послабление касается только записей истории.
func TestLegacyUpgradeRejectedAsNew(t *testing.T) {
	require.Error(t, legacyUpgrade().Validate())
	require.NoError(t, legacyUpgrade().ValidateHistorical())
}

// TestUpgradeScheduleIsExactlyOne проверяет структурное требование,
// которое осталось после снятия правила о высоте: срок задан ровно одним
// способом. Заявка без срока вовсе или с двумя сразу — испорченная
// запись, и в истории тоже.
func TestUpgradeScheduleIsExactlyOne(t *testing.T) {
	at := time.Now()

	neither := &types.SoftwareUpgradeArgs{Name: "1.1.1"}
	require.Error(t, neither.ValidateHistorical(), "срок не задан вовсе")

	both := &types.SoftwareUpgradeArgs{Name: "1.1.1", Height: 23770, Time: &at}
	require.Error(t, both.ValidateHistorical(), "срок задан дважды")

	byTime := &types.SoftwareUpgradeArgs{Name: "1.1.1", Time: &at}
	require.NoError(t, byTime.ValidateHistorical())
	require.NoError(t, byTime.Validate(), "нынешний способ должен проходить обе проверки")
}
