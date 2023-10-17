#!/bin/bash
set -euo pipefail

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

PROJECT_DIR="$(realpath $(dirname "$0")/../..)"
WORKFLOW_DIR=$PROJECT_DIR/workloads/workflow
BOKI_DIR=$PROJECT_DIR/boki

function build_beldi {
    cd $WORKFLOW_DIR/beldi
    go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
    go mod tidy
    ./build.sh $BUILD_TYPE
}

function build_boki {
    cd $WORKFLOW_DIR/boki
    go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
    go mod tidy
    ./build.sh $BUILD_TYPE
}

function build_asynclog {
    cd $WORKFLOW_DIR/asynclog
    go mod edit -replace cs.utexas.edu/zjia/faas=$BOKI_DIR/worker/golang
    go mod tidy
    ./build.sh $BUILD_TYPE
}

build_beldi
build_boki
build_asynclog
