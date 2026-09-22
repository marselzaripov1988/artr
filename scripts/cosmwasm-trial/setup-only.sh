#!/bin/bash
# Готовит сеть и НЕ отправляет ни одной транзакции.
#
# Нужно именно так: клиент не умеет читать последовательность счёта, и
# каждая лишняя транзакция в подготовке сбивает нумерацию в проверке.
# Начав с чистого счёта, можно вести счётчик от нуля и не гадать.
set -uo pipefail

export LD_LIBRARY_PATH=/work
export ARTR_HOME=/root/.artrd
BIN=/work/artrd-wasm
CHAIN=wasm-trial-1

rm -rf "$ARTR_HOME"
$BIN init trial --chain-id "$CHAIN" --home "$ARTR_HOME" >/dev/null 2>&1

WASM_DEFAULT=$(jq -c '.app_state.wasm' "$ARTR_HOME/config/genesis.json")

$BIN keys add dev --keyring-backend test --home "$ARTR_HOME" --output json > /tmp/k.raw 2>&1
sed -n '/^{/,/^}/p' /tmp/k.raw > /tmp/k.json
DEV=$(jq -r .address /tmp/k.json)

VALPUB=$(python3 /devnet/conspub.py "$ARTR_HOME/config/priv_validator_key.json")
TOP=$(jq -r '.app_state.referral.top_level_accounts[0]' /template/test-genesis.json)
ACCNUM=$(jq '.app_state.auth.accounts | length' /template/test-genesis.json)
TS=$(date -u +%Y-%m-%dT%H:%M:%SZ)

jq --arg chain "$CHAIN" --arg ts "$TS" --arg valpub "$VALPUB" --arg dev "$DEV" \
   --arg top "$TOP" --arg accnum "$ACCNUM" --argjson wasm "$WASM_DEFAULT" '
    .chain_id = $chain
  | .genesis_time = $ts
  | .initial_height = "1"
  | .app_state.noding.active[0].pub_key = $valpub
  | .app_state.noding.params.min_criteria = {"status":"STATUS_LUCKY","self_stake":"0","total_stake":"0"}
  | .app_state.auth.accounts += [{"@type":"/cosmos.auth.v1beta1.BaseAccount","address":$dev,"accountNumber":$accnum,"sequence":"0"}]
  | .app_state.artrbank.balances += [{"address":$dev,"coins":[{"denom":"uartr","amount":"100000000000000"}]}]
  | .app_state.referral.other_accounts |= map(if .referrer == $top then .referrals += [$dev] else . end)
  | .app_state.voting.government = [$dev]
  | .app_state.profile.profiles += [{"address":$dev,"profile":{"active_until":"2100-01-01T00:00:00Z","nickname":"dev"}}]
  | .app_state.wasm = $wasm
  ' /template/test-genesis.json > "$ARTR_HOME/config/genesis.json"

$BIN validate-genesis --home "$ARTR_HOME" 2>&1 | tail -1

cat > "$ARTR_HOME/config/client.toml" <<EOF
chain-id = "$CHAIN"
keyring-backend = "test"
output = "json"
node = "tcp://localhost:26657"
broadcast-mode = "sync"
EOF
sed -i 's|^minimum-gas-prices = .*|minimum-gas-prices = "0uartr"|' "$ARTR_HOME/config/app.toml"
sed -i 's|^timeout_commit =.*|timeout_commit = "2s"|' "$ARTR_HOME/config/config.toml"

# Получателем берём счёт, который УЖЕ существует в генезисе: банк Artery
# отказывает в переводе на незаведённый счёт, а заводить его отдельной
# транзакцией значит снова считать последовательность.
jq -r '.app_state.auth.accounts[0].address' /template/test-genesis.json > /tmp/bob.addr

echo "подготовлено, транзакций не отправлено"
