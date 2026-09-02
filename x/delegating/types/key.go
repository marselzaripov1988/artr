package types

const (
	// ModuleName is the name of the module
	ModuleName = "delegating"

	// MainStoreKey to be used when creating the KVStore
	MainStoreKey = ModuleName

	// RouterKey to be used for routing msgs
	RouterKey = ModuleName

	// QuerierRoute to be used for querierer msgs
	QuerierRoute = ModuleName

	RevokeHookName = "delegating/revoke"
	AccrueHookName = "delegating/accrue"
)

// Префиксы ключей в сторе модуля.
//
// Записи делегирования адресуются прямо адресом аккаунта, а экспорт
// генезиса обходит стор целиком. После переноса параметров из
// подпространства x/params их нужно держать вне этого пространства ключей.
var (
	// ParamsKey — параметры модуля.
	ParamsKey = []byte{0x00}

	// RecordPrefix — записи делегирования по адресу аккаунта.
	RecordPrefix = []byte{0x01}
)
