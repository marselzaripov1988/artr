package types

const (
	// ModuleName is the name of the module
	ModuleName = "profile"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	// Used when creating account alias KVStore
	AliasStoreKey = ModuleName + "Aliases"

	// Used when creating account card numbers KVStore
	CardStoreKey = ModuleName + "Cards"

	// RouterKey to be used for routing msgs
	RouterKey = ModuleName

	// QuerierRoute to be used for querierer msgs
	QuerierRoute = ModuleName

	RefreshHookName   = ModuleName + "/refresh"
	RefreshImHookName = ModuleName + "/refresh-im"
)

// Префиксы ключей в основном сторе модуля.
//
// Профили адресуются адресом аккаунта, а экспорт генезиса обходит стор
// целиком. После переноса параметров из подпространства x/params их
// пространство ключей разведено с профилями.
var (
	// ParamsKey — параметры модуля.
	ParamsKey = []byte{0x00}

	// ProfilePrefix — профили по адресу аккаунта.
	ProfilePrefix = []byte{0x01}
)
