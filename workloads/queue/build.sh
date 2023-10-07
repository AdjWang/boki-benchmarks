#!/bin/bash
set -euo pipefail

APP_DIR="$(realpath $(dirname "$0"))"
PROJECT_DIR=$(realpath $APP_DIR/../../)
BOKI_DIR=$(realpath $PROJECT_DIR/boki)

cd $APP_DIR
go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
go mod edit -replace cs.utexas.edu/zjia/faas/slib=$BOKI_DIR/slib
go mod tidy

export CGO_ENABLED=1
go build -o bin/main main.go
go build -o bin/init_queues tools/init_queues.go
go build -o bin/benchmark tools/benchmark.go
cd -
