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

// upgradeByHeight — заявка эпохи 1.1.x и снова нынешняя: срок задан
// высотой. Таких в истории мейннета одиннадцать из ста пятнадцати.
func upgradeByHeight() types.Proposal {
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

// upgradeByTime — заявка эпохи 2.x: срок задан временем. Подать такую
// сегодня нельзя, но в истории их большинство.
func upgradeByTime() types.Proposal {
	at := time.Date(2023, 1, 1, 3, 0, 0, 0, time.UTC)
	return types.Proposal{
		Name:   "2.4.0",
		Author: addr1,
		Type:   types.PROPOSAL_TYPE_SOFTWARE_UPGRADE,
		Args: &types.Proposal_SoftwareUpgrade{
			SoftwareUpgrade: &types.SoftwareUpgradeArgs{
				Name: "2.4.0",
				Time: &at,
			},
		},
	}
}

// TestBothUpgradeFormsAcceptedInHistory — прошлое сети остаётся
// читаемым, каким бы способом там ни назначали срок.
//
// Способ менялся дважды: сначала высота, потом время, теперь снова
// высота. Выгрузка мейннета содержит обе разновидности, и объявлять
// половину истории недействительной нельзя.
func TestBothUpgradeFormsAcceptedInHistory(t *testing.T) {
	record := func(p types.Proposal) types.ProposalHistoryRecord {
		return types.ProposalHistoryRecord{
			Proposal:   p,
			Government: []string{addr1, addr2},
			Agreed:     []string{addr1, addr2},
			Started:    23000,
			Finished:   23100,
		}
	}
	gs := types.GenesisState{
		Params:     validParams(),
		Government: []string{addr1, addr2},
		History:    []types.ProposalHistoryRecord{record(upgradeByHeight()), record(upgradeByTime())},
	}

	require.NoError(t, types.ValidateGenesis(gs))
}

// TestUpgradeByTimeRejectedAsNew — сегодня срок задаётся высотой.
//
// Не вкусовщина: Plan.ValidateBasic в SDK отвергает непустое Time, и
// заявка со временем прошла бы голосование, а потом упала при
// исполнении. Ловить это надо при подаче.
func TestUpgradeByTimeRejectedAsNew(t *testing.T) {
	require.Error(t, upgradeByTime().Validate(), "время как срок больше не принимается")
	require.NoError(t, upgradeByTime().ValidateHistorical(), "но в истории остаётся законным")

	require.NoError(t, upgradeByHeight().Validate(), "высота — нынешний способ")
}

// TestUpgradeScheduleIsExactlyOne — структурное требование, общее для
// прошлого и настоящего: срок задан ровно одним способом. Ни одного или
// сразу два — испорченная запись, и в истории тоже.
func TestUpgradeScheduleIsExactlyOne(t *testing.T) {
	at := time.Now()

	neither := &types.SoftwareUpgradeArgs{Name: "1.1.1"}
	require.Error(t, neither.ValidateHistorical(), "срок не задан вовсе")

	both := &types.SoftwareUpgradeArgs{Name: "1.1.1", Height: 23770, Time: &at}
	require.Error(t, both.ValidateHistorical(), "срок задан дважды")

	byHeight := &types.SoftwareUpgradeArgs{Name: "1.1.1", Height: 23770}
	require.NoError(t, byHeight.ValidateHistorical())
	require.NoError(t, byHeight.Validate(), "нынешний способ проходит обе проверки")

	require.Error(t, (&types.SoftwareUpgradeArgs{Height: 23770}).ValidateHistorical(), "имя обязательно")
}
