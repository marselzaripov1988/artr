package app

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// Ответ на запрос параметров стейкинга — ровно один, и только ради
// совместимости.
//
// Модуля x/staking у Artery нет: валидаторами ведает собственный
// x/noding. Но релееры, обозреватели и кошельки спрашивают период
// развязки по стандартному пути cosmos.staking.v1beta1.Query/Params, и
// без ответа Artery для них немая. Hermes на этом останавливается ещё до
// создания клиента: "unknown query path".
//
// Остальные методы отвечают Unimplemented честно — делегаций в понимании
// SDK у Artery нет, и выдумывать их вредно: лучше явный отказ, чем
// правдоподобно выглядящая пустота, по которой чужой код сделает неверный
// вывод.

// UnbondingTime — период развязки, который Artery объявляет наружу.
//
// Взят коротким сознательно, и это стоит объяснить.
//
// В модели IBC доверительный период светлого клиента выводится из периода
// развязки: заголовок принимается, пока он моложе, потому что внутри
// этого окна валидатора, подписавшего ложный заголовок, можно наказать
// деньгами.
//
// У Artery такого наказания нет. MarkByzantine записывает нарушение и на
// втором выносит пожизненный бан — право валидировать закрывается, но
// средства не трогаются. Значит экономического сдерживателя, на который
// рассчитывает IBC, не существует.
//
// Объявить здесь 21 день — период отзыва делегации — значило бы обещать
// контрагентам защиту, которой нет. Трое суток обещают втрое меньше:
// доверительный период выйдет двое, релееру придётся обновлять клиента
// чаще, зато окно, в котором отсутствие наказания имеет значение,
// соответственно уже.
//
// Настоящее решение — добавить сжигание доли делегации за византийское
// поведение в x/noding. Это правка протокола, консенсусная, и до неё
// значение здесь остаётся заниженным намеренно.
const UnbondingTime = 3 * 24 * time.Hour

type stakingShim struct{}

func (stakingShim) Params(_ context.Context, _ *stakingTypes.QueryParamsRequest) (*stakingTypes.QueryParamsResponse, error) {
	params := stakingTypes.DefaultParams()
	params.UnbondingTime = UnbondingTime
	params.BondDenom = "uartr"
	return &stakingTypes.QueryParamsResponse{Params: params}, nil
}

func unsupported(what string) error {
	return status.Errorf(codes.Unimplemented,
		"%s: Artery не использует x/staking, валидаторами ведает x/noding", what)
}

func (stakingShim) Validators(context.Context, *stakingTypes.QueryValidatorsRequest) (*stakingTypes.QueryValidatorsResponse, error) {
	return nil, unsupported("validators")
}

func (stakingShim) Validator(context.Context, *stakingTypes.QueryValidatorRequest) (*stakingTypes.QueryValidatorResponse, error) {
	return nil, unsupported("validator")
}

func (stakingShim) ValidatorDelegations(context.Context, *stakingTypes.QueryValidatorDelegationsRequest) (*stakingTypes.QueryValidatorDelegationsResponse, error) {
	return nil, unsupported("validator delegations")
}

func (stakingShim) ValidatorUnbondingDelegations(context.Context, *stakingTypes.QueryValidatorUnbondingDelegationsRequest) (*stakingTypes.QueryValidatorUnbondingDelegationsResponse, error) {
	return nil, unsupported("validator unbonding delegations")
}

func (stakingShim) Delegation(context.Context, *stakingTypes.QueryDelegationRequest) (*stakingTypes.QueryDelegationResponse, error) {
	return nil, unsupported("delegation")
}

func (stakingShim) UnbondingDelegation(context.Context, *stakingTypes.QueryUnbondingDelegationRequest) (*stakingTypes.QueryUnbondingDelegationResponse, error) {
	return nil, unsupported("unbonding delegation")
}

func (stakingShim) DelegatorDelegations(context.Context, *stakingTypes.QueryDelegatorDelegationsRequest) (*stakingTypes.QueryDelegatorDelegationsResponse, error) {
	return nil, unsupported("delegator delegations")
}

func (stakingShim) DelegatorUnbondingDelegations(context.Context, *stakingTypes.QueryDelegatorUnbondingDelegationsRequest) (*stakingTypes.QueryDelegatorUnbondingDelegationsResponse, error) {
	return nil, unsupported("delegator unbonding delegations")
}

func (stakingShim) Redelegations(context.Context, *stakingTypes.QueryRedelegationsRequest) (*stakingTypes.QueryRedelegationsResponse, error) {
	return nil, unsupported("redelegations")
}

func (stakingShim) DelegatorValidators(context.Context, *stakingTypes.QueryDelegatorValidatorsRequest) (*stakingTypes.QueryDelegatorValidatorsResponse, error) {
	return nil, unsupported("delegator validators")
}

func (stakingShim) DelegatorValidator(context.Context, *stakingTypes.QueryDelegatorValidatorRequest) (*stakingTypes.QueryDelegatorValidatorResponse, error) {
	return nil, unsupported("delegator validator")
}

func (stakingShim) HistoricalInfo(context.Context, *stakingTypes.QueryHistoricalInfoRequest) (*stakingTypes.QueryHistoricalInfoResponse, error) {
	return nil, unsupported("historical info")
}

func (stakingShim) Pool(context.Context, *stakingTypes.QueryPoolRequest) (*stakingTypes.QueryPoolResponse, error) {
	return nil, unsupported("pool")
}
