#!/bin/bash
# Проверка исходящего перевода из контракта на заданную сумму.
#
# hackatom при release отдаёт весь баланс и потому упирается в комиссию
# Artery. reflect умеет отправить ровно столько, сколько скажут, — на
# нём видно, работает ли перевод, когда на комиссию оставлен запас.
#
# Счёт dev чистый (подготовка не слала транзакций), поэтому
# последовательность ведётся от нуля.
set -uo pipefail

export LD_LIBRARY_PATH=/work
export ARTR_HOME=/root/.artrd
BIN=/work/artrd-wasm
Q="--home $ARTR_HOME"
SEQ=0

step() { echo; echo "############ $* ############"; }

DEV=$($BIN keys show dev -a --keyring-backend test --home "$ARTR_HOME")
BOB=$(cat /tmp/bob.addr)

step "запуск узла"
$BIN start --home "$ARTR_HOME" > /tmp/node.log 2>&1 &
NODE=$!
for i in $(seq 1 40); do
    H=$(curl -s localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height // empty')
    [ -n "$H" ] && [ "$H" != "0" ] && break
    sleep 2
done
[ -z "${H:-}" ] && { echo "узел не поднялся"; tail -20 /tmp/node.log; exit 1; }
echo "высота $H"
echo "dev = $DEV"
echo "bob = $BOB (счёт из генезиса)"
echo -n "баланс bob до: "; $BIN query artrbank balance "$BOB" $Q 2>&1 | head -1

send() {
    local what="$1"; shift
    echo "--- $what (последовательность $SEQ) ---"
    $BIN tx wasm "$@" --from dev --keyring-backend test --home "$ARTR_HOME" \
        --account-number 0 --sequence "$SEQ" --gas 30000000 --fees 0uartr -y 2>&1 | head -3 > /tmp/tx.json
    sleep 7
    local h; h=$(jq -r '.txhash // empty' /tmp/tx.json 2>/dev/null)
    if [ -n "$h" ]; then
        $BIN query tx "$h" $Q 2>&1 | jq -c '{code, log: (.raw_log|tostring|.[0:260])}' 2>/dev/null
    else
        cat /tmp/tx.json
    fi
    SEQ=$((SEQ + 1))
}

step "1. загружаю reflect"
send "store" store /work/reflect.wasm
CODE=$($BIN query wasm list-code $Q 2>&1 | jq -r '.code_infos[-1].code_id // empty')
echo "код: ${CODE:-НЕТ}"
[ -z "${CODE:-}" ] && { kill $NODE; exit 1; }

step "2. экземпляр с 1 ARTR"
send "instantiate" instantiate "$CODE" '{}' --amount 1000000uartr --label reflect --no-admin
ADDR=$($BIN query wasm list-contract-by-code "$CODE" $Q 2>&1 | jq -r '.contracts[0] // empty')
echo "контракт: ${ADDR:-НЕ СОЗДАН}"
[ -z "${ADDR:-}" ] && { kill $NODE; exit 1; }
echo -n "баланс контракта: "; $BIN query artrbank balance "$ADDR" $Q 2>&1 | head -1

step "3. КОНТРАКТ ОТПРАВЛЯЕТ 900000uartr, оставив запас на комиссию"
MSG=$(jq -nc --arg to "$BOB" '{reflect_msg:{msgs:[{bank:{send:{to_address:$to, amount:[{denom:"uartr", amount:"900000"}]}}}]}}')
send "execute" execute "$ADDR" "$MSG"

step "4. деньги дошли?"
echo -n "контракт: "; $BIN query artrbank balance "$ADDR" $Q 2>&1 | head -1
echo -n "bob:      "; $BIN query artrbank balance "$BOB" $Q 2>&1 | head -1

step "итог"
echo "паник: $(grep -ci panic /tmp/node.log)"
kill $NODE 2>/dev/null
