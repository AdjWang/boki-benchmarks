#!/bin/bash
set -euo pipefail

DEBUG_BUILD=0
TEST_DIR="$(realpath $(dirname "$0"))"
BOKI_DIR=$(realpath $TEST_DIR/../boki)
SCRIPT_DIR=$(realpath $TEST_DIR/../scripts/local_debug)
DOCKERFILE_DIR=$(realpath $SCRIPT_DIR/dockerfiles)

BENCH_EXP_DIR=$(realpath $TEST_DIR/../experiments/microbenchmark)
BENCH_SRC_DIR=$(realpath $TEST_DIR/../workloads/microbenchmark)
QUEUE_EXP_DIR=$(realpath $TEST_DIR/../experiments/queue)
QUEUE_SRC_DIR=$(realpath $TEST_DIR/../workloads/queue)
RETWIS_EXP_DIR=$(realpath $TEST_DIR/../experiments/retwis)
RETWIS_SRC_DIR=$(realpath $TEST_DIR/../workloads/retwis)
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

    cp $SCRIPT_DIR/zk_setup.sh $WORK_DIR/config
    cp $SCRIPT_DIR/zk_health_check/zk_health_check $WORK_DIR/config
    if [[ $TEST_CASE == microbench ]]; then
        cp $BENCH_EXP_DIR/nightcore_config.json $WORK_DIR/config/nightcore_config.json
        cp $BENCH_EXP_DIR/run_launcher $WORK_DIR/config
    elif [[ $TEST_CASE == queue ]]; then
        cp $QUEUE_EXP_DIR/boki/nightcore_config.json $WORK_DIR/config/nightcore_config.json
        cp $QUEUE_EXP_DIR/boki/run_launcher $WORK_DIR/config
    elif [[ $TEST_CASE == retwis ]]; then
        cp $RETWIS_EXP_DIR/boki/nightcore_config.json $WORK_DIR/config/nightcore_config.json
        cp $RETWIS_EXP_DIR/boki/run_launcher $WORK_DIR/config
    elif [[ $TEST_CASE == sharedlog ]]; then
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

function build_testcases {
    echo "========== build sharedlog =========="
    $TEST_DIR/workloads/sharedlog/build.sh
    docker run --rm -v $TEST_DIR/..:/boki-benchmark adjwang/boki-benchbuildenv:dev \
        /boki-benchmark/tests/workloads/sharedlog/build.sh

    # build test docker image
    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki-tests:dev \
        -f $DOCKERFILE_DIR/Dockerfile.testcases \
        $TEST_DIR/workloads
}
function build_microbench {
    echo "========== build bench =========="
    docker run --rm -v $TEST_DIR/..:/boki-benchmark adjwang/boki-benchbuildenv:dev \
        /boki-benchmark/workloads/microbenchmark/build.sh

    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki-microbench:dev \
        -f $DOCKERFILE_DIR/Dockerfile.microbench \
        $BENCH_SRC_DIR
}
function build_queue {
    echo "========== build queue =========="
    docker run --rm -v $TEST_DIR/..:/boki-benchmark adjwang/boki-benchbuildenv:dev \
        /boki-benchmark/workloads/queue/build.sh

    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki-queuebench:dev \
        -f $DOCKERFILE_DIR/Dockerfile.queuebench \
        $QUEUE_SRC_DIR
}
function build_retwis {
    echo "========== build retwis =========="
    docker run --rm -v $TEST_DIR/..:/boki-benchmark adjwang/boki-benchbuildenv:dev \
        /boki-benchmark/workloads/retwis/build.sh

    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki-retwisbench:dev \
        -f $DOCKERFILE_DIR/Dockerfile.retwisbench \
        $RETWIS_SRC_DIR
}
function build_workflow {
    echo "========== build workloads =========="
    docker run --rm -v $TEST_DIR/..:/boki-benchmark adjwang/boki-benchbuildenv:dev \
        /boki-benchmark/workloads/workflow/build_all.sh

    # build app docker image
    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki-beldibench:dev \
        -f $DOCKERFILE_DIR/Dockerfile.beldibench \
        $WORKFLOW_SRC_DIR
}
function build_boki {
    echo "========== build boki =========="
    docker run --rm -v $BOKI_DIR:/boki adjwang/boki-buildenv:dev bash -c "cd /boki && make -j$(nproc)"

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

    set -euxo pipefail
    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki-queuebench:dev \
        -f $DOCKERFILE_DIR/Dockerfile.queuebench \
        $QUEUE_SRC_DIR
}

function test_sharedlog {
    echo "========== test sharedlog =========="

    echo "setup env..."
    python3 $SCRIPT_DIR/docker-compose-generator.py \
        --metalog-reps=3 \
        --userlog-reps=3 \
        --index-reps=1 \
        --test-case=sharedlog \
        --workdir=$WORK_DIR \
        --output=$WORK_DIR

    setup_env 3 3 1 sharedlog

    echo "setup cluster..."
    cd $WORK_DIR && docker compose up -d --remove-orphans

    echo "waiting to startup..."
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

function test_microbench {
    echo "setup env..."
    python3 $SCRIPT_DIR/docker-compose-generator.py \
        --metalog-reps=3 \
        --userlog-reps=3 \
        --index-reps=1 \
        --test-case=microbench \
        --workdir=$WORK_DIR \
        --output=$WORK_DIR

    setup_env 3 3 1 microbench

    echo "setup cluster..."
    cd $WORK_DIR && docker compose up -d --remove-orphans

    echo "waiting to startup..."
    sleep_count_down 15

    echo "list functions"
    timeout 1 curl -f -X POST -d "abc" http://localhost:9000/list_functions ||
        assert_should_success $LINENO

    # example:
    # curl -s -X POST -d '{"PayloadSize":1024,"BatchSize":1}' http://localhost:9000/function/benchAsyncLogRead | gunzip

    set -x
    # $BENCH_SRC_DIR/bin/benchmark \
    #     --faas_gateway=localhost:9000 --bench_case="write" \
    #     --batch_size=10 --concurrency=10 \
    #     --payload_size=1024 --duration=3

    # $BENCH_SRC_DIR/bin/benchmark \
    #     --faas_gateway=localhost:9000 --bench_case="read" \
    #     --batch_size=10 --concurrency=10 \
    #     --payload_size=1024 --duration=3

    $BENCH_SRC_DIR/bin/benchmark \
        --faas_gateway=localhost:9000 --bench_case="read_cached" \
        --batch_size=100 --concurrency=2 \
        --payload_size=1024 --duration=3
}

function test_queue {
    APP_NAME="queue"
    APP_SRC_DIR=$QUEUE_SRC_DIR/"queue"

    QUEUE_PREFIX=$(echo $RANDOM | md5sum | head -c8)
    QUEUE_PREFIX="${QUEUE_PREFIX}-"

    echo "setup env..."
    python3 $SCRIPT_DIR/docker-compose-generator.py \
        --metalog-reps=3 \
        --userlog-reps=3 \
        --index-reps=1 \
        --test-case=queue \
        --workdir=$WORK_DIR \
        --output=$WORK_DIR

    setup_env 3 3 1 queue

    echo "setup cluster..."
    cd $WORK_DIR && docker compose up -d --remove-orphans

    echo "waiting to startup..."
    sleep_count_down 15

    echo "list functions"
    timeout 1 curl -f -X POST -d "abc" http://localhost:9000/list_functions ||
        assert_should_success $LINENO

    NUM_SHARDS=2
    INTERVAL1=800 # ms
    INTERVAL2=500 # ms
    NUM_PRODUCER=2
    NUM_CONSUMER=2

    set -x
    $QUEUE_SRC_DIR/bin/benchmark \
        --faas_gateway=localhost:9000 --fn_prefix=slib \
        --queue_prefix=$QUEUE_PREFIX --num_queues=1 --queue_shards=$NUM_SHARDS \
        --num_producer=$NUM_PRODUCER --num_consumer=$NUM_CONSUMER \
        --producer_interval=$INTERVAL1 --consumer_interval=$INTERVAL2 \
        --consumer_fix_shard=true \
        --payload_size=1024 --duration=3
}

function test_retwis {
    echo "setup env..."
    python3 $SCRIPT_DIR/docker-compose-generator.py \
        --metalog-reps=3 \
        --userlog-reps=3 \
        --index-reps=1 \
        --test-case=retwis \
        --workdir=$WORK_DIR \
        --output=$WORK_DIR

    setup_env 3 3 1 retwis

    echo "setup cluster..."
    cd $WORK_DIR && docker compose up -d --remove-orphans

    echo "waiting to startup..."
    sleep_count_down 15

    echo "list functions"
    timeout 1 curl -f -X POST -d "abc" http://localhost:9000/list_functions ||
        assert_should_success $LINENO

    CONCURRENCY=8 # 64, 96, 128, 192
    # NUM_USERS=10000
    NUM_USERS=100

    set -x
    # init
    curl -X POST http://localhost:9000/function/RetwisInit
    # create users
    $RETWIS_SRC_DIR/bin/create_users --faas_gateway=localhost:9000 --num_users=$NUM_USERS --concurrency=32
    # run benchmark
    $RETWIS_SRC_DIR/bin/benchmark \
        --faas_gateway=localhost:9000 --num_users=$NUM_USERS \
        --percentages=15,30,50,5 \
        --duration=5 --concurrency=$CONCURRENCY
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
    beldi-singleop-baseline)
        APP_NAME="singleop"
        APP_SRC_DIR=$WORKFLOW_SRC_DIR/"beldi"
        BELDI_BASELINE="1"
        ;;
    boki-singleop-baseline)
        APP_NAME="singleop"
        APP_SRC_DIR=$WORKFLOW_SRC_DIR/"boki"
        BELDI_BASELINE="0"
        ;;
    boki-singleop-asynclog)
        APP_NAME="singleop"
        APP_SRC_DIR=$WORKFLOW_SRC_DIR/"asynclog"
        BELDI_BASELINE="0"
        ;;
    *)
        echo "[ERROR] TEST_CASE should be either beldi-hotel|movie-baseline or boki-hotel|movie-baseline|asynclog, given $TEST_CASE"
        exit 1
        ;;
    esac

    if [[ $APP_NAME == "hotel" || $APP_NAME == "singleop" ]]; then
        DB_DATA=""
    else
        DB_DATA=$APP_SRC_DIR/internal/media/data/compressed.json
    fi

    echo "========== test workflow $TEST_CASE =========="

    # strange bug: head not generating EOF and just stucks. Only on my vm, tested ok in WSL.
    # TABLE_PREFIX=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 8 | head -n 1)
    TABLE_PREFIX=$(echo $RANDOM | md5sum | head -c8)
    TABLE_PREFIX="${TABLE_PREFIX}-"

    echo "setup env..."
    python3 $SCRIPT_DIR/docker-compose-generator.py \
        --metalog-reps=3 \
        --userlog-reps=3 \
        --index-reps=1 \
        --test-case=$TEST_CASE \
        --table-prefix=$TABLE_PREFIX \
        --workdir=$WORK_DIR \
        --output=$WORK_DIR

    setup_env 3 3 1 $TEST_CASE

    echo "setup cluster..."
    cd $WORK_DIR && docker compose up -d --remove-orphans

    echo "waiting to startup..."
    sleep_count_down 15

    echo "list functions"
    timeout 1 curl -f -X POST -d "abc" http://localhost:9000/list_functions ||
        assert_should_success $LINENO

    if [[ $APP_NAME == "singleop" ]]; then
        echo "test singleop"
        if [[ $BELDI_BASELINE == "0" ]]; then
            timeout 10 curl -f -X POST -d "{}" http://localhost:9000/function/singleop ||
                assert_should_success $LINENO
        else
            timeout 10 curl -f -X POST -d "{}" http://localhost:9000/function/bsingleop ||
                assert_should_success $LINENO
        fi
        echo ""
    elif [[ $APP_NAME == "hotel" ]]; then
        echo "test read (search) request"
        timeout 10 curl -X POST -H "Content-Type: application/json" -d '{"InstanceId":"","CallerName":"","Async":false,"Input":{"Function":"search","Input":{"InDate":"2015-04-21","Lat":37.785999999999996,"Lon":-122.40999999999999,"OutDate":"2015-04-24"}}}' \
            http://localhost:9000/function/gateway ||
            assert_should_success $LINENO
        echo ""

        echo "test read (recommend) request"
        timeout 10 curl -X POST -H "Content-Type: application/json" -d '{"InstanceId":"","CallerName":"","Async":false,"Input":{"Function":"recommend","Input":{"Require":"price","Lat":37.988,"Lon":-122.067}}}' \
            http://localhost:9000/function/gateway ||
            assert_should_success $LINENO
        echo ""

        echo "test read (user) request"
        timeout 10 curl -X POST -H "Content-Type: application/json" -d '{"InstanceId":"","CallerName":"","Async":false,"Input":{"Function":"user","Input":{"Username":"Cornell_72","Password":"72727272727272727272"}}}' \
            http://localhost:9000/function/gateway ||
            assert_should_success $LINENO
        echo ""

        echo "test write (reserve) request"
        timeout 10 curl -X POST -H "Content-Type: application/json" -d '{"InstanceId":"","CallerName":"","Async":false,"Input":{"Function":"reserve","Input":{"userId":"user1","hotelId":"75","flightId":"8"}}}' \
            http://localhost:9000/function/gateway ||
            assert_should_success $LINENO
        echo ""
    else # $APP_NAME == "media"
        echo "test basic"
        curl -X POST -H "Content-Type: application/json" -d '{"InstanceId":"","CallerName":"","Async":false,"Input":{"Function":"Compose","Input":{"Username":"username_80","Password":"password_80","Title":"Welcome to Marwen","Rating":7,"Text":"cZQPir9Ka9kcRJPBEsGfAoMAwMrMDMsh6ztv6wHXOioeTJY2ol3CKG1qrCm80blj38ACrvF7XuarfpQSjMkdpCrBJo7NbBtJUBtYKOuGtdBJ0HM9vv77N2JGI3mrcwyPGB9xdlnXOMUwlldt8NVpkjEBGjM1b4VOBwO3lYSxn34qhrnY7x6oOrlGN5PO70Bgxnckdf0wdRrYWdIw5qKY7sN5Gzuaq1fkeLbHGmHPeHtJ8iOfAVkizGHyRXukRqln"}}}' \
            http://localhost:9000/function/Frontend ||
            assert_should_success $LINENO
        echo ""
    fi

    echo "test more requests"
    set -x
    # DEBUG: benchmarks printing responses
    WRK="docker run --rm --net=host -v $SCRIPT_DIR:$SCRIPT_DIR 1vlad/wrk2-docker"
    BASELINE=$BELDI_BASELINE $WRK -t 2 -c 2 -d 3 -s $SCRIPT_DIR/benchmark/$APP_NAME/workload.lua http://localhost:9000 -R 5

    # BASELINE=$BELDI_BASELINE $WRK -t 2 -c 2 -d 150 -s $APP_SRC_DIR/benchmark/$APP_NAME/workload.lua http://localhost:9000 -R 5
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
    build_boki
    # build_testcases
    # build_microbench
    # build_queue
    # build_retwis
    build_workflow
    ;;
push)
    echo "========== push docker images =========="
    docker push adjwang/boki:dev
    # docker push adjwang/boki-microbench:dev
    # docker push adjwang/boki-queuebench:dev
    # docker push adjwang/boki-retwisbench:dev
    docker push adjwang/boki-beldibench:dev
    ;;
clean)
    cleanup
    ;;
run)
    # test_sharedlog

    # test_microbench
    # test_queue
    # test_retwis

    # test_workflow beldi-hotel-baseline
    # test_workflow beldi-movie-baseline
    test_workflow boki-hotel-baseline
    # test_workflow boki-movie-baseline
    # test_workflow boki-hotel-asynclog
    # test_workflow boki-movie-asynclog
    # test_workflow beldi-singleop-baseline
    # test_workflow boki-singleop-baseline
    # test_workflow boki-singleop-asynclog
    ;;
*)
    echo "[ERROR] unknown arg '$1', needs ['build', 'push', 'clean', 'run']"
    ;;
esac
