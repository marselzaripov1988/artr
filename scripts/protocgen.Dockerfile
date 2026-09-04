# Образ для генерации protobuf.
#
# Собственный, а не готовый ghcr.io/cosmos/proto-builder: тот недоступен
# из этой сети (403 на выдаче токена), да и версии плагинов в нём свои, а
# нам нужны ровно те, что стоят в go.mod — сгенерированный код обязан
# сходиться с тем, против чего собирается проект.
#
#   docker build -f scripts/protocgen.Dockerfile -t artr-protogen .
#   docker run --rm -v "$PWD:/src" -w /src artr-protogen sh scripts/protocgen.sh

ARG GO_VERSION=1.23

FROM golang:${GO_VERSION}-alpine

RUN apk add --no-cache protoc git bash

# Версии закреплены по go.mod проекта.
#
# gocosmos — генератор gogo-совместимого кода из cosmos/gogoproto. Именно
# он понимает gogoproto.customtype, которым в Artery объявлены денежные
# типы, и подставляет cosmossdk.io/math.Int.
ARG GOGOPROTO_VERSION=v1.7.0
ARG GRPC_GATEWAY_VERSION=v1.16.0

RUN go install github.com/cosmos/gogoproto/protoc-gen-gocosmos@${GOGOPROTO_VERSION} && \
    go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@${GRPC_GATEWAY_VERSION}

# Стандартные описания (google/protobuf/*.proto) в alpine-пакете protoc не
# поставляются, а без descriptor.proto не разбирается ни один файл с
# расширениями gogoproto. Берём их из самого gogoproto и кладём туда, где
# protoc ищет включения по умолчанию.
RUN mkdir -p /usr/include/google &&     cp -r /go/pkg/mod/github.com/cosmos/gogoproto@${GOGOPROTO_VERSION}/protobuf/google/.           /usr/include/google/

ENV PATH="/go/bin:${PATH}"
