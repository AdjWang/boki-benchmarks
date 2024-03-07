#!/bin/bash
set -euo pipefail

DEBUG_BUILD=0
TEST_DIR="$(realpath $(dirname "$0"))"
BOKI_DIR=$(realpath $TEST_DIR/../boki)
SCRIPT_DIR=$(realpath $TEST_DIR/../scripts)
DEBUG_SCRIPT_DIR=$(realpath $TEST_DIR/../scripts/local_debug)
DOCKERFILE_DIR=$(realpath $DEBUG_SCRIPT_DIR/dockerfiles)
WORKFLOW_EXP_DIR=$(realpath $TEST_DIR/../experiments/workflow)
WORKFLOW_SRC_DIR=$(realpath $TEST_DIR/../workloads/workflow)

WORK_DIR=/tmp/boki-test

DOCKER_BUILDER="docker buildx"
NO_CACHE=""

function setup_env {
    METALOG_REPLICATION=$1
    USERLOG_REPLICATION=$2
    INDEX_REPLICATION=$3
    TEST_CASE=$4

    # remove old files and folders
    rm -rf $WORK_DIR/config
    mkdir -p $WORK_DIR/config

    cp $DEBUG_SCRIPT_DIR/zk_setup.sh $WORK_DIR/config
    # inspect unhealthy log:
    # docker inspect --format "{{json .State.Health }}" $(docker ps -a | grep unhealthy | awk '{print $1}') | jq
    if [ ! -f $DEBUG_SCRIPT_DIR/zk_health_check/zk_health_check ]; then
        docker run --rm -v $TEST_DIR/..:/boki-benchmark adjwang/boki-benchbuildenv:dev \
            bash -c "cd /boki-benchmark/scripts/local_debug/zk_health_check && make"
    fi
    cp $DEBUG_SCRIPT_DIR/zk_health_check/zk_health_check $WORK_DIR/config
    if [[ $TEST_CASE == sharedlog ]]; then
        cp $TEST_DIR/workloads/sharedlog/nightcore_config.json $WORK_DIR/config/nightcore_config.json
        cp $TEST_DIR/workloads/sharedlog/run_launcher $WORK_DIR/config
    else
        cp $WORKFLOW_EXP_DIR/$TEST_CASE/nightcore_config.json $WORK_DIR/config/nightcore_config.json
        cp $WORKFLOW_EXP_DIR/$TEST_CASE/run_launcher $WORK_DIR/config
    fi

    rm -rf $WORK_DIR/mnt
    mkdir -p $WORK_DIR/mnt

    # dynamodb
    mkdir -p $WORK_DIR/mnt/dynamodb

    # gateway
    mkdir $WORK_DIR/mnt/inmem_gateway
    mkdir $WORK_DIR/mnt/inmem_gateway/store

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

    # echo "========== build sharedlog =========="
    # $TEST_DIR/workloads/sharedlog/build.sh

    echo "========== build workloads =========="
    # $WORKFLOW_SRC_DIR/build_all.sh
    docker run --rm -v $TEST_DIR/..:/boki-benchmark adjwang/boki-benchbuildenv:dev \
        /boki-benchmark/workloads/workflow/build_all.sh

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
        $TEST_DIR/workloads
    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki-beldibench:dev \
        -f $DOCKERFILE_DIR/Dockerfile.beldibench \
        $WORKFLOW_SRC_DIR
}

function push {
    # DEPRECATED
    # echo "========== build workloads =========="
    # # $WORKFLOW_SRC_DIR/build_all.sh REMOTE
    # docker run --rm -v $TEST_DIR/..:/boki-benchmark adjwang/boki-benchbuildenv:dev \
    #     bash -c "/boki-benchmark/workloads/workflow/build_all.sh REMOTE"

    echo "========== build docker images =========="
    # boki
    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki:dev \
        -f $DOCKERFILE_DIR/Dockerfile.boki \
        $BOKI_DIR
    # bokiflow
    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki-beldibench:dev \
        -f $DOCKERFILE_DIR/Dockerfile.beldibench \
        $WORKFLOW_SRC_DIR

    echo "========== push docker images =========="
    docker push adjwang/boki:dev
    docker push adjwang/boki-beldibench:dev
}

function cleanup {
    cd $WORK_DIR && docker compose down -t 1 || true
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

    TEST="abc"

    if [[ $TEST == "abc" ]]; then
        echo "ok"
    fi
}

function test_sharedlog {
    echo "========== test sharedlog =========="

    echo "setup env..."
    python3 $DEBUG_SCRIPT_DIR/docker-compose-generator.py \
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
function test_workflow {
    TEST_CASE=$1
    case $TEST_CASE in
    beldi-hotel-baseline)
        APP_NAME="hotel"
        APP_SRC_DIR=$WORKFLOW_SRC_DIR/"beldi"
        BELDI_BASELINE="1"
        ;;
    beldi-movie-baseline)
        APP_NAME="media"
        APP_SRC_DIR=$WORKFLOW_SRC_DIR/"beldi"
        BELDI_BASELINE="1"
        ;;
    boki-hotel-baseline)
        APP_NAME="hotel"
        APP_SRC_DIR=$WORKFLOW_SRC_DIR/"boki"
        BELDI_BASELINE="0"
        ;;
    boki-movie-baseline)
        APP_NAME="media"
        APP_SRC_DIR=$WORKFLOW_SRC_DIR/"boki"
        BELDI_BASELINE="0"
        ;;
    boki-finra-baseline)
        APP_NAME="finra"
        APP_SRC_DIR=$WORKFLOW_SRC_DIR/"boki"
        BELDI_BASELINE="0"
        ;;
    boki-hotel-asynclog)
        APP_NAME="hotel"
        APP_SRC_DIR=$WORKFLOW_SRC_DIR/"asynclog"
        BELDI_BASELINE="0"
        ;;
    boki-movie-asynclog)
        APP_NAME="media"
        APP_SRC_DIR=$WORKFLOW_SRC_DIR/"asynclog"
        BELDI_BASELINE="0"
        ;;
    boki-finra-asynclog)
        APP_NAME="finra"
        APP_SRC_DIR=$WORKFLOW_SRC_DIR/"asynclog"
        BELDI_BASELINE="0"
        ;;
    *)
        echo "[ERROR] TEST_CASE should be either beldi-hotel|movie-baseline or boki-hotel|movie-baseline|asynclog, given $TEST_CASE"
        exit 1
        ;;
    esac

    if [[ $APP_NAME == "media" ]]; then
        DB_DATA=$APP_SRC_DIR/internal/media/data/compressed.json
    else
        DB_DATA=""
    fi

    echo "========== test workflow $TEST_CASE =========="

    # strange bug: head not generating EOF and just stucks. Only on my vm, tested ok in WSL.
    # TABLE_PREFIX=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 8 | head -n 1)
    # TABLE_PREFIX=$(echo $RANDOM | md5sum | head -c8)
    # TABLE_PREFIX="${TABLE_PREFIX}-"

    echo "setup env..."
    python3 $SCRIPT_DIR/bokicli/bin/local_config_generator.py \
        --metalog-reps=1 \
        --userlog-reps=1 \
        --index-reps=1 \
        --test-case=$TEST_CASE \
        --workdir=$WORK_DIR \
        --output=$WORK_DIR

    setup_env 1 1 1 $TEST_CASE

    echo "setup cluster..."
    cd $WORK_DIR && docker compose up -d --remove-orphans

    echo "wait to startup..."
    sleep_count_down 15

    echo "list functions"
    timeout 1 curl -f -X POST -d "abc" http://localhost:9000/list_functions ||
        assert_should_success $LINENO

    if [[ $APP_NAME == "hotel" ]]; then
        # TODO
        # echo "test singleop"
        # timeout 10 curl -f -X POST -d "{}" http://localhost:9000/function/singleop ||
        #     assert_should_success $LINENO
        # echo ""

        echo "test read request"
        timeout 10 curl -X POST -H "Content-Type: application/json" -d '{"Async":false,"CallerName":"","Input":{"Function":"search","Input":{"InDate":"2015-04-21","Lat":37.785999999999996,"Lon":-122.40999999999999,"OutDate":"2015-04-24"}},"InstanceId":"b1f69474bc9147ae89850ccb57be7085"}' \
            http://localhost:9000/function/gateway ||
            assert_should_success $LINENO
        echo ""

        # echo "test write request"
        # timeout 10 curl -X POST -H "Content-Type: application/json" -d '{"InstanceId":"","CallerName":"","Async":false,"Input":{"Function":"reserve","Input":{"userId":"user1","hotelId":"75","flightId":"8"}}}' \
        #     http://localhost:9000/function/gateway ||
        #     assert_should_success $LINENO
        # echo ""
    elif [[ $APP_NAME == "media" ]]; then
        echo "test basic"
        if [[ $BELDI_BASELINE == "0" ]]; then
            curl -X POST -H "Content-Type: application/json" -d '{"InstanceId":"","CallerName":"","Async":false,"Input":{"Function":"Compose","Input":{"Username":"username_80","Password":"password_80","Title":"Welcome to Marwen","Rating":7,"Text":"cZQPir9Ka9kcRJPBEsGfAoMAwMrMDMsh6ztv6wHXOioeTJY2ol3CKG1qrCm80blj38ACrvF7XuarfpQSjMkdpCrBJo7NbBtJUBtYKOuGtdBJ0HM9vv77N2JGI3mrcwyPGB9xdlnXOMUwlldt8NVpkjEBGjM1b4VOBwO3lYSxn34qhrnY7x6oOrlGN5PO70Bgxnckdf0wdRrYWdIw5qKY7sN5Gzuaq1fkeLbHGmHPeHtJ8iOfAVkizGHyRXukRqln"}}}' \
                http://localhost:9000/function/Frontend ||
                assert_should_success $LINENO
        else # $BELDI_BASELINE == "1"
            curl -X POST -H "Content-Type: application/json" -d '{"InstanceId":"","CallerName":"","Async":false,"Input":{"Function":"Compose","Input":{"Username":"username_80","Password":"password_80","Title":"Welcome to Marwen","Rating":7,"Text":"cZQPir9Ka9kcRJPBEsGfAoMAwMrMDMsh6ztv6wHXOioeTJY2ol3CKG1qrCm80blj38ACrvF7XuarfpQSjMkdpCrBJo7NbBtJUBtYKOuGtdBJ0HM9vv77N2JGI3mrcwyPGB9xdlnXOMUwlldt8NVpkjEBGjM1b4VOBwO3lYSxn34qhrnY7x6oOrlGN5PO70Bgxnckdf0wdRrYWdIw5qKY7sN5Gzuaq1fkeLbHGmHPeHtJ8iOfAVkizGHyRXukRqln"}}}' \
                http://localhost:9000/function/bFrontend ||
                assert_should_success $LINENO
        fi
        echo ""
    elif [[ $APP_NAME == "finra" ]]; then
        curl -X POST -H "Content-Type: application/json" -d '{"InstanceId":"","CallerName":"","Async":false,"Input":{"Function":"fetchData","Input":{"body":{"n_parallel":1,"portfolioType":"S&P","portfolio":"1234"}}}}' \
            http://localhost:9000/function/fetchData ||
            assert_should_success $LINENO
        echo ""
    else
        echo "unknown app name ${APP_NAME}"
        exit 1
    fi

    # echo "test more requests"
    # WRKBENCHDIR=$DEBUG_SCRIPT_DIR
    # # WRKBENCHDIR=$APP_SRC_DIR
    # echo "using wrkload: $WRKBENCHDIR/benchmark/$APP_NAME/workload.lua"
    # WRK="docker run --rm --net=host -e BASELINE=$BELDI_BASELINE -v $WRKBENCHDIR:/workdir 1vlad/wrk2-docker"
    
    # set -x
    # # DEBUG: benchmarks printing responses
    # $WRK -t 2 -c 2 -d 10 -s /workdir/benchmark/$APP_NAME/workload.lua http://localhost:9000 -L -U -R 10

    # # curl -X GET -H "Content-Type: application/json" http://localhost:9000/mark_event?name=warmup_start
    # # $WRK -t 2 -c 2 -d 30 -s /workdir/benchmark/$APP_NAME/workload.lua http://localhost:9000 -L -U -R 100
    # # curl -X GET -H "Content-Type: application/json" http://localhost:9000/mark_event?name=warmup_end
    # # sleep_count_down 10
    # # curl -X GET -H "Content-Type: application/json" http://localhost:9000/mark_event?name=benchmark_start
    # # $WRK -t 2 -c 2 -d 30 -s /workdir/benchmark/$APP_NAME/workload.lua http://localhost:9000 -L -U -R 100
    # # curl -X GET -H "Content-Type: application/json" http://localhost:9000/mark_event?name=benchmark_end
    # # sleep_count_down 10

    # # wc -l /tmp/boki-test/mnt/inmem_gateway/store/async_results
    # # python3 $DEBUG_SCRIPT_DIR/compute_latency.py --async-result-file /tmp/boki-test/mnt/inmem_gateway/store/async_results
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
    # cleanup
    # test_workflow beldi-hotel-baseline
    # test_workflow beldi-movie-baseline
    # test_workflow boki-hotel-baseline
    # test_workflow boki-movie-baseline
    # test_workflow boki-finra-baseline
    # test_workflow boki-finra-asynclog
    test_workflow boki-hotel-asynclog
    # test_workflow boki-movie-asynclog
    ;;
*)
    echo "[ERROR] unknown arg '$1', needs ['build', 'push', 'clean', 'run']"
    ;;
esac
