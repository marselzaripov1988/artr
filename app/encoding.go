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
	// Описания берутся из объединённого реестра, а не из HybridResolver.
	//
	// Разница в порядке сборки. HybridResolver строит дескриптор каждого
	// файла в момент его регистрации, разрешая импорты по тому, что уже
	// зарегистрировано, и подставляя заглушку на то, чего ещё нет.
	// Регистрация идёт из init() сгенерированных файлов, а те внутри
	// пакета исполняются по алфавиту имён: tx.pb.go раньше types.pb.go.
	//
	// Из-за этого artery.voting.v1beta1.Proposal, на который ссылается
	// MsgPropose, оказывался заглушкой — без полей и без опций. Подписант
	// у MsgPropose лежит именно там (Proposal.author), и x/tx его не
	// находил: "no cosmos.msg.v1.signer option found for message
	// artery.voting.v1beta1.Proposal". Голосование и запуск опроса не
	// отправлялись, притом что остальные 26 сообщений проходили.
	//
	// MergedRegistry собирает все описания в один набор и разрешает
	// перекрёстные ссылки целиком, порядок регистрации ему безразличен.
	// Стоит это одного прохода по всем файлам при старте.
	protoFiles, err := proto.MergedRegistry()
	if err != nil {
		panic(err)
	}

	ir, err := codecTypes.NewInterfaceRegistryWithOptions(codecTypes.InterfaceRegistryOptions{
		ProtoFiles: protoFiles,
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
