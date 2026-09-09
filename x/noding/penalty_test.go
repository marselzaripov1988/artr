//go:build testing
// +build testing

package noding_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/arterynetwork/artr/app"
	"github.com/arterynetwork/artr/util"
	"github.com/arterynetwork/artr/x/noding/types"
)

// Изъятие за византийское поведение.
//
// До появления этих проверок сеть на двойную подпись отвечала только
// пожизненным баном: нарушение записывалось, право валидировать
// закрывалось, но средства не трогались. Экономического сдерживателя, на
// который рассчитывает светлый клиент IBC, не существовало вовсе.

// TestPenaltyTakesShareOfSelfStake — изъятие берёт ровно долю
// собственной делегации нарушителя.
func (s *Suite) TestPenaltyTakesShareOfSelfStake() {
	bk := s.app.GetBankKeeper()
	user2 := app.DefaultGenesisUsers["user2"]

	before := bk.GetBalance(s.ctx, user2).AmountOf(util.ConfigDelegatedDenom)
	s.Require().True(before.IsPositive(), "у нарушителя должна быть собственная делегация")

	share := s.k.GetParams(s.ctx).MisbehaviourPenalty
	s.Require().False(share.IsNullValue(), "доля изъятия не задана в генезисе")
	want := share.MulInt64(before.Int64()).Int64()
	s.Require().Positive(want, "доля от делегации округлилась в ноль — проверять нечего")

	s.markByzantine(user2)

	after := bk.GetBalance(s.ctx, user2).AmountOf(util.ConfigDelegatedDenom)
	s.Equal(before.Int64()-want, after.Int64(), "изъято не столько, сколько положено")
}

// TestPenaltyGoesToModuleNotBurned — изъятое переходит на счёт модуля, а
// не исчезает.
//
// Это и есть отличие изъятия от сжигания, и проверять его надо отдельно:
// баланс нарушителя уменьшается одинаково в обоих случаях.
func (s *Suite) TestPenaltyGoesToModuleNotBurned() {
	bk := s.app.GetBankKeeper()
	ak := s.app.GetAccountKeeper()
	user2 := app.DefaultGenesisUsers["user2"]

	treasury := ak.GetModuleAddress(types.ModuleName)
	s.Require().NotNil(treasury, "счёт модуля не заведён в правах")

	supplyBefore := bk.GetSupply(s.ctx).GetTotal().AmountOf(util.ConfigDelegatedDenom)
	treasuryBefore := bk.GetBalance(s.ctx, treasury).AmountOf(util.ConfigDelegatedDenom)
	stakeBefore := bk.GetBalance(s.ctx, user2).AmountOf(util.ConfigDelegatedDenom)

	s.markByzantine(user2)

	taken := stakeBefore.Sub(bk.GetBalance(s.ctx, user2).AmountOf(util.ConfigDelegatedDenom))
	s.Require().True(taken.IsPositive(), "ничего не изъято")

	treasuryAfter := bk.GetBalance(s.ctx, treasury).AmountOf(util.ConfigDelegatedDenom)
	s.Equal(treasuryBefore.Add(taken).Int64(), treasuryAfter.Int64(),
		"на счёт модуля пришло не то, что изъято")

	supplyAfter := bk.GetSupply(s.ctx).GetTotal().AmountOf(util.ConfigDelegatedDenom)
	s.Equal(supplyBefore.Int64(), supplyAfter.Int64(),
		"предложение монет изменилось — значит сожгли, а не изъяли")
}

// TestPenaltyIsReportedInEvent — изъятое видно снаружи.
//
// Без этого наблюдатель не отличит наказанного нарушителя от просто
// отмеченного.
func (s *Suite) TestPenaltyIsReportedInEvent() {
	user2 := app.DefaultGenesisUsers["user2"]

	// События блока берём из возвращаемого значения, а не из контекста:
	// менеджер модулей заводит для блока отдельный сборщик, и в контексте
	// вызывающего они не оседают.
	_, bb := s.markByzantine(user2)

	var found bool
	for _, e := range bb.Events {
		// Тип короткий, без пространства имён: так их именует
		// util.EmitEvent.
		if e.Type != "byzantine" {
			continue
		}
		for _, a := range e.Attributes {
			if a.Key == "penalty" && a.Value != "[]" && a.Value != "null" {
				found = true
			}
		}
	}
	s.True(found, "в событии о нарушении нет изъятой суммы")
}

// markByzantine проводит блок со свидетельством о двойной подписи
// указанного счёта.
func (s *Suite) markByzantine(acc sdk.AccAddress) (sdk.EndBlock, sdk.BeginBlock) {
	s.T().Helper()

	_, pub, cons := app.NewTestConsPubAddress()
	s.Require().NoError(s.k.SwitchOn(s.ctx, acc, pub))

	proposer := util.MustParseConsPubKey(app.DefaultUser1ConsPubKey)
	val := abci.Validator{Address: cons, Power: 10}

	return s.nextBlock(
		proposer,
		[]abci.VoteInfo{{Validator: val, BlockIdFlag: tmproto.BlockIDFlagCommit}},
		[]abci.Misbehavior{{
			Type:      abci.MisbehaviorType_DUPLICATE_VOTE,
			Validator: val,
			Height:    s.ctx.BlockHeight(),
		}},
	)
}
