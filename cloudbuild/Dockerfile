FROM golang:1.21.3 AS build

ARG VERSION

COPY . /go/src/github.com/jmakaron/compman
WORKDIR /go/src/github.com/jmakaron/compman

RUN \ 
    mkdir -p bin && make compman

FROM debian:12-slim

RUN \
    apt update -y && \
    apt upgrade -y && \
    apt install ca-certificates curl less mime-support openssl tzdata tar && \
    apt clean && rm -rf /var/lib/apt/lists/*

COPY --from=build /go/src/github.com/jmakaron/compman/bin /compman

ENTRYPOINT [ "/compman/compman" ]