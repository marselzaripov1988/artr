//go:build testing
// +build testing

package keeper_test

import (
	"time"

	"github.com/arterynetwork/artr/app"
	"github.com/arterynetwork/artr/x/voting/types"
)

// Назначение обновления через голосование.
//
// Это механизм согласованной остановки: обновление назначают не ради
// самого обновления, а чтобы вся сеть встала на одной высоте. Только
// тогда выгрузка состояния у всех валидаторов получается одинаковой, а
// без этого перенос мейннета на новый код невозможен — узлы разойдутся
// на разных генезисах.
//
// Механизм был сломан. Заявка несла срок временем, а Plan.ValidateBasic
// в SDK с 0.47 отвергает непустое Time и отдельно требует положительной
// высоты. Хуже того, EndProposal записывает ошибку в журнал и идёт
// дальше: предложение оказалось бы в истории принятым, а остановки не
// случилось бы вовсе.
//
// Поэтому проверки смотрят на назначенный план, а не на отсутствие
// ошибки: именно план и был пуст.

// TestSoftwareUpgradeIsScheduled — принятое предложение действительно
// назначает остановку на заданной высоте.
func (s *Suite) TestSoftwareUpgradeIsScheduled() {
	const at = 1_000_000

	_, err := s.app.GetUpgradeKeeper().GetUpgradePlan(s.ctx)
	s.Require().Error(err, "до голосования плана быть не должно")

	s.passUpgradeProposal(at)

	plan, err := s.app.GetUpgradeKeeper().GetUpgradePlan(s.ctx)
	s.Require().NoError(err, "план не назначен — остановки не будет")
	s.EqualValues(at, plan.Height, "остановка назначена не на ту высоту")
	s.True(plan.Time.IsZero(), "время в плане должно оставаться пустым: SDK такой план отвергает")
	s.Equal("3.0.0", plan.Name)
}

// TestSoftwareUpgradeCanBeCancelled — отмена убирает назначенную
// остановку.
//
// Нужна не меньше самой остановки: если дату перенесли, сеть не должна
// встать по прежнему плану.
func (s *Suite) TestSoftwareUpgradeCanBeCancelled() {
	s.passUpgradeProposal(1_000_000)
	_, err := s.app.GetUpgradeKeeper().GetUpgradePlan(s.ctx)
	s.Require().NoError(err)

	s.passProposal(types.Proposal{
		Name:   "cancel the upgrade",
		Author: app.DefaultGenesisUsers["user1"].String(),
		Type:   types.PROPOSAL_TYPE_CANCEL_SOFTWARE_UPGRADE,
	})

	_, err = s.app.GetUpgradeKeeper().GetUpgradePlan(s.ctx)
	s.Error(err, "план остался после отмены")
}

// TestUpgradeByTimeIsRefusedAtTxBoundary — заявку со сроком по времени
// сеть не принимает.
//
// Отказ приходит на границе транзакции: baseapp зовёт ValidateBasic у
// каждого сообщения (validateBasicTxMsgs), а MsgPropose.ValidateBasic
// проверяет саму заявку. Хранитель Propose её не перепроверяет — и это
// нормально, лишь бы отказ случился до голосования.
//
// Важно, что до, а не после. Пройди такая заявка голосование, отказ
// случился бы при исполнении, где его никто не увидит: EndProposal
// пишет ошибку в журнал и идёт дальше, а предложение остаётся в истории
// принятым. Сеть бы решила, что остановка назначена, и не встала.
func (s *Suite) TestUpgradeByTimeIsRefusedAtTxBoundary() {
	at := s.ctx.BlockTime().Add(24 * time.Hour)
	byTime := types.MsgPropose{Proposal: types.Proposal{
		Name:   "upgrade by time",
		Author: app.DefaultGenesisUsers["user1"].String(),
		Type:   types.PROPOSAL_TYPE_SOFTWARE_UPGRADE,
		Args: &types.Proposal_SoftwareUpgrade{
			SoftwareUpgrade: &types.SoftwareUpgradeArgs{Name: "3.0.0", Time: &at},
		},
	}}
	s.Error(byTime.ValidateBasic(), "срок по времени больше не принимается")

	byHeight := types.MsgPropose{Proposal: types.Proposal{
		Name:   "upgrade by height",
		Author: app.DefaultGenesisUsers["user1"].String(),
		Type:   types.PROPOSAL_TYPE_SOFTWARE_UPGRADE,
		Args: &types.Proposal_SoftwareUpgrade{
			SoftwareUpgrade: &types.SoftwareUpgradeArgs{Name: "3.0.0", Height: 1_000_000},
		},
	}}
	s.NoError(byHeight.ValidateBasic(), "высота — нынешний способ, она обязана проходить")
}

// passUpgradeProposal проводит предложение об обновлении через
// голосование целиком.
func (s *Suite) passUpgradeProposal(height int64) {
	s.passProposal(types.Proposal{
		Name:   "upgrade to 3.0.0",
		Author: app.DefaultGenesisUsers["user1"].String(),
		Type:   types.PROPOSAL_TYPE_SOFTWARE_UPGRADE,
		Args: &types.Proposal_SoftwareUpgrade{
			SoftwareUpgrade: &types.SoftwareUpgradeArgs{
				Name:   "3.0.0",
				Height: height,
				Info:   "https://example.com/3.0.0/info.json?checksum=sha256:0000",
			},
		},
	})
}

// passProposal подаёт предложение и добирает голоса до принятия.
//
// Правительство в тестовом генезисе — user1, user2 и user3; автор
// считается согласившимся, поэтому голосуют двое остальных.
func (s *Suite) passProposal(p types.Proposal) {
	s.Require().NoError(s.k.Propose(s.ctx, types.MsgPropose{Proposal: p}))
	s.Require().NoError(s.k.Vote(s.ctx, app.DefaultGenesisUsers["user2"], true))
	s.Require().NoError(s.k.Vote(s.ctx, app.DefaultGenesisUsers["user3"], true))
}
