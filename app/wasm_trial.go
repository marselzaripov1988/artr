package app

// Переходники для пробного подключения CosmWasm.
//
// ЭТО ПРОБА, а не готовая интеграция. Файл живёт только в отдельной копии
// дерева на стенде и нужен, чтобы выяснить цену вопроса: что именно
// требует x/wasm от сети, у которой свой банк и нет стейкинга вовсе.
//
// Заглушки возвращают ошибку, а не nil. Контракт, спросивший у сети
// список валидаторов, получит внятный отказ, а не падение узла: nil здесь
// разыменовался бы внутри wasmd, и виноватым выглядел бы не контракт.

import (
	"context"
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrTypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	abci "github.com/cometbft/cometbft/abci/types"
)

// errNoStaking — общий ответ на всё, чего в Artery нет.
//
// Формулировка обращена к автору контракта: он должен понять, что
// ошибся сетью, а не что сеть сломалась.
var errNoStaking = errors.New("artery has no staking module: validator set is governed by x/noding")

// --- Банк ---

// wasmBank дополняет bankAdapter тем, чего требует именно x/wasm.
//
// Остальное (SendCoins, BurnCoins, IsSendEnabledCoins, BlockedAddr,
// SendCoinsFromAccountToModule, GetAllBalances) уже написано для IBC и
// наследуется как есть.
type wasmBank struct {
	bankAdapter
}

// GetBalance — у Artery метод с тем же именем отдаёт ВСЕ монеты счёта, а
// не одну. Одноимённость обманчива, поэтому денóм выбираем здесь.
func (b wasmBank) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	coins := b.k.GetBalance(sdk.UnwrapSDKContext(ctx), addr)
	return sdk.NewCoin(denom, coins.AmountOf(denom))
}

func (b wasmBank) GetSupply(ctx context.Context, denom string) sdk.Coin {
	// Supply у Artery — структура со списком Total, а не sdk.Coins.
	total := sdk.Coins(b.k.GetSupply(sdk.UnwrapSDKContext(ctx)).Total)
	return sdk.NewCoin(denom, total.AmountOf(denom))
}

func (b wasmBank) GetDenomMetaData(ctx context.Context, denom string) (bankTypes.Metadata, bool) {
	return b.k.GetDenomMetaData(sdk.UnwrapSDKContext(ctx), denom)
}

// DenomsMetadata — списком, как того ждёт gRPC-запрос из контракта.
//
// Постраничности нет намеренно: денóмов у Artery три, и заводить ради них
// курсор незачем.
func (b wasmBank) DenomsMetadata(
	ctx context.Context, _ *bankTypes.QueryDenomsMetadataRequest,
) (*bankTypes.QueryDenomsMetadataResponse, error) {
	var out []bankTypes.Metadata
	b.k.IterateAllDenomMetaData(sdk.UnwrapSDKContext(ctx), func(md bankTypes.Metadata) bool {
		out = append(out, md)
		return false
	})
	return &bankTypes.QueryDenomsMetadataResponse{Metadatas: out}, nil
}

// IsSendEnabledCoin — то же, что IsSendEnabledCoins, но для одной
// монеты. Требуется модулю x/wasm для симуляций.
func (b wasmBank) IsSendEnabledCoin(ctx context.Context, coin sdk.Coin) bool {
	return b.k.IsSendEnabledCoins(sdk.UnwrapSDKContext(ctx), coin) == nil
}

// SpendableCoins — доступная сумма счёта целиком. У Artery метод
// есть, отличается только тип контекста.
func (b wasmBank) SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	return b.k.SpendableCoins(sdk.UnwrapSDKContext(ctx), addr)
}

// --- Стейкинг, которого нет ---

// wasmStaking отвечает отказом на всё.
//
// Именно здесь видно, насколько Artery расходится с обычной сетью Cosmos:
// x/wasm считает существование стейкинга само собой разумеющимся, а у
// Artery валидаторов назначает x/noding по реферальному дереву.
type wasmStaking struct{}

func (wasmStaking) BondDenom(context.Context) (string, error) {
	return "", errNoStaking
}

func (wasmStaking) GetValidator(context.Context, sdk.ValAddress) (stakingTypes.Validator, error) {
	return stakingTypes.Validator{}, errNoStaking
}

func (wasmStaking) GetBondedValidatorsByPower(context.Context) ([]stakingTypes.Validator, error) {
	return nil, errNoStaking
}

func (wasmStaking) GetAllDelegatorDelegations(
	context.Context, sdk.AccAddress,
) ([]stakingTypes.Delegation, error) {
	return nil, errNoStaking
}

func (wasmStaking) GetDelegation(
	context.Context, sdk.AccAddress, sdk.ValAddress,
) (stakingTypes.Delegation, error) {
	return stakingTypes.Delegation{}, errNoStaking
}

func (wasmStaking) HasReceivingRedelegation(
	context.Context, sdk.AccAddress, sdk.ValAddress,
) (bool, error) {
	return false, errNoStaking
}

// ApplyAndReturnValidatorSetUpdates — требуется модулю как источник
// набора валидаторов (ValidatorSetSource); зовётся только при заведении
// сети из генезиса с контрактами. Пустой список честен: набор валидаторов
// Artery формирует x/noding, и wasm на него не влияет.
func (wasmStaking) ApplyAndReturnValidatorSetUpdates(context.Context) ([]abci.ValidatorUpdate, error) {
	return nil, nil
}

// --- Распределение наград, которого тоже нет ---

type wasmDistr struct{}

func (wasmDistr) DelegatorWithdrawAddress(
	context.Context, *distrTypes.QueryDelegatorWithdrawAddressRequest,
) (*distrTypes.QueryDelegatorWithdrawAddressResponse, error) {
	return nil, errNoStaking
}

func (wasmDistr) DelegationRewards(
	context.Context, *distrTypes.QueryDelegationRewardsRequest,
) (*distrTypes.QueryDelegationRewardsResponse, error) {
	return nil, errNoStaking
}

func (wasmDistr) DelegationTotalRewards(
	context.Context, *distrTypes.QueryDelegationTotalRewardsRequest,
) (*distrTypes.QueryDelegationTotalRewardsResponse, error) {
	return nil, errNoStaking
}

func (wasmDistr) DelegatorValidators(
	context.Context, *distrTypes.QueryDelegatorValidatorsRequest,
) (*distrTypes.QueryDelegatorValidatorsResponse, error) {
	return nil, errNoStaking
}
