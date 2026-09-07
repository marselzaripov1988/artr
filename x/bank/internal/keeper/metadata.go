package keeper

import (
	"cosmossdk.io/store/prefix"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/arterynetwork/artr/x/bank/types"
)

// Метаданные денома: как его показывать и из каких единиц он состоит.
//
// Собственным деномам Artery они не нужны — их знает кошелёк сети. Нужны
// они пришедшим по IBC: при первом переводе извне модуль переводов
// заводит воучер вида ibc/<хеш> и записывает рядом описание, иначе
// кошельки покажут пользователю голый хеш.
//
// Место под них было размечено давно — DenomMetadataPrefix и
// DenomMetadataKey лежат в types/key.go с тех пор, как банк отпочковался
// от банка SDK, — но методов не было. Здесь они и появляются.
//
// Хранится структура из SDK, а не своя: её же требует интерфейс модуля
// переводов, и заводить рядом второй формат ради одного поля незачем.

// GetDenomMetaData возвращает метаданные денома и признак их наличия.
func (k BaseKeeper) GetDenomMetaData(ctx sdk.Context, denom string) (bankTypes.Metadata, bool) {
	bz := ctx.KVStore(k.storeKey).Get(types.DenomMetadataKey(denom))
	if bz == nil {
		return bankTypes.Metadata{}, false
	}

	var md bankTypes.Metadata
	k.cdc.MustUnmarshal(bz, &md)
	return md, true
}

// HasDenomMetaData сообщает, заведены ли метаданные денома.
func (k BaseKeeper) HasDenomMetaData(ctx sdk.Context, denom string) bool {
	return ctx.KVStore(k.storeKey).Has(types.DenomMetadataKey(denom))
}

// SetDenomMetaData записывает метаданные денома.
func (k BaseKeeper) SetDenomMetaData(ctx sdk.Context, md bankTypes.Metadata) {
	ctx.KVStore(k.storeKey).Set(types.DenomMetadataKey(md.Base), k.cdc.MustMarshal(&md))
}

// IterateAllDenomMetaData обходит метаданные всех деномов.
//
// Нужен выгрузке генезиса: без него описания воучеров пропадут при
// экспорте и не вернутся при импорте.
func (k BaseKeeper) IterateAllDenomMetaData(ctx sdk.Context, cb func(bankTypes.Metadata) bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DenomMetadataPrefix)
	it := store.Iterator(nil, nil)
	defer it.Close()

	for ; it.Valid(); it.Next() {
		var md bankTypes.Metadata
		k.cdc.MustUnmarshal(it.Value(), &md)
		if cb(md) {
			break
		}
	}
}
