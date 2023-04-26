#!/bin/bash
set -euxo pipefail

BASE_DIR=`realpath $(dirname $0)`
ROOT_DIR=`realpath $BASE_DIR/../../..`
WORKLOAD_BIN_DIR=$ROOT_DIR/workloads/workflow/boki/bin

AWS_REGION=us-east-2

# EXP_DIR=$BASE_DIR/results/$1
# QPS=$2

HELPER_SCRIPT=$ROOT_DIR/scripts/exp_helper
WRK_DIR=/usr/local/bin

# TODO: strange bug: head not generating EOF and just stucks. Only on my vm, tested ok in WSL.
# TABLE_PREFIX=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 8 | head -n 1 || true)
TABLE_PREFIX=$(echo $RANDOM | md5sum | head -c8)
TABLE_PREFIX="${TABLE_PREFIX}-"

TABLE_PREFIX=$TABLE_PREFIX docker compose up -d

TABLE_PREFIX=$TABLE_PREFIX AWS_REGION=$AWS_REGION \
    $WORKLOAD_BIN_DIR/singleop/init create cayon
TABLE_PREFIX=$TABLE_PREFIX AWS_REGION=$AWS_REGION \
    $WORKLOAD_BIN_DIR/singleop/init populate cayon

# curl -X POST -d "{}" http://127.0.0.1:9000/function/singleop
# ENDPOINT=http://localhost:9000/function/singleop wrk -t 2 -c 2 -d 3 -L -U -s ./workloads/workflow/boki/benchmark/singleop/workload.lua http://127.0.0.1:9000 -R 5
# p50
# egrep -o 'latencyRead.*:(\w+)' wrkresp.log | tr ':' ' ' | awk '{print $2}' | xargs python3 -c "import sys;l=[int(i) for i in sys.argv[1:]];print(sorted(l)[len(l)//2])"

# clean up env
# TABLE_PREFIX=$TABLE_PREFIX AWS_REGION=$AWS_REGION \
#     $WORKLOAD_BIN_DIR/init clean cayon
