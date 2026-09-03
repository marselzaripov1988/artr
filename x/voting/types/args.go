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
func (args *SoftwareUpgradeArgs) Validate() error {
	if err := args.ValidateHistorical(); err != nil {
		return err
	}
	if args.Height > 0 {
		return errors.New("upgrade height is deprecated, use time instead")
	}
	if args.Time == nil {
		return errors.New("upgrade time is nil")
	}
	return nil
}

// ValidateHistorical проверяет заявку на обновление без правил о том, как
// её допустимо подавать сегодня.
//
// Отказ от высоты в пользу времени — правило про будущие заявки, а не
// признак испорченной записи. В истории голосований лежат обновления
// эпохи 1.1.x, когда высота была единственным способом: у них Height
// задан, а Time пуст. Проверять их сегодняшним правилом значит объявлять
// собственное прошлое сети невалидным — на выгрузке мейннета ровно это и
// происходит, одиннадцать записей из ста пятнадцати.
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
