#!/bin/bash

BASE_DIR="$(realpath $(dirname "$0"))"
mkdir -p $BASE_DIR/bin

export CGO_ENABLED=0

( cd $BASE_DIR && \
    go build -buildvcs=false -o bin/main main.go && \
    go build -buildvcs=false -o bin/create_users tools/create_users.go && \
    go build -buildvcs=false -o bin/benchmark tools/benchmark.go \
)
