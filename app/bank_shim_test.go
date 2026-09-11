//go:build testing
// +build testing

package app

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/arterynetwork/artr/util"
)

// Переходник к банку.
//
// Проверяется одно: по общепринятому пути видно ровно то же, что по
// родному пути Artery. До появления переходника по нему не было видно
// ничего — узел отвечал 501, и сеть в кошельке показывала бы пустой
// счёт при непустом.

func TestBankShim(t *testing.T) {
	suite.Run(t, new(BankShimSuite))
}

type BankShimSuite struct {
	suite.Suite

	app     *ArteryApp
	cleanup func()
	ctx     sdk.Context
	shim    bankShim
}

func (s *BankShimSuite) SetupTest() {
	s.app, s.cleanup, s.ctx = NewAppFromGenesis(nil)
	s.shim = bankShim{k: s.app.GetBankKeeper()}
}

func (s *BankShimSuite) TearDownTest() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

// TestAllBalancesMatchesNativePath — общепринятый путь отдаёт то же, что
// родной.
//
// Это главная проверка файла: расхождение здесь означает, что кошелёк
// показывает пользователю не его деньги.
func (s *BankShimSuite) TestAllBalancesMatchesNativePath() {
	user := DefaultGenesisUsers["user1"]

	native := s.app.GetBankKeeper().GetBalance(s.ctx, user)
	s.Require().False(native.IsZero(), "у подопытного счёта пусто — проверять нечего")

	resp, err := s.shim.AllBalances(sdk.WrapSDKContext(s.ctx), &bankTypes.QueryAllBalancesRequest{
		Address: user.String(),
	})
	s.Require().NoError(err)
	s.Equal(native, resp.Balances, "остатки по двум путям разошлись")
	s.Require().NotNil(resp.Pagination)
	s.EqualValues(len(native), resp.Pagination.Total)
}

// TestBalanceByDenom — остаток одного денома.
func (s *BankShimSuite) TestBalanceByDenom() {
	user := DefaultGenesisUsers["user1"]
	want := s.app.GetBankKeeper().GetBalance(s.ctx, user).AmountOf(util.ConfigMainDenom)
	s.Require().True(want.IsPositive())

	resp, err := s.shim.Balance(sdk.WrapSDKContext(s.ctx), &bankTypes.QueryBalanceRequest{
		Address: user.String(),
		Denom:   util.ConfigMainDenom,
	})
	s.Require().NoError(err)
	s.Require().NotNil(resp.Balance)
	s.Equal(util.ConfigMainDenom, resp.Balance.Denom)
	s.Equal(want, resp.Balance.Amount)
}

// TestUnknownDenomIsZeroNotError — неизвестный деном даёт ноль.
//
// Так поступает и SDK. Отказ здесь ломал бы кошельки, которые спрашивают
// про деном до того, как он на счету появится.
func (s *BankShimSuite) TestUnknownDenomIsZeroNotError() {
	resp, err := s.shim.Balance(sdk.WrapSDKContext(s.ctx), &bankTypes.QueryBalanceRequest{
		Address: DefaultGenesisUsers["user1"].String(),
		Denom:   "ibc/0000000000000000000000000000000000000000000000000000000000000000",
	})
	s.Require().NoError(err)
	s.Require().NotNil(resp.Balance)
	s.True(resp.Balance.Amount.IsZero())
}

// TestTotalSupplyMatchesNativePath — эмиссия по двум путям совпадает.
func (s *BankShimSuite) TestTotalSupplyMatchesNativePath() {
	native := sdk.Coins(s.app.GetBankKeeper().GetSupply(s.ctx).Total)
	s.Require().False(native.IsZero())

	resp, err := s.shim.TotalSupply(sdk.WrapSDKContext(s.ctx), &bankTypes.QueryTotalSupplyRequest{})
	s.Require().NoError(err)
	s.Equal(native, resp.Supply)

	one, err := s.shim.SupplyOf(sdk.WrapSDKContext(s.ctx), &bankTypes.QuerySupplyOfRequest{
		Denom: util.ConfigMainDenom,
	})
	s.Require().NoError(err)
	s.Equal(native.AmountOf(util.ConfigMainDenom), one.Amount.Amount)
}

// TestBadAddressIsRejected — мусор вместо адреса не должен проходить за
// пустой счёт.
func (s *BankShimSuite) TestBadAddressIsRejected() {
	for _, addr := range []string{"", "не адрес", "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq"} {
		_, err := s.shim.AllBalances(sdk.WrapSDKContext(s.ctx), &bankTypes.QueryAllBalancesRequest{Address: addr})
		s.Require().Error(err, "адрес %q принят", addr)
		s.Equal(codes.InvalidArgument, status.Code(err), "адрес %q: не тот код ошибки", addr)
	}
}

// TestDenomOwnersRefusesInsteadOfLying — на запрос держателей денома
// приходит отказ, а не пустой список.
//
// Разница существенная: пустой список чужой код примет за факт «денома
// ни у кого нет» и на этом построит вывод. Указателя деном->счета у
// Artery нет, и притворяться, что ответ известен, нельзя.
func (s *BankShimSuite) TestDenomOwnersRefusesInsteadOfLying() {
	_, err := s.shim.DenomOwners(sdk.WrapSDKContext(s.ctx), &bankTypes.QueryDenomOwnersRequest{
		Denom: util.ConfigMainDenom,
	})
	s.Require().Error(err)
	s.Equal(codes.Unimplemented, status.Code(err))
}

// TestDenomMetadataResolvesDisplay — заведённое описание денома
// подставляется вместо самого денома, когда об этом просят.
//
// Ради этого описания и писался metadata.go: воучер ibc/<хеш> без него
// кошелёк показывает голым хешем.
func (s *BankShimSuite) TestDenomMetadataResolvesDisplay() {
	const voucher = "ibc/1111111111111111111111111111111111111111111111111111111111111111"

	bk := s.app.GetBankKeeper()
	user := DefaultGenesisUsers["user1"]

	bk.SetDenomMetaData(s.ctx, bankTypes.Metadata{
		Base:    voucher,
		Display: "atom",
		Symbol:  "ATOM",
		Name:    "Cosmos Hub Atom",
		DenomUnits: []*bankTypes.DenomUnit{
			{Denom: voucher, Exponent: 0},
			{Denom: "atom", Exponent: 6},
		},
	})
	s.Require().NoError(bk.AddCoins(s.ctx, user, sdk.NewCoins(sdk.NewInt64Coin(voucher, 42))))

	md, err := s.shim.DenomMetadata(sdk.WrapSDKContext(s.ctx), &bankTypes.QueryDenomMetadataRequest{Denom: voucher})
	s.Require().NoError(err)
	s.Equal("atom", md.Metadata.Display)

	plain, err := s.shim.AllBalances(sdk.WrapSDKContext(s.ctx), &bankTypes.QueryAllBalancesRequest{
		Address: user.String(),
	})
	s.Require().NoError(err)
	s.EqualValues(42, plain.Balances.AmountOf(voucher).Int64(), "без просьбы деном подменять нельзя")

	resolved, err := s.shim.AllBalances(sdk.WrapSDKContext(s.ctx), &bankTypes.QueryAllBalancesRequest{
		Address:      user.String(),
		ResolveDenom: true,
	})
	s.Require().NoError(err)
	s.EqualValues(42, resolved.Balances.AmountOf("atom").Int64(), "деном не подменён на отображаемый")
}

// TestMissingMetadataIsNotFound — описания нет, значит NotFound, а не
// пустая структура.
func (s *BankShimSuite) TestMissingMetadataIsNotFound() {
	_, err := s.shim.DenomMetadata(sdk.WrapSDKContext(s.ctx), &bankTypes.QueryDenomMetadataRequest{
		Denom: "ibc/2222222222222222222222222222222222222222222222222222222222222222",
	})
	s.Require().Error(err)
	s.Equal(codes.NotFound, status.Code(err))
}

// TestPaginationCutsAndCounts — смещение и предел режут выдачу, а общее
// число остаётся полным.
func (s *BankShimSuite) TestPaginationCutsAndCounts() {
	items := []int{0, 1, 2, 3, 4}

	all, resp, err := paginate(items, nil)
	s.Require().NoError(err)
	s.Equal(items, all)
	s.EqualValues(5, resp.Total)

	page, resp, err := paginate(items, &query.PageRequest{Offset: 1, Limit: 2})
	s.Require().NoError(err)
	s.Equal([]int{1, 2}, page)
	s.EqualValues(5, resp.Total, "общее число должно оставаться полным, а не размером страницы")

	tail, _, err := paginate(items, &query.PageRequest{Offset: 4, Limit: 10})
	s.Require().NoError(err)
	s.Equal([]int{4}, tail, "предел за концом среза не должен выходить за границы")

	past, _, err := paginate(items, &query.PageRequest{Offset: 99, Limit: 10})
	s.Require().NoError(err)
	s.Empty(past)

	_, _, err = paginate(items, &query.PageRequest{Key: []byte("курсор")})
	s.Require().Error(err, "ключевая разбивка не поддерживается и должна отказывать явно")
	s.Equal(codes.InvalidArgument, status.Code(err))
}

// TestSendEnabledIsEmptyByDesign — пустой список здесь правильный ответ.
//
// По описанию SDK запрос возвращает только деномы с отдельной
// настройкой отправки. У Artery таких нет: она ограничивает отправку по
// отправителю, а не по деному.
func (s *BankShimSuite) TestSendEnabledIsEmptyByDesign() {
	resp, err := s.shim.SendEnabled(sdk.WrapSDKContext(s.ctx), &bankTypes.QuerySendEnabledRequest{})
	s.Require().NoError(err)
	s.Empty(resp.SendEnabled)

	params, err := s.shim.Params(sdk.WrapSDKContext(s.ctx), &bankTypes.QueryParamsRequest{})
	s.Require().NoError(err)
	s.True(params.Params.DefaultSendEnabled, "отправка у Artery по умолчанию разрешена")
}
