#!/bin/bash
set -euo pipefail

BASE_DIR="$(realpath $(dirname "$0"))"
WORKLOAD_DIR=$BASE_DIR
BOKI_DIR=$(realpath $BASE_DIR/../../boki)

function build_sharedlog {
    echo "build sharedlog"
    cd $WORKLOAD_DIR/sharedlog
    ./build.sh
}

function build_bokiflow {
    echo "build bokiflow"
    cd $WORKLOAD_DIR/bokiflow
    go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
    go mod tidy
    make hotel
    make media
    make singleop
}

build_sharedlog
build_bokiflow
