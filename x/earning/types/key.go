package types

const (
	// ModuleName is the name of the module
	ModuleName = "earning"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	// RouterKey to be used for routing msgs
	RouterKey = ModuleName

	// QuerierRoute to be used for querierer msgs
	QuerierRoute = ModuleName

	// VpnCollectorName is the root string for an account address for Artery VPN tariff payment collection
	VpnCollectorName = "vpn"

	// StorageCollectorName is the root string for an account address for Artery Storage tariff payment collection
	StorageCollectorName = "storage"
)

// Префиксы ключей в сторе модуля.
//
// До SDK 0.47 параметры модуля лежали в подпространстве x/params, а стор
// earning содержал только записи получателей — прямо по адресу, без
// префикса. С переносом параметров внутрь модуля разделение стало
// обязательным: обходы стора (экспорт генезиса, clear) иначе принимали бы
// запись параметров за получателя.
var (
	// ParamsKey — параметры модуля.
	ParamsKey = []byte{0x00}

	// EarnerPrefix — записи получателей начислений.
	EarnerPrefix = []byte{0x01}
)
