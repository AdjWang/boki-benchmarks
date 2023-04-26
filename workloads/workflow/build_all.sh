#!/bin/bash
set -euo pipefail

function print_usage {
    echo "usage: build_all.sh LOCAL|REMOTE"
    echo "LOCAL: connect to local dynamodb for debugging."
    echo "REMOTE: connect to remote dynamodb for evaluating."
}
if [ $# -eq 0 ]; then
    echo "[ERROR] not enough arguments"
    print_usage
    exit 1
fi

DBENV=$1
if ! [[ "$DBENV" =~ ^(LOCAL|REMOTE)$ ]]; then
    echo "[ERROR] invalid argument"
    print_usage
    exit 1
fi

PROJECT_DIR="$(realpath $(dirname "$0")/../..)"
WORKFLOW_DIR=$PROJECT_DIR/workloads/workflow
BOKI_DIR=$PROJECT_DIR/boki

function build_beldi {
    cd $WORKFLOW_DIR/beldi
    go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
    go mod tidy
    make hotel-baseline
    make media-baseline
    make hotel
    make media
}

function build_boki {
    cd $WORKFLOW_DIR/boki
    go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
    go mod tidy
    make hotel
    make media
    make singleop
}

function build_asynclog {
    cd $WORKFLOW_DIR/asynclog
    go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
    go mod tidy
    # make hotel
    # make hotel-baseline
    # make media
    # make media-baseline
    # make singleop
    make all -j$(nproc) DBENV=$DBENV
}

build_beldi
build_boki
build_asynclog
