#!/bin/bash
set -euo pipefail

ASYNC_BENCH=0

BUILD_TYPE="Release"
while [ ! $# -eq 0 ]
do
  case "$1" in
    --debug)
      BUILD_TYPE="Debug"
      ;;
  esac
  shift
done

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

export CGO_ENABLED=1
CGO_CFLAGS="$(go env CGO_CFLAGS) -I$BOKI_DIR/lib/shared_index/include"
if [[ $BUILD_TYPE == "Debug" ]]; then
    CGO_LDFLAGS="$(go env CGO_LDFLAGS) -L$BOKI_DIR/lib/shared_index/bin/debug -lrt -ldl -lindex"
else
    CGO_LDFLAGS="$(go env CGO_LDFLAGS) -L$BOKI_DIR/lib/shared_index/bin/release -lrt -ldl -lindex"
fi

cd $APP_DIR
go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
go mod edit -replace cs.utexas.edu/zjia/faas/slib=$BOKI_DIR/slib
go mod tidy
CGO_CFLAGS=$CGO_CFLAGS CGO_LDFLAGS=$CGO_LDFLAGS go build -o bin/main main.go
go build -o bin/create_users tools/create_users.go
go build -o bin/benchmark tools/benchmark.go
cd -
