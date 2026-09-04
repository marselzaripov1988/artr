//go:build testing
// +build testing

package keeper_test

import (
	"github.com/arterynetwork/artr/util"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	storeTypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authK "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	"github.com/arterynetwork/artr/app"
	"github.com/arterynetwork/artr/x/bank"
	"github.com/arterynetwork/artr/x/earning"
	profileK "github.com/arterynetwork/artr/x/profile/keeper"
	"github.com/arterynetwork/artr/x/referral"
)

func TestEarningKeeper(t *testing.T) {
	suite.Run(t, new(Suite))
}

type Suite struct {
	suite.Suite

	app     *app.ArteryApp
	cleanup func()

	cdc      codec.BinaryCodec
	ctx      sdk.Context
	k        earning.Keeper
	ak       authK.AccountKeeper
	bk       bank.Keeper
	pk       profileK.Keeper
	rk       referral.Keeper
	storeKey storeTypes.StoreKey

	bbHeader tmproto.Header
}

func (s *Suite) SetupTest() {
	defer func() {
		if e := recover(); e != nil {
			s.FailNow("panic on setup", "%s", e)
		}
	}()
	s.app, s.cleanup, s.ctx = app.NewAppFromGenesis(nil)

	s.cdc = s.app.Codec()
	s.k = s.app.GetEarningKeeper()
	s.storeKey = s.app.GetKeys()[earning.ModuleName]
	s.ak = s.app.GetAccountKeeper()
	s.bk = s.app.GetBankKeeper()
	s.pk = s.app.GetProfileKeeper()
	s.rk = s.app.GetReferralKeeper()

	s.bbHeader = tmproto.Header{
		ProposerAddress: util.MustParseConsPubKey(app.DefaultUser1ConsPubKey).Address().Bytes(),
	}
}

func (s *Suite) TearDownTest() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

func (s *Suite) nextBlock() (sdk.EndBlock, sdk.BeginBlock) {
	ebr, err := s.app.EndBlocker(s.ctx)
	s.Require().NoError(err)
	// Предложивший блок теперь берётся из контекста, а не из запроса.
	// Заголовок ставится первым: WithBlockHeader заменяет его целиком, а
	// время и высота блока хранятся именно в нём.
	s.ctx = s.ctx.
		WithBlockHeader(s.bbHeader).
		WithBlockHeight(s.ctx.BlockHeight() + 1).
		WithBlockTime(s.ctx.BlockTime().Add(30 * time.Second))
	bbr, err := s.app.BeginBlocker(s.ctx)
	s.Require().NoError(err)
	return ebr, bbr
}
