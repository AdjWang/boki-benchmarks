#!/bin/bash
set -euxo pipefail

BASE_DIR=$(realpath $(dirname $0))
BOKI_ROOT=$(realpath $(dirname $0)/../../boki)
BUILD_TYPE=release

ZOOKEEPER_HOST="localhost:2181"
IFACE="enp0s3"

rm -rf $BASE_DIR/outputs
mkdir -p $BASE_DIR/outputs

# run zookeeper
docker run --rm -p 2181:2181 zookeeper:3.6.2 \
    >$BASE_DIR/outputs/zookeeper.log \
    2>$BASE_DIR/outputs/zookeeper_err.log &

sleep 3

# setup zookeeper
docker run --rm -i --net=host zookeeper:3.6.2 \
    bash -c "ZOO_LOG4J_PROP="WARN,CONSOLE" ./bin/zkCli.sh -server $ZOOKEEPER_HOST" <<EOF
deleteall /faas
create /faas
create /faas/node
create /faas/view
create /faas/freeze
create /faas/cmd
quit
EOF

if [[ $? != 0 ]]; then
    echo "Failed to setup zookeeper"
    exit 1
fi

# run stack
$BOKI_ROOT/bin/$BUILD_TYPE/controller \
    --zookeeper_host=$ZOOKEEPER_HOST \
    --metalog_replicas=2 \
    --userlog_replicas=1 \
    --index_replicas=1 \
    --v=1 \
    2>$BASE_DIR/outputs/controller.log &

$BOKI_ROOT/bin/$BUILD_TYPE/gateway \
    --zookeeper_host=$ZOOKEEPER_HOST \
    --listen_iface=$IFACE \
    --http_port=9000 \
    --func_config_file=$BASE_DIR/func_config.json \
    --num_io_workers=2 \
    --io_uring_entries=64 \
    --io_uring_fd_slots=128 \
    --lb_per_fn_round_robin \
    --max_running_requests=0 \
    --v=1 \
    2>$BASE_DIR/outputs/gateway.log &

$BOKI_ROOT/bin/$BUILD_TYPE/engine \
    --zookeeper_host=zookeeper:2181 \
    --listen_iface=eth0 \
    --root_path_for_ipc=/tmp/boki/ipc \
    --func_config_file=$BASE_DIR/func_config.json \
    --num_io_workers=4 \
    --instant_rps_p_norm=0.8 \
    --io_uring_entries=64 \
    --io_uring_fd_slots=128 \
    --enable_shared_log \
    --slog_engine_enable_cache \
    --slog_engine_cache_cap_mb=512 \
    --slog_engine_propagate_auxdata \
    --node_id=1 \
    --v=1 \
    2>$BASE_DIR/outputs/engine.log &

sleep 1

$BOKI_ROOT/bin/$BUILD_TYPE/launcher \
    --func_id=1 --fprocess_mode=go \
    --fprocess_output_dir=$BASE_DIR/outputs \
    --fprocess=$BASE_DIR/bin/main \
    --v=1 2>$BASE_DIR/outputs/launcher_foo.log &

$BOKI_ROOT/bin/$BUILD_TYPE/launcher \
    --func_id=2 --fprocess_mode=go \
    --fprocess_output_dir=$BASE_DIR/outputs \
    --fprocess=$BASE_DIR/bin/main \
    --v=1 2>$BASE_DIR/outputs/launcher_bar.log &

sleep 1

# start
docker run --rm -i --net=host zookeeper:3.6.2 \
    bash -c "ZOO_LOG4J_PROP="WARN,CONSOLE" ./bin/zkCli.sh -server localhost:2181" <<EOF
create /faas/cmd/start
quit
EOF

wait
