//go:build testing
// +build testing

package app

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	path "path/filepath"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"cosmossdk.io/log"
	tmjson "github.com/cometbft/cometbft/libs/json"
	tmtypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"

	storeTypes "cosmossdk.io/store/types"
	upgradeKeeper "cosmossdk.io/x/upgrade/keeper"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	crypto "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/kv"
	authKeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	"github.com/arterynetwork/artr/x/bank"
	"github.com/arterynetwork/artr/x/delegating"
	"github.com/arterynetwork/artr/x/earning"
	"github.com/arterynetwork/artr/x/noding"
	profileKeeper "github.com/arterynetwork/artr/x/profile/keeper"
	"github.com/arterynetwork/artr/x/referral"
	scheduleKeeper "github.com/arterynetwork/artr/x/schedule/keeper"
	schedule "github.com/arterynetwork/artr/x/schedule/types"
	votingKeeper "github.com/arterynetwork/artr/x/voting/keeper"
)

func init() {
	InitConfig()
	initDefaultGenesisUsers()
}

const verbose = false
const printGenesis = false

func (app ArteryApp) GetKeys() map[string]*storeTypes.KVStoreKey { return app.keys }

func (app ArteryApp) GetAccountKeeper() authKeeper.AccountKeeper { return app.accountKeeper }
func (app ArteryApp) GetBankKeeper() bank.Keeper                 { return app.bankKeeper }
func (app ArteryApp) GetUpgradeKeeper() *upgradeKeeper.Keeper    { return app.upgradeKeeper }
func (app ArteryApp) GetReferralKeeper() referral.Keeper         { return *app.referralKeeper }
func (app ArteryApp) GetProfileKeeper() profileKeeper.Keeper     { return app.profileKeeper }
func (app ArteryApp) GetScheduleKeeper() scheduleKeeper.Keeper   { return app.scheduleKeeper }
func (app ArteryApp) GetDelegatingKeeper() delegating.Keeper     { return *app.delegatingKeeper }
func (app ArteryApp) GetVotingKeeper() votingKeeper.Keeper       { return app.votingKeeper }
func (app ArteryApp) GetNodingKeeper() noding.Keeper             { return app.nodingKeeper }
func (app ArteryApp) GetEarningKeeper() earning.Keeper           { return app.earningKeeper }

func NewAppFromGenesis(genesis []byte) (app *ArteryApp, cleanup func(), ctx sdk.Context) {
	var logger log.Logger
	if verbose {
		logger = log.NewLogger(os.Stderr)
	} else {
		logger = log.NewNopLogger()
	}
	dir, _ := ioutil.TempDir("", "goleveldb-app-sim")
	db, _ := dbm.NewGoLevelDB("Simulation", dir, nil)

	cleanup = func() {
		_ = db.Close()
		_ = os.RemoveAll(dir)
	}

	ec := NewEncodingConfig()
	app = NewArteryApp(logger, db, nil, true, 0, ec, fauxMerkleModeOpt)

	if genesis == nil {
		cwd, err := os.Getwd()
		if err != nil {
			panic(errors.Wrap(err, "cannot get current dir"))
		}
		dir := cwd
		for dir != "" {
			var tail string
			dir, tail = path.Split(path.Clean(dir))
			if tail == "x" || tail == "app" {
				break
			}
		}
		if dir == "" {
			panic(errors.Errorf("path '%s' is out of project", cwd))
		}
		genesis, err = ioutil.ReadFile(path.Join(dir, "app", "test-genesis.json")) //TODO: Cleanup file contents
	}

	var (
		genesisDoc   tmtypes.GenesisDoc
		genesisState GenesisState
	)
	if err := tmjson.Unmarshal(genesis, &genesisDoc); err != nil {
		panic(err)
	}
	if err := tmjson.Unmarshal(genesisDoc.AppState, &genesisState); err != nil {
		panic(err)
	}

	ctx = app.NewContext(true).WithBlockTime(genesisDoc.GenesisTime).WithBlockHeight(genesisDoc.InitialHeight)
	if _, err := app.mm.InitGenesis(ctx, ec.Marshaler, genesisState); err != nil {
		panic(errors.Wrap(err, "cannot init genesis"))
	}

	return app, cleanup, ctx
}

func NewTestConsPubAddress() (crypto.PrivKey, crypto.PubKey, sdk.ConsAddress) {
	privKey := ed25519.GenPrivKey()
	pubKey := privKey.PubKey()
	addr := pubKey.Address()

	return privKey, pubKey, sdk.ConsAddress(addr.Bytes())
}

type Decoder func(bz []byte) (string, error)

func AccAddressDecoder(bz []byte) (string, error) {
	err := sdk.VerifyAddressFormat(bz)
	if err != nil {
		return "", err
	}
	return sdk.AccAddress(bz).String(), nil
}

func StringDecoder(bz []byte) (string, error) {
	if utf8.Valid(bz) {
		return string(bz), nil
	}
	return "", errors.New("non-Unicode string")
}

func Uint64Decoder(bz []byte) (string, error) {
	if len(bz) != 8 {
		return "", fmt.Errorf("wrong uint64 length")
	}
	return fmt.Sprintf("%d", binary.BigEndian.Uint64(bz)), nil
}

func DummyDecoder(_ []byte) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func ScheduleDecoder(bz []byte) (string, error) {
	var sch schedule.Schedule
	if err := sch.Unmarshal(bz); err != nil {
		return "", err
	}
	return fmt.Sprintf("%+v", sch), nil
}

func TimeDecoder(bz []byte) (string, error) {
	if len(bz) != 8 {
		return "", fmt.Errorf("wrong key length")
	}
	timestamp := binary.BigEndian.Uint64(bz)
	if timestamp > math.MaxInt64 {
		return "", fmt.Errorf("timestamp too big")
	}
	return fmt.Sprintf("%s", time.Unix(0, int64(timestamp)).String()), nil
}

func (app ArteryApp) CheckExportImport(t *testing.T, time time.Time, storeKeys []string, keyDecoders, valueDecoders map[string]Decoder, ignorePrefixes map[string][][]byte) {
	ctx := app.NewContext(true).WithBlockHeight(app.LastBlockHeight()).WithBlockTime(time)
	if _, err := app.EndBlocker(ctx); err != nil {
		t.Fatalf("EndBlocker: %v", err)
	}
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	appState, err := app.ExportAppStateAndValidators(false, nil, nil)
	assert.NoError(t, err)

	if printGenesis {
		fmt.Println(string(appState.AppState))
	}

	var logger log.Logger
	if verbose {
		logger = log.NewLogger(os.Stderr)
	} else {
		logger = log.NewNopLogger()
	}
	dir, _ := ioutil.TempDir("", "goleveldb-app-sim-2")
	db, _ := dbm.NewGoLevelDB("Simulation-2", dir, nil)

	defer func() {
		_ = db.Close()
		_ = os.RemoveAll(dir)
	}()

	app2 := NewArteryApp(logger, db, nil, true, 0, app.ec, fauxMerkleModeOpt)
	var genesisState GenesisState
	if err := tmjson.Unmarshal(appState.AppState, &genesisState); err != nil {
		panic(err)
	}
	ensureActiveValidator(genesisState)

	ctx2 := app2.NewContext(true).WithBlockHeight(app2.LastBlockHeight()).WithBlockTime(time)
	if _, err := app2.mm.InitGenesis(ctx2, app2.ec.Marshaler, genesisState); err != nil {
		t.Fatalf("InitGenesis: %v", err)
	}

	for _, key := range storeKeys {
		store1 := ctx.KVStore(app.GetKeys()[key])
		store2 := ctx2.KVStore(app2.GetKeys()[key])
		kvA, kvB := diffKVStores(store1, store2, ignorePrefixes[key])
		var triples []kvTriple
		var kvAExtra, kvBExtra []kv.Pair
		for _, a := range kvA {
			found := false
			for _, b := range kvB {
				if bytes.Equal(a.Key, b.Key) {
					triples = append(triples, kvTriple{a.Key, a.Value, b.Value})
					found = true
					break
				}
			}
			if !found {
				kvAExtra = append(kvAExtra, a)
			}
		}
		for _, b := range kvB {
			found := false
			for _, t := range triples {
				if bytes.Equal(b.Key, t.Key) {
					found = true
					break
				}
			}
			if !found {
				kvBExtra = append(kvBExtra, b)
			}
		}

		dkvA := decodeKVPairs(kvAExtra, keyDecoders[key], valueDecoders[key])
		dkvB := decodeKVPairs(kvBExtra, keyDecoders[key], valueDecoders[key])
		dTriples := decodeKVTriples(triples, keyDecoders[key], valueDecoders[key])
		assert.Empty(t, dTriples, "MUTATED pair(s) in %s kvstore", key)
		assert.Empty(t, dkvA, "VANISHED pair(s) in %s kvstore", key)
		assert.Empty(t, dkvB, "ARTIFACT pair(s) in %s kvstore", key)
	}
}

// ensureActiveValidator дописывает в выгруженный генезис активного
// валидатора, если там не осталось ни одного.
//
// С SDK 0.47 InitGenesis паникует на пустом наборе валидаторов. Часть
// тестов прокручивает время на несколько суток, единственный валидатор
// тестового генезиса за это время деактивируется, и обратный импорт
// падает — хотя проверяют эти тесты совсем другое (содержимое сторов
// referral и schedule, noding в сличение не входит).
//
// Это не только тестовая деталь: выбранный способ миграции — выгрузка
// состояния и запуск с нового генезиса — на состоянии без активных
// валидаторов теперь не заведётся. Инструмент patch-genesis решает это
// флагом --adam, оставляющим ровно одного активного.
func ensureActiveValidator(genesisState GenesisState) {
	raw, ok := genesisState[noding.ModuleName]
	if !ok {
		return
	}

	var section map[string]json.RawMessage
	if err := tmjson.Unmarshal(raw, &section); err != nil {
		panic(err)
	}

	var active, nonActive []json.RawMessage
	_ = tmjson.Unmarshal(section["active"], &active)
	_ = tmjson.Unmarshal(section["non_active"], &nonActive)
	if len(active) > 0 || len(nonActive) == 0 {
		return
	}

	// Первого отключённого делаем активным, а критерии допуска обнуляем:
	// после компрессии аккаунт им заведомо не удовлетворяет.
	section["active"] = mustMarshalJSON([]json.RawMessage{nonActive[0]})
	section["non_active"] = mustMarshalJSON(nonActive[1:])

	var params map[string]json.RawMessage
	if err := tmjson.Unmarshal(section["params"], &params); err != nil {
		panic(err)
	}
	params["min_criteria"] = json.RawMessage(
		`{"status":"STATUS_LUCKY","self_stake":"0","total_stake":"0"}`)
	section["params"] = mustMarshalJSON(params)

	genesisState[noding.ModuleName] = mustMarshalJSON(section)
}

func mustMarshalJSON(v interface{}) json.RawMessage {
	bz, err := tmjson.Marshal(v)
	if err != nil {
		panic(err)
	}
	return bz
}

//---------------------------------------------------------------------------------------------------

// Pass this in as an option to use a dbStoreAdapter instead of an IAVLStore for simulation speed.
func fauxMerkleModeOpt(bapp *baseapp.BaseApp) {
	bapp.SetFauxMerkleMode()
}

type kvTriple struct {
	Key, ValueA, ValueB []byte
}

func decodeKVPairs(kvz []kv.Pair, keyDecoder func([]byte) (string, error), valDecoder func([]byte) (string, error)) []string {
	result := make([]string, len(kvz))
	for i, kv := range kvz {
		var keyStr, valStr string

		s, err := keyDecoder(kv.Key)
		if err == nil {
			keyStr = s
		} else {
			keyStr = fmt.Sprintf("%v", kv.Key)
		}

		s, err = valDecoder(kv.Value)
		if err == nil {
			valStr = s
		} else {
			valStr = fmt.Sprintf("%v", kv.Value)
		}

		result[i] = fmt.Sprintf("\n{%s -> %s}", keyStr, valStr)
	}
	return result
}

func decodeKVTriples(kvz []kvTriple, keyDecoder func([]byte) (string, error), valDecoder func([]byte) (string, error)) []string {
	result := make([]string, len(kvz))
	for i, kv := range kvz {
		var keyStr, valStrA, valStrB string

		s, err := keyDecoder(kv.Key)
		if err == nil {
			keyStr = s
		} else {
			keyStr = fmt.Sprintf("%v", kv.Key)
		}

		s, err = valDecoder(kv.ValueA)
		if err == nil {
			valStrA = s
		} else {
			valStrA = fmt.Sprintf("%v", kv.ValueA)
		}

		s, err = valDecoder(kv.ValueB)
		if err == nil {
			valStrB = s
		} else {
			valStrB = fmt.Sprintf("%v", kv.ValueB)
		}

		result[i] = fmt.Sprintf("\n{%s -> \n\t%s <<<|>>> \n\t%s\n}", keyStr, valStrA, valStrB)
	}
	return result
}

func diffKVStores(a, b storeTypes.KVStore, ignore [][]byte) (kvAs, kvBs []kv.Pair) {
	iterA := a.Iterator(nil, nil)
	iterB := b.Iterator(nil, nil)

	for {
		shouldIgnore := func(key []byte) bool {
			for _, prefix := range ignore {
				if len(key) >= len(prefix) && bytes.Equal(prefix, key[:len(prefix)]) {
					return true
				}
			}
			return false
		}
		for iterA.Valid() && shouldIgnore(iterA.Key()) {
			iterA.Next()
		}
		for iterB.Valid() && shouldIgnore(iterB.Key()) {
			iterB.Next()
		}

		if !iterA.Valid() && !iterB.Valid() {
			break
		}

		var kvA, kvB kv.Pair
		if !iterA.Valid() {
			for ; iterB.Valid(); iterB.Next() {
				kvBs = append(kvBs, kv.Pair{Key: iterB.Key(), Value: iterB.Value()})
			}
			break
		}
		if !iterB.Valid() {
			for ; iterA.Valid(); iterA.Next() {
				kvAs = append(kvAs, kv.Pair{Key: iterA.Key(), Value: iterA.Value()})
			}
			break
		}

		kvA = kv.Pair{Key: iterA.Key(), Value: iterA.Value()}
		iterA.Next()

		kvB = kv.Pair{Key: iterB.Key(), Value: iterB.Value()}
		iterB.Next()

		if !bytes.Equal(kvA.Key, kvB.Key) || !bytes.Equal(kvA.Value, kvB.Value) {
			kvAs = append(kvAs, kvA)
			kvBs = append(kvBs, kvB)
		}
	}
	return kvAs, kvBs
}
