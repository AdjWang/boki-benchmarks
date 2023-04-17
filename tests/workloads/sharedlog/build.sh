#!/bin/bash
set -euo pipefail

BASE_DIR="$(realpath $(dirname "$0"))"
BOKI_DIR=$(realpath $BASE_DIR/../../../boki)
WORKLOAD_DIR=$(realpath $BASE_DIR/..)

mkdir -p $BASE_DIR/bin

export CGO_ENABLED=0

cd $BASE_DIR
go mod edit -replace=cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
go mod edit -replace github.com/eniac/Beldi=$WORKLOAD_DIR/bokiflow
go mod tidy
go build -o bin/ ./cmd/...
