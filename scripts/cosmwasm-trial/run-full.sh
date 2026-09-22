#!/bin/bash
# Весь цикл за один запуск узла: загрузка кода, экземпляр с монетами,
# запрос к контракту, вызов с обратным переводом.
#
# Последовательность счёта ведём сами. Причина в отдельной находке:
# клиент её не определяет (см. шаг «что клиент знает о счёте»), и каждая
# вторая транзакция подряд отваливается по подписи. К CosmWasm это
# отношения не имеет, но мешает проверке, поэтому обходим явно.
set -uo pipefail

export LD_LIBRARY_PATH=/work
export ARTR_HOME=/root/.artrd
BIN=/work/artrd-wasm
Q="--home $ARTR_HOME"
SEQ=0

step() { echo; echo "############ $* ############"; }

DEV=$($BIN keys show dev -a --keyring-backend test --home "$ARTR_HOME")

step "запуск узла"
$BIN start --home "$ARTR_HOME" > /tmp/node.log 2>&1 &
NODE=$!
for i in $(seq 1 40); do
    H=$(curl -s localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height // empty')
    [ -n "$H" ] && [ "$H" != "0" ] && break
    sleep 2
done
[ -z "${H:-}" ] && { echo "узел не поднялся"; tail -20 /tmp/node.log; exit 1; }
echo "высота $H, dev = $DEV"

step "что клиент знает о счёте"
echo "-- есть ли команда query auth account:"
$BIN query auth account "$DEV" $Q 2>&1 | head -4
echo "-- прямой запрос к службе счетов через REST:"
curl -s "localhost:1317/cosmos/auth/v1beta1/accounts/$DEV" 2>&1 | head -c 300
echo

send() {
    local what="$1"; shift
    echo "--- $what (последовательность $SEQ) ---"
    $BIN tx wasm "$@" \
        --from dev --keyring-backend test --home "$ARTR_HOME" \
        --account-number 0 --sequence "$SEQ" \
        --gas 30000000 --fees 0uartr -y 2>&1 | head -3 > /tmp/tx.json
    jq -c '{code, raw_log: (.raw_log|tostring|.[0:200]), txhash}' /tmp/tx.json 2>/dev/null || cat /tmp/tx.json
    sleep 7
    local h; h=$(jq -r '.txhash // empty' /tmp/tx.json 2>/dev/null)
    if [ -n "$h" ]; then
        $BIN query tx "$h" $Q 2>&1 | jq -c '{code, log: (.raw_log|tostring|.[0:250])}' 2>/dev/null
    fi
    SEQ=$((SEQ + 1))
}

step "1. загрузка кода"
send "store" store /work/contract.wasm

step "2. экземпляр С ПЕРЕВОДОМ СРЕДСТВ"
INIT=$(jq -nc --arg v "$DEV" --arg b "$DEV" '{verifier:$v, beneficiary:$b}')
send "instantiate" instantiate 1 "$INIT" --amount 1000000uartr --label proba --no-admin

step "3. адрес экземпляра"
ADDR=$($BIN query wasm list-contract-by-code 1 $Q 2>&1 | jq -r '.contracts[0] // empty' 2>/dev/null)
echo "контракт: ${ADDR:-НЕ СОЗДАН}"

if [ -n "${ADDR:-}" ]; then
    step "4. монеты дошли до контракта?"
    $BIN query artrbank balance "$ADDR" $Q 2>&1 | head -3

    step "5. запрос к контракту — работа самого движка"
    $BIN query wasm contract-state smart "$ADDR" '{"verifier":{}}' $Q 2>&1 | head -3

    step "6. вызов release — контракт отдаёт монеты сам"
    send "execute" execute "$ADDR" '{"release":{}}'

    step "7. баланс контракта после release"
    $BIN query artrbank balance "$ADDR" $Q 2>&1 | head -3
fi

step "итог"
echo "блоков: $(grep -c 'committed state' /tmp/node.log)"
echo "паник: $(grep -ci panic /tmp/node.log)"
grep -i panic /tmp/node.log | head -3
kill $NODE 2>/dev/null
