#!/usr/bin/env bash
#
# Генерация protobuf.
#
# Запускается внутри образа из scripts/protocgen.Dockerfile — там стоят
# protoc и плагины тех версий, что закреплены в go.mod. Своими руками на
# хосте это ставить не нужно:
#
#   make proto-gen
#
# Прежний вариант скрипта не работал: он требовал в go.mod замены
# gogo/protobuf на форк regen-network, которой у нас нет с переезда на
# cosmos/gogoproto, и звал `buf protoc` — подкоманду, убранную в buf v1.
#
# От buf отказались сознательно. Здесь он давал только проверку на
# несовместимые изменения (buf breaking) против снимков в proto/*.bin, а
# его конфигурация осталась версии v1beta1 и требует отдельной миграции.
# Генерацию же он выполнял тем же protoc — его и зовём напрямую.
set -euo pipefail

# Плагин gocosmos понимает gogoproto.customtype: им в Artery объявлены
# денежные типы, и без него math.Int превращается в обычную строку.
#
# Mgoogle/protobuf/any.proto — сопоставление, без которого Any берётся из
# стандартной библиотеки protobuf вместо codec/types SDK, и реестр
# интерфейсов перестаёт разбирать сообщения.
GOCOSMOS_OPTS=$(cat <<'EOF'
plugins=interfacetype+grpc,Mgoogle/protobuf/any.proto=github.com/cosmos/cosmos-sdk/codec/types:.
EOF
)

proto_dirs=$(find ./proto -name '*.proto' -print0 | xargs -0 -n1 dirname | sort -u)

for dir in $proto_dirs; do
    protoc \
        -I proto \
        -I third_party/proto \
        --gocosmos_out="$GOCOSMOS_OPTS" \
        --grpc-gateway_out=logtostderr=true:. \
        $(find "${dir}" -maxdepth 1 -name '*.proto')
done

# protoc раскладывает результат по полному пути пакета из go_package.
# Переносим на место и убираем пустой каркас каталогов.
if [ -d github.com/arterynetwork/artr ]; then
    cp -r github.com/arterynetwork/artr/* ./
    rm -rf github.com
fi

echo "==> готово"
