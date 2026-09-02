package types

const (
	// ModuleName is the name of the module
	ModuleName = "noding"

	// StoreKey is to be used when creating the KVStore for module data
	StoreKey    = ModuleName
	IdxStoreKey = StoreKey + "-index"

	// RouterKey to be used for routing msgs
	RouterKey = ModuleName

	// QuerierRoute to be used for querierer msgs
	QuerierRoute = ModuleName
)

// Префиксы ключей в основном сторе модуля.
//
// Записи валидаторов адресуются адресом аккаунта, и по стору идут полные
// обходы (сбор активных, обновления, экспорт). Параметры, перенесённые из
// подпространства x/params, разведены с данными префиксом.
var (
	// ParamsKey — параметры модуля.
	ParamsKey = []byte{0x00}

	// InfoPrefix — записи валидаторов по адресу аккаунта.
	InfoPrefix = []byte{0x01}
)
