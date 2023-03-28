#!/bin/bash
# set -euxo pipefail

ROOT_DIR="$(realpath $(dirname "$0")/../../)"
BOKI_DIR=$ROOT_DIR/boki
WORKLOADS_DIR=$ROOT_DIR/workloads

function build_goexample {
    cd $WORKLOADS_DIR/goexample

    go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang

    export CGO_ENABLED=0
    go build -o bin/main main.go
}

function build_queue {
    cd $WORKLOADS_DIR/queue

    go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
    go mod edit -replace cs.utexas.edu/zjia/faas/slib=$BOKI_DIR/slib

    export CGO_ENABLED=0
    go build -o bin/main main.go
    go build -o bin/init_queues tools/init_queues.go
    go build -o bin/benchmark tools/benchmark.go
}

function build_retwis {
    cd $WORKLOADS_DIR/retwis

    go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
    go mod edit -replace cs.utexas.edu/zjia/faas/slib=$BOKI_DIR/slib

    export CGO_ENABLED=0
    go build -o bin/main main.go
    go build -o bin/create_users tools/create_users.go
    go build -o bin/benchmark tools/benchmark.go
}

function build_workflow {
    cd $WORKLOADS_DIR/workflow/beldi
    go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
    make hotel-baseline
    make media-baseline
    make hotel
    make media

    cd $WORKLOADS_DIR/workflow/boki
    go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
    make hotel
    make media
    make singleop
}

build_goexample
build_queue
build_retwis
build_workflow
