package app

import (
	"encoding/json"
	serverTypes "github.com/cosmos/cosmos-sdk/server/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ExportAppStateAndValidators exports the state of the application for a genesis
// file.
func (app *ArteryApp) ExportAppStateAndValidators(
	forZeroHeight bool, jailWhiteList []string,
	// SDK 0.47 добавил список модулей для выборочного экспорта. Artery
	// выгружает состояние целиком, поэтому параметр не используется.
	modulesToExport []string,
) (serverTypes.ExportedApp, error) {

	// as if they could withdraw from the start of the next block
	// С SDK 0.50 контекст строится без заголовка: высота берётся из
	// самого приложения.
	ctx := app.NewContext(true)

	// We export at last height + 1, because that's the height at which
	// Tendermint will start InitChain.
	height := app.LastBlockHeight() + 1

	if forZeroHeight {
		height = 0
		app.prepForZeroHeightGenesis(ctx, jailWhiteList)
	}

	genState, err := app.mm.ExportGenesis(ctx, app.ec.Marshaler)
	if err != nil {
		return serverTypes.ExportedApp{}, err
	}
	appState, err := json.MarshalIndent(genState, "", "  ")
	if err != nil {
		return serverTypes.ExportedApp{}, err
	}

	// We should never have genesis validators per se.
	// All validators should be added via noding module instead.
	return serverTypes.ExportedApp{
		AppState:        appState,
		Validators:      nil,
		Height:          height,
		ConsensusParams: app.BaseApp.GetConsensusParams(ctx),
	}, nil
}

// prepare for fresh start at zero height
// NOTE zero height genesis is a temporary feature which will be deprecated
//      in favour of export at a block height
func (app *ArteryApp) prepForZeroHeightGenesis(ctx sdk.Context, jailWhiteList []string) {
	panic("export to zero height genesis is not supported")
	// Almost every module schedules something and has block height somewhere in its data.
	// All these heights must be carefully patched if we want this feature implemented.
}
