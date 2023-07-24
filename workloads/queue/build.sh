#!/bin/bash
set -euo pipefail

ASYNC_BENCH=1

APP_DIR="$(realpath $(dirname "$0"))"
PROJECT_DIR=$(realpath $APP_DIR/../../)
BOKI_DIR=$(realpath $PROJECT_DIR/boki)

if [[ $ASYNC_BENCH == 1 ]]; then
    find $APP_DIR/handlers -name '*.go' | \
    xargs sed -i 's/\"cs.utexas.edu\/zjia\/faas\/slib\/sync\"/sync \"cs.utexas.edu\/zjia\/faas\/slib\/asyncqueue\"/g'
else
    find $APP_DIR/handlers -name '*.go' | \
    xargs sed -i 's/sync \"cs.utexas.edu\/zjia\/faas\/slib\/asyncqueue\"/\"cs.utexas.edu\/zjia\/faas\/slib\/sync\"/g'
fi

cd $APP_DIR
go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
go mod edit -replace cs.utexas.edu/zjia/faas/slib=$BOKI_DIR/slib
go mod tidy

export CGO_ENABLED=0
go build -o bin/main main.go
go build -o bin/init_queues tools/init_queues.go
go build -o bin/benchmark tools/benchmark.go
cd -
