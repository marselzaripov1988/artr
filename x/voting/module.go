package voting

import (
	"context"
	"encoding/json"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codecTypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/arterynetwork/artr/x/voting/client/cli"
	"github.com/arterynetwork/artr/x/voting/keeper"
	"github.com/arterynetwork/artr/x/voting/types"
)

// TypeCode check to ensure the interface is properly implemented
var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// AppModuleBasic defines the basic application module used by the voting module.
type AppModuleBasic struct{}

// Name returns the voting module's name.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterCodec registers the voting module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

func (AppModuleBasic) RegisterInterfaces(registry codecTypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// DefaultGenesis returns default genesis state as raw bytes for the voting
// module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the voting module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var data types.GenesisState
	err := cdc.UnmarshalJSON(bz, &data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal x/voting genesis state")
	}
	return types.ValidateGenesis(data)
}

// GetTxCmd returns the root tx command for the voting module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}

// GetQueryCmd returns no root query command for the voting module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.NewQueryCmd()
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx))
}

//____________________________________________________________________________

// AppModule implements an application module for the voting module.
type AppModule struct {
	AppModuleBasic

	keeper           keeper.Keeper
	scheduleKeeper   types.ScheduleKeeper
	upgradeKeeper    types.UprgadeKeeper
	nodingKeeper     types.NodingKeeper
	delegatingKeeper types.DelegatingKeeper
	referralKeeper   types.ReferralKeeper
	profileKeeper    types.ProfileKeeper
	earningKeeper    types.EarningKeeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(k keeper.Keeper,
	scheduleKeeper types.ScheduleKeeper,
	uprgadeKeeper types.UprgadeKeeper,
	nodingKeeper types.NodingKeeper,
	delegatingKeeper types.DelegatingKeeper,
	referralKeeper types.ReferralKeeper,
	profileKeeper types.ProfileKeeper,
	earningKeeper types.EarningKeeper,
) AppModule {
	return AppModule{
		AppModuleBasic:   AppModuleBasic{},
		keeper:           k,
		scheduleKeeper:   scheduleKeeper,
		upgradeKeeper:    uprgadeKeeper,
		nodingKeeper:     nodingKeeper,
		delegatingKeeper: delegatingKeeper,
		referralKeeper:   referralKeeper,
		profileKeeper:    profileKeeper,
		earningKeeper:    earningKeeper,
	}
}

// Name returns the voting module's name.
func (AppModule) Name() string { return types.ModuleName }

func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.MsgServer(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.QueryServer(am.keeper))
}

// RegisterInvariants registers the voting module invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// InitGenesis performs genesis initialization for the voting module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, mrshl codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState types.GenesisState
	mrshl.MustUnmarshalJSON(data, &genesisState)
	InitGenesis(ctx, am.keeper, genesisState)
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the exported genesis state as raw bytes for the voting
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, mrshl codec.JSONCodec) json.RawMessage {
	gs := ExportGenesis(ctx, am.keeper)
	return mrshl.MustMarshalJSON(gs)
}

// ConsensusVersion версия схемы состояния модуля. Требование SDK 0.43+:
// менеджер модулей сравнивает её с записанной в сторе и по расхождению
// запускает зарегистрированные миграции. Единица — исходная версия, с
// которой модуль входит в новую схему.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// IsAppModule и IsOnePerModuleType — метки appmodule.AppModule из SDK 0.50:
// пустые методы, которыми модуль объявляет себя модулем.
func (AppModule) IsAppModule()        {}
func (AppModule) IsOnePerModuleType() {}
