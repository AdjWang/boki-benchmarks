#!/bin/bash
set -uxo pipefail

BASE_DIR=`realpath $(dirname $0)`
ROOT_DIR=`realpath $BASE_DIR/../../..`

EXP_DIR=$BASE_DIR/results/$1

NUM_SHARDS=$2
INTERVAL1=$3
INTERVAL2=$4
NUM_PRODUCER=$5
NUM_CONSUMER=$NUM_SHARDS

HELPER_SCRIPT=$ROOT_DIR/scripts/exp_helper

# MANAGER_HOST=`$HELPER_SCRIPT get-docker-manager-host --base-dir=$BASE_DIR`
# CLIENT_HOST=`$HELPER_SCRIPT get-client-host --base-dir=$BASE_DIR`
# ENTRY_HOST=`$HELPER_SCRIPT get-service-host --base-dir=$BASE_DIR --service=boki-gateway`
# ALL_HOSTS=`$HELPER_SCRIPT get-all-server-hosts --base-dir=$BASE_DIR`

ENTRY_HOST="0.0.0.0"
# TODO: strange bug: head not generating EOF and just stucks. Only on my vm, tested ok in WSL.
# QUEUE_PREFIX=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 8 | head -n 1)
QUEUE_PREFIX=$(echo $RANDOM | md5sum | head -c8)

if [[ -d $EXP_DIR ]]; then
    rm -rf $EXP_DIR
fi
mkdir -p $EXP_DIR

$ROOT_DIR/workloads/queue/bin/benchmark \
    --faas_gateway=$ENTRY_HOST:9000 --fn_prefix=slib \
    --queue_prefix=$QUEUE_PREFIX --num_queues=1 --queue_shards=$NUM_SHARDS \
    --num_producer=$NUM_PRODUCER --num_consumer=$NUM_CONSUMER \
    --producer_interval=$INTERVAL1 --consumer_interval=$INTERVAL2 \
    --consumer_fix_shard=true \
    --payload_size=1024 --duration=10 >$EXP_DIR/results.log

# $HELPER_SCRIPT collect-container-logs --base-dir=$BASE_DIR --log-path=$EXP_DIR/logs
