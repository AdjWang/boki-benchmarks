#!/bin/bash
set -euo pipefail

ASYNC_BENCH=0

# CONSISTENCY=SEQUENTIAL
CONSISTENCY=STRONG

# TXN_CHECK_METHOD=APPEND
TXN_CHECK_METHOD=CHECKSEQ

SW_STAT=ON
# SW_STAT=OFF

APP_DIR="$(realpath $(dirname "$0"))"
PROJECT_DIR=$(realpath $APP_DIR/../../)
BOKI_DIR=$(realpath $PROJECT_DIR/boki)

if [[ $ASYNC_BENCH == 1 ]]; then
    find $APP_DIR/handlers -name '*.go' | \
    xargs sed -i 's/\"cs.utexas.edu\/zjia\/faas\/slib\/statestore\"/statestore \"cs.utexas.edu\/zjia\/faas\/slib\/asyncstatestore\"/g'
else
    find $APP_DIR/handlers -name '*.go' | \
    xargs sed -i 's/statestore \"cs.utexas.edu\/zjia\/faas\/slib\/asyncstatestore\"/\"cs.utexas.edu\/zjia\/faas\/slib\/statestore\"/g'
fi

cd $APP_DIR
go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
go mod edit -replace cs.utexas.edu/zjia/faas/slib=$BOKI_DIR/slib
go mod tidy

export CGO_ENABLED=1
go build -ldflags="-X cs.utexas.edu/zjia/faas/slib/common.CONSISTENCY=$CONSISTENCY \
                   -X cs.utexas.edu/zjia/faas/slib/common.TXN_CHECK_METHOD=$TXN_CHECK_METHOD \
                   -X cs.utexas.edu/zjia/faas/slib/common.SW_STAT=$SW_STAT" \
         -o bin/main main.go
go build -o bin/create_users tools/create_users.go
go build -o bin/benchmark tools/benchmark.go
cd -
