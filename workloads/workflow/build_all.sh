#!/bin/bash
set -euo pipefail

PROJECT_DIR="$(realpath $(dirname "$0")/../..)"
WORKFLOW_DIR=$PROJECT_DIR/workloads/workflow
BOKI_DIR=$PROJECT_DIR/boki

function build_beldi {
    cd $WORKFLOW_DIR/beldi
    go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
    go mod edit -replace cs.utexas.edu/zjia/faas/slib=$BOKI_DIR/slib
    go mod tidy
    make hotel-baseline media-baseline hotel media -j$(nproc)
}

function build_boki {
    cd $WORKFLOW_DIR/boki
    go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
    go mod edit -replace cs.utexas.edu/zjia/faas/slib=$BOKI_DIR/slib
    go mod tidy
    make hotel media singleop finra -j$(nproc)
}

function build_asynclog {
    cd $WORKFLOW_DIR/asynclog
    go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
    go mod edit -replace cs.utexas.edu/zjia/faas/slib=$BOKI_DIR/slib
    go mod tidy
    # make hotel
    # make hotel-baseline
    # make media
    # make media-baseline
    # make singleop
    make all -j$(nproc)
}

function build_optimal {
    cd $WORKFLOW_DIR/optimal
    go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
    go mod tidy
    make hotel media singleop -j$(nproc)
}

# DEBUG: test optimal
# build_beldi
# build_boki
# build_asynclog
build_optimal
