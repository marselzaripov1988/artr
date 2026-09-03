//go:build testing
// +build testing

package schedule_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/arterynetwork/artr/app"
	"github.com/arterynetwork/artr/util"
	"github.com/arterynetwork/artr/x/schedule/keeper"
	"github.com/arterynetwork/artr/x/schedule/types"
)

func TestScheduleGenesis(t *testing.T) {
	suite.Run(t, new(Suite))
}

type Suite struct {
	suite.Suite

	app     *app.ArteryApp
	cleanup func()
	ctx     sdk.Context
	k       keeper.Keeper

	bbHeader abci.RequestBeginBlock
}

func (s *Suite) SetupTest() {
	defer func() {
		if e := recover(); e != nil {
			s.FailNow("panic on setup", e)
		}
	}()
	s.app, s.cleanup, s.ctx = app.NewAppFromGenesis(nil)
	s.k = s.app.GetScheduleKeeper()

	s.bbHeader = abci.RequestBeginBlock{
		Header: tmproto.Header{
			ProposerAddress: util.MustUnmarshalConsPubKey(app.DefaultUser1ConsPubKey).Address().Bytes(),
		},
	}
}

func (s *Suite) TearDownTest() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

func (s Suite) TestCleanGenesis() {
	s.checkExportImport()
}

// TestParamsSurviveExport проверяет, что параметры переживают круговой
// экспорт-импорт. С переносом параметров из подпространства x/params они
// лежат в сторе модуля рядом с задачами, поэтому путь стоит проверять.
func (s *Suite) TestParamsSurviveExport() {
	params := s.k.GetParams(s.ctx)
	s.NotZero(params.DayNanos, "день не должен быть нулевым")

	s.checkExportImport()

	s.Equal(params, s.k.GetParams(s.ctx), "параметры после выгрузки")
}

// TestParamsKeyIsOutsideTaskRange проверяет само правило, а не его
// следствия: ключ параметров не должен попадать в диапазон, по которому
// идут обходы задач.
//
// Проверка структурная и потому надёжнее поведенческих: она падает сразу,
// как только ключи разъезжаются, независимо от того, проявилось это уже
// на каких-то данных или ещё нет.
func (s Suite) TestParamsKeyIsOutsideTaskRange() {
	s.Require().NotEmpty(types.TaskPrefix, "префикс задач не задан")
	s.Require().NotEmpty(types.ParamsKey, "ключ параметров не задан")
	s.False(
		bytes.HasPrefix(types.ParamsKey, types.TaskPrefix),
		"ключ параметров лежит внутри диапазона задач — обходы расписания будут его цеплять",
	)
}

// TestParamsAreNotATask ловит грубый случай: запись параметров попала в
// выборку задач и видна там как задача с пустым обработчиком.
//
// Проверка слабее структурной выше и сама по себе достаточной не является:
// при некоторых способах разъезда ключей запись просто не разбирается в
// задачу и в выборку не попадает вовсе. Оставлена как дополнительная —
// она читаема и покрывает наиболее заметное проявление.
func (s *Suite) TestParamsAreNotATask() {
	// Диапазон заведомо шире любого разумного расписания.
	from := s.ctx.BlockTime().Add(-1000 * 24 * time.Hour)
	to := s.ctx.BlockTime().Add(1000 * 24 * time.Hour)

	for _, task := range s.k.GetTasks(s.ctx, from, to) {
		s.NotEmpty(task.HandlerName, "задача без обработчика — похоже, в выборку попали параметры")
	}

	_, tasks := s.k.ExportGenesis(s.ctx)
	for _, task := range tasks {
		s.NotEmpty(task.HandlerName, "в выгрузку генезиса попала запись без обработчика")
	}
}

// TestScheduleAndFire проверяет полный цикл: задача планируется, находится
// обходом диапазона, переживает выгрузку и срабатывает в свой срок.
func (s *Suite) TestScheduleAndFire() {
	const event = "test/schedule-and-fire"

	var fired []string
	s.k.AddHook(event, func(_ sdk.Context, data []byte, _ time.Time) {
		fired = append(fired, string(data))
	})

	// Планируем на два блока вперёд: блок в тесте — тридцать секунд.
	at := s.ctx.BlockTime().Add(time.Minute)
	s.k.ScheduleTask(s.ctx, at, event, []byte("payload"))

	found := s.tasksOf(event, at.Add(-time.Hour), at.Add(time.Hour))
	s.Require().Len(found, 1, "запланированная задача не найдена обходом диапазона")
	s.Equal([]byte("payload"), found[0].Data)

	s.Empty(fired, "задача сработала раньше срока")

	for i := 0; i < 3; i++ {
		s.nextBlock()
	}

	s.Equal([]string{"payload"}, fired, "задача не сработала в свой срок")
	s.Empty(s.tasksOf(event, at.Add(-time.Hour), at.Add(time.Hour)), "выполненная задача осталась в сторе")
}

// TestScheduledTaskSurvivesExport проверяет, что незавершённая задача
// переживает выгрузку и обратный импорт.
func (s *Suite) TestScheduledTaskSurvivesExport() {
	const event = "test/survives-export"

	at := s.ctx.BlockTime().Add(24 * time.Hour)
	s.k.ScheduleTask(s.ctx, at, event, []byte("payload"))

	_, tasks := s.k.ExportGenesis(s.ctx)
	var found *types.Task
	for i := range tasks {
		if tasks[i].HandlerName == event {
			found = &tasks[i]
			break
		}
	}
	s.Require().NotNil(found, "задача не попала в выгрузку генезиса")
	s.Equal([]byte("payload"), found.Data)
	s.True(found.Time.Equal(at), "время задачи изменилось при выгрузке")

	s.checkExportImport()
}

// TestDeleteTask проверяет удаление: после него задача не должна ни
// находиться обходом, ни срабатывать.
func (s *Suite) TestDeleteTask() {
	const event = "test/delete"

	var fired int
	s.k.AddHook(event, func(_ sdk.Context, _ []byte, _ time.Time) { fired++ })

	at := s.ctx.BlockTime().Add(time.Minute)
	s.k.ScheduleTask(s.ctx, at, event, []byte("payload"))
	s.Require().Len(s.tasksOf(event, at.Add(-time.Hour), at.Add(time.Hour)), 1)

	s.k.Delete(s.ctx, at, event, []byte("payload"))
	s.Empty(s.tasksOf(event, at.Add(-time.Hour), at.Add(time.Hour)), "удалённая задача осталась")

	for i := 0; i < 3; i++ {
		s.nextBlock()
	}
	s.Zero(fired, "удалённая задача всё равно сработала")
}

// tasksOf возвращает задачи указанного обработчика в диапазоне. В сторе
// лежат и задачи, поставленные самим приложением при инициализации, поэтому
// выборку нужно фильтровать.
func (s Suite) tasksOf(event string, since, to time.Time) []types.Task {
	var result []types.Task
	for _, task := range s.k.GetTasks(s.ctx, since, to) {
		if task.HandlerName == event {
			result = append(result, task)
		}
	}
	return result
}

func (s *Suite) nextBlock() (abci.ResponseEndBlock, abci.ResponseBeginBlock) {
	ebr := s.app.EndBlocker(s.ctx, abci.RequestEndBlock{})
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1).WithBlockTime(s.ctx.BlockTime().Add(30 * time.Second))
	bbr := s.app.BeginBlocker(s.ctx, s.bbHeader)
	return ebr, bbr
}

func (s Suite) checkExportImport() {
	s.app.CheckExportImport(s.T(),
		s.ctx.BlockTime(),
		[]string{
			types.StoreKey,
		},
		map[string]app.Decoder{
			types.StoreKey: app.DummyDecoder,
		},
		map[string]app.Decoder{
			types.StoreKey: app.DummyDecoder,
		},
		make(map[string][][]byte, 0),
	)
}
