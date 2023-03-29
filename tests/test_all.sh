#!/bin/bash
set -euo pipefail

TEST_DIR="$(realpath $(dirname "$0"))"
BOKI_DIR=$(realpath $TEST_DIR/../boki)
DOCKERFILE_DIR=$TEST_DIR/dockerfiles
WORKLOAD_DIR=$TEST_DIR/workloads

WORK_DIR=/tmp/boki-test

DOCKER_BUILDER=$HOME/.docker/cli-plugins/docker-buildx
NO_CACHE=""

function setup_env {
    METALOG_REPLICATION=$1
    USERLOG_REPLICATION=$2
    INDEX_REPLICATION=$3
    TEST_CASE=$4

    # remove old files and folders
    rm -rf $WORK_DIR/config
    mkdir -p $WORK_DIR/config

    cp $TEST_DIR/scripts/zk_setup.sh $WORK_DIR/config
    cp $TEST_DIR/scripts/zk_health_check/zk_health_check $WORK_DIR/config
    cp $WORKLOAD_DIR/$TEST_CASE/nightcore_config.json $WORK_DIR/config
    cp $WORKLOAD_DIR/$TEST_CASE/run_launcher $WORK_DIR/config

    rm -rf $WORK_DIR/mnt
    mkdir -p $WORK_DIR/mnt

    # dynamodb
    mkdir -p $WORK_DIR/mnt/dynamodb

    # engine nodes
    for node_i in $(seq 1 $INDEX_REPLICATION); do
        mkdir $WORK_DIR/mnt/inmem$node_i
        mkdir $WORK_DIR/mnt/inmem$node_i/boki
        mkdir $WORK_DIR/mnt/inmem$node_i/gperf

        cp $WORK_DIR/config/nightcore_config.json $WORK_DIR/mnt/inmem$node_i/boki/func_config.json
        cp $WORK_DIR/config/run_launcher $WORK_DIR/mnt/inmem$node_i/boki/run_launcher

        mkdir $WORK_DIR/mnt/inmem$node_i/boki/output
        mkdir $WORK_DIR/mnt/inmem$node_i/boki/ipc
    done

    # storage nodes
    for node_i in $(seq 1 $USERLOG_REPLICATION); do
        # delete old RocksDB datas
        mkdir $WORK_DIR/mnt/storage$node_i
        mkdir $WORK_DIR/mnt/storage$node_i/gperf
    done

    # sequencer nodes
    for node_i in $(seq 1 $METALOG_REPLICATION); do
        mkdir $WORK_DIR/mnt/sequencer$node_i
        mkdir $WORK_DIR/mnt/sequencer$node_i/gperf
    done
}

function build {
    echo "========== build boki =========="
    docker run --rm -v $BOKI_DIR:/boki adjwang/boki-buildenv:dev bash -c "cd /boki && make -j$(nproc)"

    echo "========== build workloads =========="
    $WORKLOAD_DIR/build_all.sh

    echo "========== build docker images =========="
    # build boki docker image
    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki:dev \
        -f $DOCKERFILE_DIR/Dockerfile.boki \
        $BOKI_DIR

    # build workloads docker image
    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki-tests:dev \
        -f $DOCKERFILE_DIR/Dockerfile.testcases \
        $WORKLOAD_DIR
}

function cleanup {
    cd $WORK_DIR && docker compose down || true
    sudo rm -rf $WORK_DIR
    mkdir -p $WORK_DIR
}

# test utils

function sleep_count_down {
    for i in $(seq 1 $1); do
        printf "\rsleep...%d   \b\b\b" $(($1 + 1 - $i))
        sleep 1
    done
    echo ""
}
function failed {
    echo "--- FAIL at line: $1"
    exit 1
}
function assert_should_success {
    if [ $? -ne 0 ]; then
        failed $1
    fi
}
function assert_should_fail {
    if [ $? -eq 0 ]; then
        failed $1
    fi
}
function debug {
    # sleep_count_down 12
    # assert_should_fail $LINENO

    echo "test Foo"
    timeout 1 curl -X POST -d "abc" http://localhost:9000/function/Foo -f ||
        assert_should_success $LINENO

    echo "test unknown"
    timeout 1 curl -X POST -d "abc" http://localhost:9000/function/unknown -fs ||
        assert_should_fail $LINENO
    echo "end"
}

function test_sharedlog {
    echo "========== test sharedlog =========="

    echo "setup env..."
    python3 $TEST_DIR/scripts/docker-compose-generator.py \
        --metalog-reps=3 \
        --userlog-reps=3 \
        --index-reps=1 \
        --test-case=sharedlog \
        --workdir=$WORK_DIR \
        --output=$WORK_DIR

    setup_env 3 3 1 sharedlog

    echo "setup cluster..."
    cd $WORK_DIR && docker compose up -d

    echo "wait to startup..."
    sleep_count_down 15

    echo "list functions"
    timeout 1 curl -f -X POST -d "abc" http://localhost:9000/list_functions ||
        assert_should_success $LINENO

    echo "test Foo"
    OUTPUT=$(timeout 1 curl -f -X POST -d "abc" http://localhost:9000/function/Foo 2>/dev/null)
    if [[ $OUTPUT != "foo invokes bar, output=bar invoked with arg=abc" ]]; then
        failed $LINENO
    fi

    echo "test Bar"
    OUTPUT=$(timeout 1 curl -f -X POST -d "abc" http://localhost:9000/function/Bar 2>/dev/null)
    if [[ $OUTPUT != "bar invoked with arg=abc" ]]; then
        failed $LINENO
    fi

    echo "test unknown"
    timeout 1 curl -fs -X POST -d "abc" http://localhost:9000/function/unknown ||
        assert_should_fail $LINENO

    echo "test shared log operations"
    timeout 1 curl -f -X POST -d "abc" http://localhost:9000/function/BasicLogOp ||
        assert_should_success $LINENO

    echo "run bench"
    timeout 10 curl -f -X POST -d "abc" http://localhost:9000/function/Bench ||
        assert_should_success $LINENO

    echo "check docker status"
    if [ $(docker ps -a -f name=boki-test-* -f status=exited -q | wc -l) -ne 0 ]; then
        failed $LINENO
    fi

    echo "shutdown cluster..."
    cd $WORK_DIR && docker compose down
}

if [ $# -eq 0 ]; then
    echo "[ERROR] needs an arg ['build', 'clean', 'run']"
    exit 1
fi
case "$1" in
debug)
    debug $LINENO
    ;;
build)
    build
    ;;
clean)
    cleanup
    ;;
run)
    test_sharedlog
    ;;
*)
    echo "[ERROR] unknown arg '$1', needs ['build', 'clean', 'run']"
    ;;
esac
