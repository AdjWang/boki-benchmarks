#!/bin/bash
set -euxo pipefail

SCRIPT_DIR=$(dirname $0)
WORKFLOW_DIR=$(realpath $SCRIPT_DIR/../..)

TABLE_PREFIX=$(echo $RANDOM | md5sum | head -c8)
TABLE_PREFIX="${TABLE_PREFIX}-"

mkdir -p /tmp/boki
echo "http://localhost:8000" > /tmp/boki/dbendpoint

MOVIE_DATA_FILE=$WORKFLOW_DIR/internal/media/data/compressed.json
echo "========= init db ========="
TABLE_PREFIX=$TABLE_PREFIX DBENV=REMOTE go run $WORKFLOW_DIR/internal/media/init/init.go create cayon
TABLE_PREFIX=$TABLE_PREFIX DBENV=REMOTE go run $WORKFLOW_DIR/internal/media/init/init.go populate cayon $MOVIE_DATA_FILE
TABLE_PREFIX=$TABLE_PREFIX DBENV=REMOTE go run $WORKFLOW_DIR/internal/media/init/init.go health_check

# echo "========= run benchmark ========="
# # TABLE_PREFIX=$TABLE_PREFIX DBENV=REMOTE DATA=$MOVIE_DATA_FILE go run $WORKFLOW_DIR/cmd/bench/db_bench.go \
# #                                                                      $WORKFLOW_DIR/cmd/bench/data.go &
# TABLE_PREFIX=$TABLE_PREFIX DBENV=REMOTE DATA=$MOVIE_DATA_FILE go run $WORKFLOW_DIR/cmd/bench/db_bench.go \
#                                                                      $WORKFLOW_DIR/cmd/bench/data.go

# echo "========= clean db ========="
# TABLE_PREFIX=$TABLE_PREFIX DBENV=REMOTE go run $WORKFLOW_DIR/internal/media/init/init.go clean cayon
