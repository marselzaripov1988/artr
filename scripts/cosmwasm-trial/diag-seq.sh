#!/bin/bash
# Диагностика: почему клиент не определяет последовательность счёта.
#
# Проверяем три вещи по отдельности, чтобы не спутать причину со
# следствием:
#   1. отвечает ли служба счетов по REST;
#   2. отвечает ли она по gRPC через ABCI (этим путём ходит сам клиент);
#   3. что происходит с двумя транзакциями подряд без --sequence.
set -uo pipefail

export LD_LIBRARY_PATH=/work
export ARTR_HOME=/root/.artrd
BIN=/work/artrd-wasm
Q="--home $ARTR_HOME"

step() { echo; echo "############ $* ############"; }

# API наружу для проверки REST: в подготовке он выключен.
awk '
    /^\[/            { section = $0 }
    /^enable = false/ && (section == "[api]" || section == "[grpc]") {
                       print "enable = true"; next }
                     { print }
' "$ARTR_HOME/config/app.toml" > /tmp/app.toml && mv /tmp/app.toml "$ARTR_HOME/config/app.toml"
sed -i 's|^address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:1317"|' "$ARTR_HOME/config/app.toml"

DEV=$($BIN keys show dev -a --keyring-backend test --home "$ARTR_HOME")

$BIN start --home "$ARTR_HOME" > /tmp/node.log 2>&1 &
NODE=$!
for i in $(seq 1 40); do
    H=$(curl -s localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height // empty')
    [ -n "$H" ] && [ "$H" != "0" ] && break
    sleep 2
done
[ -z "${H:-}" ] && { echo "узел не поднялся"; tail -20 /tmp/node.log; exit 1; }
echo "высота $H, dev = $DEV"

step "1. служба счетов по REST"
curl -s "localhost:1317/cosmos/auth/v1beta1/accounts/$DEV" | head -c 500
echo

step "2. она же через ABCI (этим путём ходит клиент)"
# Клиент зовёт cosmos.auth.v1beta1.Query/Account абцишным запросом.
curl -s "localhost:26657/abci_query?path=%22/cosmos.auth.v1beta1.Query/Account%22" | jq -c '.result.response | {code, log, codespace}' 2>&1 | head -3

step "3. список зарегистрированных служб"
curl -s "localhost:1317/cosmos/base/tendermint/v1beta1/node_info" >/dev/null 2>&1 \
  && echo "REST-шлюз жив" || echo "REST-шлюз молчит"

step "4. две транзакции подряд БЕЗ --sequence"
echo "--- первая ---"
$BIN tx wasm store /work/reflect.wasm --from dev --keyring-backend test $Q \
    --gas 30000000 --fees 0uartr -y 2>&1 | head -3 | jq -c '{code, raw_log: (.raw_log|tostring|.[0:160])}' 2>/dev/null
sleep 7
echo "--- вторая ---"
$BIN tx wasm store /work/contract.wasm --from dev --keyring-backend test $Q \
    --gas 30000000 --fees 0uartr -y 2>&1 | head -3 | jq -c '{code, raw_log: (.raw_log|tostring|.[0:160])}' 2>/dev/null
sleep 7

step "5. что в состоянии на самом деле"
curl -s "localhost:1317/cosmos/auth/v1beta1/accounts/$DEV" | head -c 500
echo

kill $NODE 2>/dev/null
