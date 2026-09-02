package types

const (
	// ModuleName is the name of the module
	ModuleName = "schedule"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	// RouterKey to be used for routing msgs
	RouterKey = ModuleName

	// QuerierRoute to be used for querierer msgs
	QuerierRoute = ModuleName
)

// Префиксы ключей в сторе модуля.
//
// Задачи адресуются меткой времени и читаются диапазонными обходами
// (GetTasks, PerformSchedule). Параметры, перенесённые сюда из
// подпространства x/params, обязаны лежать вне этого диапазона, иначе
// обход "с начала до момента X" зацепил бы запись параметров.
var (
	// ParamsKey — параметры модуля.
	ParamsKey = []byte{0x00}

	// TaskPrefix — запланированные задачи, ключ внутри префикса — метка времени.
	TaskPrefix = []byte{0x01}
)
