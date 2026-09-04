//go:build testing
// +build testing

package keeper_test

import (
	"github.com/arterynetwork/artr/util"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/codec"
	storeTypes "github.com/cosmos/cosmos-sdk/store/types"
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

	bbHeader abci.RequestBeginBlock
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

	s.bbHeader = abci.RequestBeginBlock{
		Header: tmproto.Header{
			ProposerAddress: util.MustParseConsPubKey(app.DefaultUser1ConsPubKey).Address().Bytes(),
		},
	}
}

func (s *Suite) TearDownTest() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

func (s *Suite) nextBlock() (abci.ResponseEndBlock, abci.ResponseBeginBlock) {
	ebr := s.app.EndBlocker(s.ctx, abci.RequestEndBlock{})
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1).WithBlockTime(s.ctx.BlockTime().Add(30 * time.Second))
	bbr := s.app.BeginBlocker(s.ctx, s.bbHeader)
	return ebr, bbr
}
