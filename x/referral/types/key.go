package types

const (
	// ModuleName is the name of the module
	ModuleName = "referral"

	// StoreKey to be used when creating the KVStore
	StoreKey      = ModuleName
	IndexStoreKey = ModuleName + "-index"

	// RouterKey to be used for routing msgs
	RouterKey = ModuleName

	// QuerierRoute to be used for querierer msgs
	QuerierRoute = ModuleName
)

// Префиксы ключей в основном сторе модуля.
//
// Записи дерева адресуются адресом аккаунта, и по стору идут полные
// обходы. Параметры, перенесённые из подпространства x/params, разведены
// с данными префиксом.
var (
	// ParamsKey — параметры модуля.
	ParamsKey = []byte{0x00}

	// InfoPrefix — записи реферального дерева по адресу аккаунта.
	InfoPrefix = []byte{0x01}
)
