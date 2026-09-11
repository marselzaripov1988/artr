package app

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/arterynetwork/artr/x/bank"
)

// Ответы на общепринятые запросы к банку поверх собственного банка Artery.
//
// Artery заменила x/bank своим модулем и обслуживает его по своему пути:
// /artery/bank/v1beta1/balance/{адрес}. Это работает, но по этому пути не
// ходит никто, кроме самой Artery. Кошельки, обозреватели, релееры и
// индексаторы спрашивают cosmos.bank.v1beta1.Query, и до появления этого
// файла получали 501: такого маршрута в узле просто не было.
//
// Последствие было бы видно не в отказе, а в нуле: сеть появляется в
// кошельке и показывает пустой счёт. Обнаружено при подготовке записи в
// реестр сетей Cosmos — точки входа там проверяются ежедневно, и пришлось
// проверить, чем узел вообще отвечает.
//
// Переходник только читает и ничего не решает сам: все ответы собираются
// из того же хранилища, что отдаёт родной путь. Где ответить нечем —
// отвечает Unimplemented, а не правдоподобной пустотой. Пустой список
// означает «ничего нет», и чужой код сделает из него вывод; отказ он
// хотя бы заметит.
type bankShim struct{ k bank.Keeper }

var _ bankTypes.QueryServer = bankShim{}

// Balance — остаток одного денома на счёте.
func (s bankShim) Balance(ctx context.Context, req *bankTypes.QueryBalanceRequest) (*bankTypes.QueryBalanceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.Denom == "" {
		return nil, status.Error(codes.InvalidArgument, "denom cannot be empty")
	}
	sdkCtx, addr, err := s.unwrap(ctx, req.Address)
	if err != nil {
		return nil, err
	}

	coin := sdk.NewCoin(req.Denom, s.k.GetBalance(sdkCtx, addr).AmountOf(req.Denom))
	return &bankTypes.QueryBalanceResponse{Balance: &coin}, nil
}

// AllBalances — все остатки счёта.
func (s bankShim) AllBalances(ctx context.Context, req *bankTypes.QueryAllBalancesRequest) (*bankTypes.QueryAllBalancesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	sdkCtx, addr, err := s.unwrap(ctx, req.Address)
	if err != nil {
		return nil, err
	}

	coins := s.k.GetBalance(sdkCtx, addr)
	if req.ResolveDenom {
		coins = s.resolveDenoms(sdkCtx, coins)
	}

	page, pageResp, err := paginate(coins, req.Pagination)
	if err != nil {
		return nil, err
	}
	return &bankTypes.QueryAllBalancesResponse{Balances: page, Pagination: pageResp}, nil
}

// SpendableBalances — доступные к трате остатки.
//
// У Artery они совпадают с остатками: блокировок нет, SpendableCoins
// собственного банка возвращает тот же GetBalance. Запрос отвечает
// честно, а не отказом, потому что ответ действительно известен.
func (s bankShim) SpendableBalances(ctx context.Context, req *bankTypes.QuerySpendableBalancesRequest) (*bankTypes.QuerySpendableBalancesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	sdkCtx, addr, err := s.unwrap(ctx, req.Address)
	if err != nil {
		return nil, err
	}

	page, pageResp, err := paginate(s.k.SpendableCoins(sdkCtx, addr), req.Pagination)
	if err != nil {
		return nil, err
	}
	return &bankTypes.QuerySpendableBalancesResponse{Balances: page, Pagination: pageResp}, nil
}

// SpendableBalanceByDenom — доступный к трате остаток одного денома.
func (s bankShim) SpendableBalanceByDenom(ctx context.Context, req *bankTypes.QuerySpendableBalanceByDenomRequest) (*bankTypes.QuerySpendableBalanceByDenomResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.Denom == "" {
		return nil, status.Error(codes.InvalidArgument, "denom cannot be empty")
	}
	sdkCtx, addr, err := s.unwrap(ctx, req.Address)
	if err != nil {
		return nil, err
	}

	coin := sdk.NewCoin(req.Denom, s.k.SpendableCoins(sdkCtx, addr).AmountOf(req.Denom))
	return &bankTypes.QuerySpendableBalanceByDenomResponse{Balance: &coin}, nil
}

// TotalSupply — вся эмиссия по деномам.
func (s bankShim) TotalSupply(ctx context.Context, req *bankTypes.QueryTotalSupplyRequest) (*bankTypes.QueryTotalSupplyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	page, pageResp, err := paginate(sdk.Coins(s.k.GetSupply(sdkCtx).Total), req.Pagination)
	if err != nil {
		return nil, err
	}
	return &bankTypes.QueryTotalSupplyResponse{Supply: page, Pagination: pageResp}, nil
}

// SupplyOf — эмиссия одного денома.
func (s bankShim) SupplyOf(ctx context.Context, req *bankTypes.QuerySupplyOfRequest) (*bankTypes.QuerySupplyOfResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.Denom == "" {
		return nil, status.Error(codes.InvalidArgument, "denom cannot be empty")
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	total := sdk.Coins(s.k.GetSupply(sdkCtx).Total)
	return &bankTypes.QuerySupplyOfResponse{Amount: sdk.NewCoin(req.Denom, total.AmountOf(req.Denom))}, nil
}

// Params — параметры банка в понятиях SDK.
//
// Свои параметры Artery отдаёт по своему пути и они другие: комиссия,
// её потолок, доли разделения. Здесь переводится ровно то, что у SDK
// вообще есть, — можно ли отправлять.
//
// Перевод односторонний и неполный сознательно. DefaultSendEnabled —
// истина: Artery запрещает отправку не по деному, а по отправителю
// (BlockedSenders), и выразить это в параметрах SDK нечем. Клиент,
// которому важно, заблокирован ли счёт, должен спрашивать Artery по её
// собственному пути; подставлять сюда ложь было бы хуже, чем не
// отвечать.
func (s bankShim) Params(ctx context.Context, req *bankTypes.QueryParamsRequest) (*bankTypes.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	return &bankTypes.QueryParamsResponse{
		Params: bankTypes.Params{DefaultSendEnabled: true},
	}, nil
}

// DenomsMetadata — описания всех деномов, о которых сеть знает.
func (s bankShim) DenomsMetadata(ctx context.Context, req *bankTypes.QueryDenomsMetadataRequest) (*bankTypes.QueryDenomsMetadataResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	var all []bankTypes.Metadata
	s.k.IterateAllDenomMetaData(sdkCtx, func(md bankTypes.Metadata) bool {
		all = append(all, md)
		return false
	})

	page, pageResp, err := paginate(all, req.Pagination)
	if err != nil {
		return nil, err
	}
	return &bankTypes.QueryDenomsMetadataResponse{Metadatas: page, Pagination: pageResp}, nil
}

// DenomMetadata — описание одного денома.
func (s bankShim) DenomMetadata(ctx context.Context, req *bankTypes.QueryDenomMetadataRequest) (*bankTypes.QueryDenomMetadataResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	md, err := s.metadata(ctx, req.Denom)
	if err != nil {
		return nil, err
	}
	return &bankTypes.QueryDenomMetadataResponse{Metadata: md}, nil
}

// DenomMetadataByQueryString — то же самое, деном приходит строкой запроса.
func (s bankShim) DenomMetadataByQueryString(ctx context.Context, req *bankTypes.QueryDenomMetadataByQueryStringRequest) (*bankTypes.QueryDenomMetadataByQueryStringResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	md, err := s.metadata(ctx, req.Denom)
	if err != nil {
		return nil, err
	}
	return &bankTypes.QueryDenomMetadataByQueryStringResponse{Metadata: md}, nil
}

// DenomOwners — кто держит деном. Ответить нечем.
//
// В SDK на это есть отдельный указатель: деном -> счета. У Artery
// хранение другое — остаток лежит целиком на счёте, одной записью со
// всеми деномами сразу, и обратного указателя нет. Чтобы ответить,
// пришлось бы прочесть все счета целиком: в выгрузке мейннета их сто
// шестьдесят тысяч. Один такой запрос занимает узел, десяток кладёт.
//
// Отказ здесь — не недоделка, а отказ открывать наружу полный перебор
// хранилища. Понадобится всерьёз — заводить указатель в x/artrbank и
// наполнять его при изменении остатка, это правка состояния.
func (s bankShim) DenomOwners(context.Context, *bankTypes.QueryDenomOwnersRequest) (*bankTypes.QueryDenomOwnersResponse, error) {
	return nil, errNoDenomIndex("denom owners")
}

func (s bankShim) DenomOwnersByQuery(context.Context, *bankTypes.QueryDenomOwnersByQueryRequest) (*bankTypes.QueryDenomOwnersByQueryResponse, error) {
	return nil, errNoDenomIndex("denom owners")
}

// SendEnabled — деномы с особой настройкой отправки.
//
// Пустой список здесь — правильный ответ, а не отговорка: по описанию
// самого SDK запрос возвращает только деномы с отдельной настройкой, а
// у Artery таких нет вовсе. Отправку она ограничивает по отправителю,
// не по деному.
func (s bankShim) SendEnabled(ctx context.Context, req *bankTypes.QuerySendEnabledRequest) (*bankTypes.QuerySendEnabledResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	return &bankTypes.QuerySendEnabledResponse{
		SendEnabled: []*bankTypes.SendEnabled{},
		Pagination:  &query.PageResponse{Total: 0},
	}, nil
}

func (s bankShim) unwrap(ctx context.Context, bech32 string) (sdk.Context, sdk.AccAddress, error) {
	if bech32 == "" {
		return sdk.Context{}, nil, status.Error(codes.InvalidArgument, "address cannot be empty")
	}
	addr, err := sdk.AccAddressFromBech32(bech32)
	if err != nil {
		return sdk.Context{}, nil, status.Errorf(codes.InvalidArgument, "invalid address: %v", err)
	}
	return sdk.UnwrapSDKContext(ctx), addr, nil
}

func (s bankShim) metadata(ctx context.Context, denom string) (bankTypes.Metadata, error) {
	if denom == "" {
		return bankTypes.Metadata{}, status.Error(codes.InvalidArgument, "denom cannot be empty")
	}
	md, ok := s.k.GetDenomMetaData(sdk.UnwrapSDKContext(ctx), denom)
	if !ok {
		return bankTypes.Metadata{}, status.Errorf(codes.NotFound, "client metadata for denom %s", denom)
	}
	return md, nil
}

// resolveDenoms заменяет деном на его отображаемое имя там, где описание
// заведено. Так поступает и SDK: воучер ibc/<хеш> без этого показывается
// голым хешем.
func (s bankShim) resolveDenoms(ctx sdk.Context, coins sdk.Coins) sdk.Coins {
	out := make(sdk.Coins, 0, len(coins))
	for _, c := range coins {
		if md, ok := s.k.GetDenomMetaData(ctx, c.Denom); ok && md.Display != "" {
			c = sdk.NewCoin(md.Display, c.Amount)
		}
		out = append(out, c)
	}
	return out
}

func errNoDenomIndex(what string) error {
	return status.Errorf(codes.Unimplemented,
		"%s: у Artery нет указателя деном->счета, а полный перебор счетов наружу не открывается", what)
}

// paginate режет готовый срез по смещению и пределу.
//
// Так можно, потому что резать нечего: остаток счёта — одна запись со
// всеми деномами, эмиссия — одна запись, описаний деномов столько,
// сколько воучеров пришло по IBC. Все три множества малы и уже в
// памяти, когда мы сюда попадаем.
//
// Ключевая разбивка (PageRequest.Key) не поддерживается намеренно:
// ключей у этих множеств нет, и притворяться, что есть, значило бы
// отдавать клиенту курсор, по которому он не вернётся.
func paginate[T any](items []T, page *query.PageRequest) ([]T, *query.PageResponse, error) {
	total := uint64(len(items))
	if page == nil {
		return items, &query.PageResponse{Total: total}, nil
	}
	if len(page.Key) != 0 {
		return nil, nil, status.Error(codes.InvalidArgument,
			"pagination by key is not supported here, use offset")
	}

	limit := page.Limit
	if limit == 0 {
		limit = query.DefaultLimit
	}

	if page.Offset >= total {
		return []T{}, &query.PageResponse{Total: total}, nil
	}
	end := page.Offset + limit
	if end > total {
		end = total
	}
	return items[page.Offset:end], &query.PageResponse{Total: total}, nil
}
