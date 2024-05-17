#!/bin/bash

BASE_DIR="$(realpath $(dirname "$0"))"
mkdir -p $BASE_DIR/bin

BOKI_DIR=$1
if [ -z "$BOKI_DIR" ]; then
    BOKI_DIR="/src/boki"
fi
# remove trailing slash
# BOKI_DIR=$(echo "$BOKI_DIR" | sed 's:/*$::')
BOKI_DIR=$(realpath "$BOKI_DIR")
go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
go mod edit -replace cs.utexas.edu/zjia/faas/slib=$BOKI_DIR/slib

export CGO_ENABLED=0

( cd $BASE_DIR && \
    go build -buildvcs=false -o bin/main main.go && \
    go build -buildvcs=false -o bin/init_queues tools/init_queues.go && \
    go build -buildvcs=false -o bin/benchmark tools/benchmark.go
)
