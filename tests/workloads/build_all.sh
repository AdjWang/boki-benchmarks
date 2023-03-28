#!/bin/bash
set -euo pipefail

BASE_DIR="$(realpath $(dirname "$0"))"
WORKLOAD_DIR=$BASE_DIR

function build_sharedlog {
    echo "build sharedlog"
    cd $WORKLOAD_DIR/sharedlog
    ./build.sh
}

build_sharedlog
