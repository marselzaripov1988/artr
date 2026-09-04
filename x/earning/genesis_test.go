//go:build testing
// +build testing

package earning_test

import (
	"github.com/arterynetwork/artr/util"
	"testing"

	"github.com/stretchr/testify/suite"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/arterynetwork/artr/app"
	"github.com/arterynetwork/artr/x/earning"
	schedule "github.com/arterynetwork/artr/x/schedule/types"
)

func TestEarningGenesis(t *testing.T) {
	suite.Run(t, new(Suite))
}

type Suite struct {
	suite.Suite

	app     *app.ArteryApp
	cleanup func()
	ctx     sdk.Context
	k       earning.Keeper

	bbHeader abci.RequestBeginBlock
}

func (s *Suite) SetupTest() {
	defer func() {
		if e := recover(); e != nil {
			s.FailNow("panic on setup", e)
		}
	}()
	s.app, s.cleanup, s.ctx = app.NewAppFromGenesis(nil)
	s.k = s.app.GetEarningKeeper()

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

func (s Suite) TestCleanGenesis() {
	s.checkExportImport()
}

func (s *Suite) TestParams() {
	s.k.SetParams(s.ctx, earning.Params{
		Signers: []string{
			app.DefaultGenesisUsers["user9"].String(),
		},
	})
	s.checkExportImport()
}

func (s Suite) checkExportImport() {
	s.app.CheckExportImport(s.T(),
		s.ctx.BlockTime(),
		[]string{
			earning.StoreKey,
			schedule.StoreKey,
		},
		map[string]app.Decoder{
			earning.StoreKey:  app.AccAddressDecoder,
			schedule.StoreKey: app.Uint64Decoder,
		},
		map[string]app.Decoder{
			earning.StoreKey:  app.DummyDecoder,
			schedule.StoreKey: app.DummyDecoder,
		},
		make(map[string][][]byte, 0),
	)
}
