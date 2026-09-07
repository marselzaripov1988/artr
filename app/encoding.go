package app

import (
	"cosmossdk.io/x/tx/signing"
	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	addressCodec "github.com/cosmos/cosmos-sdk/codec/address"
	codecTypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type EncodingConfig struct {
	Marshaler         codec.Codec
	InterfaceRegistry codecTypes.InterfaceRegistry
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

func (conf EncodingConfig) BuildClientContext() client.Context {
	return client.Context{}.
		WithCodec(conf.Marshaler).
		WithInterfaceRegistry(conf.InterfaceRegistry).
		WithTxConfig(conf.TxConfig).
		WithLegacyAmino(conf.Amino).
		WithAccountRetriever(authTypes.AccountRetriever{})
}

func NewEncodingConfig() EncodingConfig {
	// Реестру интерфейсов нужны кодеки адресов.
	//
	// С SDK 0.50 разбор транзакции идёт через x/tx, и тот переводит адреса
	// подписантов из строк в байты сам — не через глобальный конфиг, а
	// через кодеки, заданные реестру. Без них любая попытка разобрать или
	// оценить транзакцию падает с "InterfaceRegistry requires a proper
	// address codec implementation".
	//
	// Проявляется это не на своих путях: собственные сообщения Artery
	// проходят, а спотыкается симуляция транзакции по gRPC — именно ею
	// пользуется релеер, и на ней IBC и останавливался.
	//
	// Префикс валидатора совпадает с общим: у Artery нет отдельного
	// пространства адресов валидаторов, ими ведает x/noding по адресам
	// обычных счетов.
	ir, err := codecTypes.NewInterfaceRegistryWithOptions(codecTypes.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec:          addressCodec.NewBech32Codec(Bech32PrefixAccAddr),
			ValidatorAddressCodec: addressCodec.NewBech32Codec(Bech32PrefixValAddr),
		},
	})
	if err != nil {
		panic(err)
	}

	var (
		cdc      = codec.NewProtoCodec(ir)
		txConfig = tx.NewTxConfig(cdc, tx.DefaultSignModes)
		amino    = codec.NewLegacyAmino()
	)
	return EncodingConfig{
		InterfaceRegistry: ir,
		Marshaler:         cdc,
		TxConfig:          txConfig,
		Amino:             amino,
	}
}
