#!/bin/bash
set -euo pipefail

PROJECT_DIR="$(realpath $(dirname "$0")/../..)"
BOKI_DIR=$(realpath $PROJECT_DIR/boki)
BENCH_DIR=$(realpath $PROJECT_DIR/workloads/microbenchmark)
WORKFLOW_DIR=$(realpath $PROJECT_DIR/workloads/workflow)

export CGO_ENABLED=0

cd $BENCH_DIR
go mod edit -replace=cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
go mod edit -replace github.com/eniac/Beldi=$WORKFLOW_DIR/asynclog
go mod tidy
go build -o bin/ ./cmd/...
go build -o bin/benchmark tools/benchmark.go
cd -
