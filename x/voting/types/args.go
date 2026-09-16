package types

import (
	"github.com/pkg/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (args *PriceArgs) Validate() error           { return nil }
func (args *DelegationAwardArgs) Validate() error { return args.Award.Validate() }
func (args *NetworkAwardArgs) Validate() error    { return args.Award.Validate() }
func (args *AddressArgs) Validate() error {
	_, err := sdk.AccAddressFromBech32(args.Address)
	return err
}
// Validate — правило для заявок, подаваемых сегодня: срок задаётся
// высотой.
//
// Раньше здесь было ровно обратное: высота объявлена устаревшей, время
// обязательным. История развернулась. SDK начиная с 0.47 отвергает план
// со временем — Plan.ValidateBasic отвечает "time-based upgrades have
// been deprecated in the SDK" и отдельно требует положительной высоты.
// Заявка со временем прошла бы голосование и упала при исполнении, а
// именно исполнение и нужно: обновление назначают, чтобы сеть встала
// одновременно у всех.
//
// Высота для согласованной остановки и лучше по существу. Время у узлов
// своё, и «остановиться в полночь» каждый понимает чуть по-своему;
// высота одна на всех, и выгрузка после неё у всех получается
// одинаковой. Ради одинаковых выгрузок остановка и затевается.
func (args *SoftwareUpgradeArgs) Validate() error {
	if err := args.ValidateHistorical(); err != nil {
		return err
	}
	if args.Time != nil {
		return errors.New("upgrade time is deprecated, use height instead")
	}
	if args.Height <= 0 {
		return errors.New("upgrade height must be positive")
	}
	return nil
}

// ValidateHistorical проверяет заявку на обновление без правил о том, как
// её допустимо подавать сегодня.
//
// Какой из двух способов считается нынешним — правило про новые заявки, а
// не признак испорченной записи. В истории голосований лежат обновления
// эпохи 1.1.x с высотой и более поздние со временем; сегодняшнее правило
// снова требует высоты. Проверять прошлое сети текущим правилом значит
// объявлять её собственную историю невалидной — на выгрузке мейннета
// ровно это и происходило, одиннадцать записей из ста пятнадцати.
//
// Поэтому здесь остаётся только структурное требование: срок обновления
// задан ровно одним из двух способов.
func (args *SoftwareUpgradeArgs) ValidateHistorical() error {
	if args.Name == "" {
		return errors.New("empty upgrade name")
	}
	if (args.Height > 0) == (args.Time != nil) {
		return errors.New("upgrade must be scheduled either by height or by time")
	}
	return nil
}
func (args *MinAmountArgs) Validate() error   { return nil }
func (args *CountArgs) Validate() error       { return nil }
func (args *StatusArgs) Validate() error      { return args.Status.Validate() }
func (args *MinCriteriaArgs) Validate() error { return args.MinCriteria.Validate() }
func (args *PeriodArgs) Validate() error {
	if args.Days < 1 {
		return errors.New("period must be at least one day")
	}
	return nil
}
func (args *RevokeArgs) Validate() error { return args.Revoke.Validate() }

func (args *AddressArgs) GetAddress() sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(args.Address)
	if err != nil {
		panic(err)
	}
	return addr
}
