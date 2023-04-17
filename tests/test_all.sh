#!/bin/bash
set -euo pipefail

DEBUG_BUILD=0
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
    if [ $DEBUG_BUILD -eq 1 ]; then
        $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki:dev \
            -f $DOCKERFILE_DIR/Dockerfile.bokidebug \
            $BOKI_DIR
    else
        $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki:dev \
            -f $DOCKERFILE_DIR/Dockerfile.boki \
            $BOKI_DIR
    fi

    # build workloads docker image
    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki-tests:dev \
        -f $DOCKERFILE_DIR/Dockerfile.testcases \
        $WORKLOAD_DIR
    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki-beldibench:dev \
        -f $DOCKERFILE_DIR/Dockerfile.beldibench \
        $WORKLOAD_DIR
}

function push {
    echo "========== build docker images =========="
    # boki
    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki:dev \
        -f $DOCKERFILE_DIR/Dockerfile.boki \
        $BOKI_DIR
    # bokiflow
    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki-beldibench:dev \
        -f $DOCKERFILE_DIR/Dockerfile.beldibench \
        $WORKLOAD_DIR

    echo "========== push docker images =========="
    docker push adjwang/boki:dev
    docker push adjwang/boki-beldibench:dev
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

    python3 $TEST_DIR/scripts/docker-compose-generator.py \
        --metalog-reps=3 \
        --userlog-reps=3 \
        --index-reps=1 \
        --test-case=sharedlog \
        --workdir=$WORK_DIR \
        --output=$WORK_DIR

    setup_env 3 3 1 sharedlog

    python3 $TEST_DIR/scripts/docker-compose-generator.py \
        --metalog-reps=3 \
        --userlog-reps=3 \
        --index-reps=8 \
        --test-case=bokiflow \
        --table-prefix="abc" \
        --workdir=$WORK_DIR \
        --output=$WORK_DIR

    setup_env 3 3 8 bokiflow
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

    echo "test async shared log operations"
    timeout 10 curl -f -X POST -d "abc" http://localhost:9000/function/AsyncLogOp ||
        assert_should_success $LINENO

    echo "run bench"
    timeout 10 curl -f -X POST -d "abc" http://localhost:9000/function/Bench ||
        assert_should_success $LINENO

    echo "check docker status"
    if [ $(docker ps -a -f name=boki-test-* -f status=exited -q | wc -l) -ne 0 ]; then
        failed $LINENO
    fi
}

# wrk -t 1 -c 1 -d 5 -s ./workloads/bokiflow/benchmark/hotel/workload.lua http://localhost:9000 -R 1
# docker run --rm -v $HOME/dev/boki-benchmarks/tests/workloads/bokiflow/benchmark/hotel:/tmp/bench ghcr.io/eniac/beldi/beldi:latest /root/beldi/tools/wrk -t 1 -c 1 -d 5 -s /tmp/bench/workload.lua http://10.0.2.15:9000 -R 1
function test_bokiflow {
    echo "========== test bokiflow =========="

    # strange bug: head not generating EOF and just stucks. Only on my vm, tested ok in WSL.
    # TABLE_PREFIX=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 8 | head -n 1)
    TABLE_PREFIX=$(echo $RANDOM | md5sum | head -c8)
    TABLE_PREFIX="${TABLE_PREFIX}-"

    echo "setup env..."
    python3 $TEST_DIR/scripts/docker-compose-generator.py \
        --metalog-reps=3 \
        --userlog-reps=3 \
        --index-reps=2 \
        --test-case=bokiflow \
        --table-prefix=$TABLE_PREFIX \
        --workdir=$WORK_DIR \
        --output=$WORK_DIR

    setup_env 3 3 2 bokiflow

    echo "restart dynamodb..."
    docker stop $(docker ps -qf ancestor=amazon/dynamodb-local) 2>/dev/null || true
    docker rm $(docker ps -aqf ancestor=amazon/dynamodb-local) 2>/dev/null || true
    docker run -d -p 8000:8000 amazon/dynamodb-local

    echo "setup cluster..."
    cd $WORK_DIR && docker compose up -d

    echo "wait to startup..."
    sleep_count_down 15

    echo "setup dynamodb..."
    $TEST_DIR/workloads/bokiflow/bin/singleop/init
    $TEST_DIR/workloads/bokiflow/bin/hotel/init clean boki
    $TEST_DIR/workloads/bokiflow/bin/hotel/init create boki
    $TEST_DIR/workloads/bokiflow/bin/hotel/init populate boki

    echo "list functions"
    timeout 1 curl -f -X POST -d "abc" http://localhost:9000/list_functions ||
        assert_should_success $LINENO

    echo "test singleop"
    timeout 10 curl -f -X POST -d "{}" http://localhost:9000/function/singleop ||
        assert_should_success $LINENO
    echo ""

    echo "test read request"
    timeout 10 curl -X POST -H "Content-Type: application/json" -d '{"Async":false,"CallerName":"","Input":{"Function":"search","Input":{"InDate":"2015-04-21","Lat":37.785999999999996,"Lon":-122.40999999999999,"OutDate":"2015-04-24"}},"InstanceId":"b1f69474bc9147ae89850ccb57be7085"}' \
        http://localhost:9000/function/gateway ||
        assert_should_success $LINENO
    echo ""

    echo "test write request"
    timeout 10 curl -X POST -H "Content-Type: application/json" -d '{"InstanceId":"","CallerName":"","Async":false,"Input":{"Function":"reserve","Input":{"userId":"user1","hotelId":"75","flightId":"8"}}}' \
        http://localhost:9000/function/gateway ||
        assert_should_success $LINENO
    echo ""

    echo "test more requests"
    wrk -t 2 -c 2 -d 150 -s $TEST_DIR/workloads/bokiflow/benchmark/hotel/workload.lua http://localhost:9000 -R 5
}

if [ $# -eq 0 ]; then
    echo "[ERROR] needs an arg ['build', 'push', 'clean', 'run']"
    exit 1
fi
case "$1" in
debug)
    debug $LINENO
    ;;
build)
    build
    ;;
push)
    push
    ;;
clean)
    cleanup
    ;;
run)
    # test_sharedlog
    test_bokiflow
    ;;
*)
    echo "[ERROR] unknown arg '$1', needs ['build', 'push', 'clean', 'run']"
    ;;
esac
