//go:build testing
// +build testing

package keeper_test

import (
	"cosmossdk.io/math"

	"github.com/arterynetwork/artr/util"
	"github.com/arterynetwork/artr/x/delegating/keeper"
)

// Начисление на счёте без профиля.
//
// GetProfile возвращает nil, когда записи нет, а IsActive объявлен со
// значимым получателем: вызов на nil разыменовывает пустой указатель.
// Паника уходила в x/schedule, тот её перехватывал и выбрасывал задачу.
//
// Наружу это выглядело так: делегирование прошло, задача в расписании
// появилась, а через сутки расписание пусто и следующая задача не
// заведена. Начисление для счёта умирало навсегда, и весь след — одна
// строка в журнале узла. Нашлось на тестнете, на счёте крана: он
// заводится правкой генезиса, в балансах и реферальном дереве есть, а
// профиля у него нет.
//
// В мейннете таких счетов ноль — все 134 519 делегирующих имеют профиль,
// потому что счета создаются через MsgCreateAccount. Но обходной путь
// существует, и цена ошибки на нём непомерна для её причины.
//
// Проверяется не выплата: счёт без профиля денег и не получает, так
// задумано — accrue выходит с записью в журнал. Проверяется, что
// обработчик доживает до конца и планирует следующую задачу.

// TestAccrueRescheduledWithoutProfile — задача переживает отсутствие
// профиля и планируется на следующие сутки.
func (s *Suite) TestAccrueRescheduledWithoutProfile() {
	user := keeper.DefaultGenesisUsers["user4"]

	// Счёт, какой бывает при заведении в обход обычного порядка.
	s.app.GetProfileKeeper().DeleteProfile(s.ctx, user)
	s.Require().Nil(s.app.GetProfileKeeper().GetProfile(s.ctx, user), "профиль должен отсутствовать")

	s.Require().NoError(s.k.Delegate(s.ctx, user, math.NewInt(1_000_000000)))

	first := s.k.Get(s.ctx, user).NextAccrue
	s.Require().NotNil(first, "после делегирования срок начисления должен быть задан")

	// Сутки: ровно на них и запланировано.
	for t := 0; t < util.BlocksOneDay; t++ {
		s.nextBlock()
	}

	second := s.k.Get(s.ctx, user).NextAccrue
	s.Require().NotNil(second, "срок начисления обнулён — начисление отменено")
	s.True(second.After(*first),
		"срок не сдвинулся (%s), значит обработчик не дошёл до конца и задача потеряна", first)
}

// TestAccrueKeepsGoingWithoutProfile — и на вторые сутки тоже.
//
// Отдельно, потому что одного сдвига мало: задача должна не просто
// перепланироваться, а продолжать это делать.
func (s *Suite) TestAccrueKeepsGoingWithoutProfile() {
	user := keeper.DefaultGenesisUsers["user4"]

	s.app.GetProfileKeeper().DeleteProfile(s.ctx, user)
	s.Require().NoError(s.k.Delegate(s.ctx, user, math.NewInt(1_000_000000)))

	var t int
	for ; t < util.BlocksOneDay; t++ {
		s.nextBlock()
	}
	afterFirstDay := *s.k.Get(s.ctx, user).NextAccrue

	for ; t < 2*util.BlocksOneDay; t++ {
		s.nextBlock()
	}
	afterSecondDay := s.k.Get(s.ctx, user).NextAccrue

	s.Require().NotNil(afterSecondDay)
	s.True(afterSecondDay.After(afterFirstDay),
		"вторых суток не случилось: %s -> %s", afterFirstDay, afterSecondDay)
}

// TestAccrueStillPaysWithProfile — сторож: правка не должна была
// изменить поведение обычного счёта.
//
// Без него легко «починить» падение, заодно сломав выплаты тем, у кого с
// профилем всё в порядке.
func (s *Suite) TestAccrueStillPaysWithProfile() {
	user := keeper.DefaultGenesisUsers["user4"]
	s.Require().NotNil(s.app.GetProfileKeeper().GetProfile(s.ctx, user), "у подопытного должен быть профиль")

	s.Require().NoError(s.k.Delegate(s.ctx, user, math.NewInt(1_000_000000)))
	before := s.bk.GetBalance(s.ctx, user).AmountOf(util.ConfigMainDenom)

	for t := 0; t < util.BlocksOneDay; t++ {
		s.nextBlock()
	}

	after := s.bk.GetBalance(s.ctx, user).AmountOf(util.ConfigMainDenom)
	s.True(after.GT(before), "начисление с профилем перестало приходить: %s -> %s", before, after)
}
