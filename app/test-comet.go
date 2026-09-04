//go:build testing
// +build testing

package app

import (
	"time"

	"cosmossdk.io/core/comet"

	abci "github.com/cometbft/cometbft/abci/types"
)

// TestCometInfo — минимальная реализация comet.BlockInfo для тестов.
//
// В ABCI 2.0 свидетельства о нарушениях приходят обработчику не
// аргументом, а через контекст, и класть их туда надо этим интерфейсом.
// Своя реализация нужна потому, что у baseapp она есть, но не вывезена
// наружу — тип неэкспортируемый.
//
// Заполнены только свидетельства: голоса ставятся отдельно, через
// WithVoteInfos, а хеш валидаторов и последний коммит x/noding не читает.
type TestCometInfo struct {
	Proposer    []byte
	Misbehavior []abci.Misbehavior
}

func (i TestCometInfo) GetEvidence() comet.EvidenceList { return testEvidence(i.Misbehavior) }
func (i TestCometInfo) GetValidatorsHash() []byte       { return nil }
func (i TestCometInfo) GetProposerAddress() []byte      { return i.Proposer }
func (i TestCometInfo) GetLastCommit() comet.CommitInfo { return testCommit{} }

type testEvidence []abci.Misbehavior

func (e testEvidence) Len() int                 { return len(e) }
func (e testEvidence) Get(i int) comet.Evidence { return testMisbehaviour{m: e[i]} }

// Обёртка, а не производный тип: у abci.Misbehavior поля названы так же,
// как методы интерфейса, и Go такого совмещения не допускает.
type testMisbehaviour struct{ m abci.Misbehavior }

func (t testMisbehaviour) Type() comet.MisbehaviorType { return comet.MisbehaviorType(t.m.Type) }
func (t testMisbehaviour) Height() int64               { return t.m.Height }
func (t testMisbehaviour) Time() time.Time             { return t.m.Time }
func (t testMisbehaviour) TotalVotingPower() int64     { return t.m.TotalVotingPower }
func (t testMisbehaviour) Validator() comet.Validator  { return testValidator{v: t.m.Validator} }

type testValidator struct{ v abci.Validator }

func (t testValidator) Address() []byte { return t.v.Address }
func (t testValidator) Power() int64    { return t.v.Power }

type testCommit struct{}

func (testCommit) Round() int32           { return 0 }
func (testCommit) Votes() comet.VoteInfos { return testVotes{} }

type testVotes struct{}

func (testVotes) Len() int               { return 0 }
func (testVotes) Get(int) comet.VoteInfo { return nil }
