# !/usr/bin/env python3
# -*- coding: utf-8 -*-
import os
from pathlib import Path
import argparse
from functools import partial

docker_compose_common = """\
version: "3.8"
services:
  zookeeper:
    image: zookeeper:3.6.2
    hostname: zookeeper
    ports:
      - 2181:2181
    restart: always

  zookeeper-setup:
    image: zookeeper:3.6.2
    command: /tmp/boki/zk_setup.sh
    depends_on:
       - zookeeper
    volumes:
      - /tmp/zk_setup.sh:/tmp/boki/zk_setup.sh
    restart: always

"""

docker_compose_faas_nightcore_f = """\
  boki-engine:
    image: {image_faas}
    hostname: faas-engine-{{{{.Task.Slot}}}}
    entrypoint:
      - /boki/engine
      - --zookeeper_host=zookeeper:2181
      - --listen_iface=eth0
      - --root_path_for_ipc=/tmp/boki/ipc
      - --func_config_file=/tmp/boki/func_config.json
      - --num_io_workers=4
      - --instant_rps_p_norm=0.8
      - --io_uring_entries=2048
      - --io_uring_fd_slots=4096
      # - --v=1
    depends_on:
      - zookeeper-setup
    volumes:
      - /mnt/inmem/boki:/tmp/boki
      - /sys/fs/cgroup:/tmp/root_cgroupfs
    environment:
      - FAAS_NODE_ID={{{{.Task.Slot}}}}
      - FAAS_CGROUP_FS_ROOT=/tmp/root_cgroupfs
    restart: always

  boki-gateway:
    image: {image_faas}
    hostname: faas-gateway
    ports:
      - 8080:8080
    entrypoint:
      - /boki/gateway
      - --zookeeper_host=zookeeper:2181
      - --listen_iface=eth0
      - --http_port=8080
      - --func_config_file=/tmp/boki/func_config.json
      - --async_call_result_path=/tmp/store/async_results
      - --num_io_workers=2
      - --io_uring_entries=2048
      - --io_uring_fd_slots=4096
      - --lb_per_fn_round_robin
      - --max_running_requests=0
      # - --v=1
    depends_on:
      - zookeeper-setup
    volumes:
      - /tmp/nightcore_config.json:/tmp/boki/func_config.json
      - /mnt/inmem/store:/tmp/store
    restart: always

"""

docker_compose_faas_boki_f = """\
  boki-engine:
    image: {image_faas}
    hostname: boki-engine-{{{{.Task.Slot}}}}
    entrypoint:
      - /boki/engine
      - --zookeeper_host=zookeeper:2181
      - --listen_iface=eth0
      - --root_path_for_ipc=/tmp/boki/ipc
      - --func_config_file=/tmp/boki/func_config.json
      - --num_io_workers=4
      - --instant_rps_p_norm=0.8
      - --io_uring_entries=2048
      - --io_uring_fd_slots=4096
      - --enable_shared_log
      - --slog_engine_enable_cache
      - --slog_engine_cache_cap_mb=1024
      - --slog_engine_propagate_auxdata
      # - --v=1
    depends_on:
      - zookeeper-setup
    volumes:
      - /mnt/inmem/boki:/tmp/boki
      - /sys/fs/cgroup:/tmp/root_cgroupfs
    environment:
      - FAAS_NODE_ID={{{{.Task.Slot}}}}
      - FAAS_CGROUP_FS_ROOT=/tmp/root_cgroupfs
    restart: always

  boki-controller:
    image: {image_faas}
    entrypoint:
      - /boki/controller
      - --zookeeper_host=zookeeper:2181
      - --metalog_replicas={n_metalog_replicas}
      - --userlog_replicas={n_userlog_replicas}
      - --index_replicas={n_index_replicas}
      # - --v=1
    depends_on:
      - zookeeper-setup
    restart: always

  boki-gateway:
    image: {image_faas}
    hostname: faas-gateway
    ports:
      - 8080:8080
    entrypoint:
      - /boki/gateway
      - --zookeeper_host=zookeeper:2181
      - --listen_iface=eth0
      - --http_port=8080
      - --func_config_file=/tmp/boki/func_config.json
      - --async_call_result_path=/tmp/store/async_results
      - --num_io_workers=2
      - --io_uring_entries=2048
      - --io_uring_fd_slots=4096
      - --lb_per_fn_round_robin
      - --max_running_requests=0
      # - --v=1
    depends_on:
      - zookeeper-setup
    volumes:
      - /tmp/nightcore_config.json:/tmp/boki/func_config.json
      - /mnt/inmem/store:/tmp/store
    restart: always

  boki-storage:
    image: {image_faas}
    hostname: faas-storage-{{{{.Task.Slot}}}}
    entrypoint:
      - /boki/storage
      - --zookeeper_host=zookeeper:2181
      - --listen_iface=eth0
      - --db_path=/tmp/storage/logdata
      - --num_io_workers=2
      - --io_uring_entries=2048
      - --io_uring_fd_slots=4096
      - --slog_local_cut_interval_us=300
      - --slog_storage_bgthread_interval_ms=1
      - --slog_storage_backend=rocksdb
      - --slog_storage_cache_cap_mb=4096
      # - --v=1
    depends_on:
      - zookeeper-setup
    volumes:
      - /mnt/storage:/tmp/storage
    environment:
      - FAAS_NODE_ID={{{{.Task.Slot}}}}
    restart: always

  boki-sequencer:
    image: {image_faas}
    hostname: faas-sequencer-{{{{.Task.Slot}}}}
    entrypoint:
      - /boki/sequencer
      - --zookeeper_host=zookeeper:2181
      - --listen_iface=eth0
      - --num_io_workers=2
      - --io_uring_entries=2048
      - --io_uring_fd_slots=4096
      - --slog_global_cut_interval_us=300
      # - --v=1
    depends_on:
      - zookeeper-setup
    environment:
      - FAAS_NODE_ID={{{{.Task.Slot}}}}
    restart: always

"""

microbench_docker_compose_f = """\
  bokilogappend-fn:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "/microbench-bin/log_rw", "1"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=2
      - GOGC=200
    depends_on:
      - boki-engine
    restart: always

  asynclogappend-fn:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "/microbench-bin/log_rw", "2"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=2
      - GOGC=200
    depends_on:
      - boki-engine
    restart: always

  bokilogread-fn:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "/microbench-bin/log_rw", "3"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=2
      - GOGC=200
    depends_on:
      - boki-engine
    restart: always

  asynclogread-fn:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "/microbench-bin/log_rw", "4"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=2
      - GOGC=200
    depends_on:
      - boki-engine
    restart: always

  ipcbench-fn:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "/microbench-bin/log_rw", "5"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=2
      - GOGC=200
    depends_on:
      - boki-engine
    restart: always

"""

microbench_nightcore_config = """\
[
    { "funcName": "benchBokiLogAppend", "funcId": 1, "minWorkers": 32, "maxWorkers": 32 },
    { "funcName": "benchAsyncLogAppend", "funcId": 2, "minWorkers": 32, "maxWorkers": 32 },
    { "funcName": "benchBokiLogRead", "funcId": 3, "minWorkers": 32, "maxWorkers": 32 },
    { "funcName": "benchAsyncLogRead", "funcId": 4, "minWorkers": 32, "maxWorkers": 32 },
    { "funcName": "ipcBench", "funcId": 5, "minWorkers": 32, "maxWorkers": 32 }
]
"""

microbench_run_once_f = """\
#!/bin/bash
set -euxo pipefail

BASE_DIR=`realpath $(dirname $0)`
ROOT_DIR=`realpath $BASE_DIR/../..`

EXP_DIR=$BASE_DIR/results/$1

BENCH_CASE=$2
NUM_CONCURRENCY=$3
NUM_BATCHSIZE=$4

HELPER_SCRIPT=$ROOT_DIR/scripts/exp_helper

MANAGER_HOST=`$HELPER_SCRIPT get-docker-manager-host --base-dir=$BASE_DIR`
CLIENT_HOST=`$HELPER_SCRIPT get-client-host --base-dir=$BASE_DIR`
ENTRY_HOST=`$HELPER_SCRIPT get-service-host --base-dir=$BASE_DIR --service=boki-gateway`
ALL_HOSTS=`$HELPER_SCRIPT get-all-server-hosts --base-dir=$BASE_DIR`

$HELPER_SCRIPT generate-docker-compose --base-dir=$BASE_DIR
scp -q $BASE_DIR/docker-compose.yml $MANAGER_HOST:/tmp
scp -q $BASE_DIR/docker-compose-generated.yml $MANAGER_HOST:/tmp

ssh -q $MANAGER_HOST -- docker stack rm boki-experiment

sleep 40

scp -q $ROOT_DIR/scripts/zk_setup.sh $MANAGER_HOST:/tmp/zk_setup.sh
ssh -q $MANAGER_HOST -- sudo mkdir -p /mnt/inmem/store

for host in $ALL_HOSTS; do
    scp -q $BASE_DIR/nightcore_config.json $host:/tmp/nightcore_config.json
done

ALL_ENGINE_HOSTS=`$HELPER_SCRIPT get-machine-with-label --base-dir=$BASE_DIR --machine-label=engine_node`
for HOST in $ALL_ENGINE_HOSTS; do
    scp -q $BASE_DIR/run_launcher $HOST:/tmp/run_launcher
    ssh -q $HOST -- sudo rm -rf /mnt/inmem/boki
    ssh -q $HOST -- sudo mkdir -p /mnt/inmem/boki
    ssh -q $HOST -- sudo mkdir -p /mnt/inmem/boki/output /mnt/inmem/boki/ipc
    ssh -q $HOST -- sudo cp /tmp/run_launcher /mnt/inmem/boki/run_launcher
    ssh -q $HOST -- sudo cp /tmp/nightcore_config.json /mnt/inmem/boki/func_config.json
done

ALL_STORAGE_HOSTS=`$HELPER_SCRIPT get-machine-with-label --base-dir=$BASE_DIR --machine-label=storage_node`
for HOST in $ALL_STORAGE_HOSTS; do
    ssh -q $HOST -- sudo rm -rf   /mnt/storage/logdata
    ssh -q $HOST -- sudo mkdir -p /mnt/storage/logdata
done

ssh -q $MANAGER_HOST -- docker stack deploy \\
    -c /tmp/docker-compose-generated.yml -c /tmp/docker-compose.yml boki-experiment
sleep 60

for HOST in $ALL_ENGINE_HOSTS; do
    ENGINE_CONTAINER_ID=`$HELPER_SCRIPT get-container-id --base-dir=$BASE_DIR --service boki-engine --machine-host $HOST`
    echo 4096 | ssh -q $HOST -- sudo tee /sys/fs/cgroup/cpu,cpuacct/docker/$ENGINE_CONTAINER_ID/cpu.shares
done

sleep 10

rm -rf $EXP_DIR
mkdir -p $EXP_DIR

ssh -q $MANAGER_HOST -- cat /proc/cmdline >>$EXP_DIR/kernel_cmdline
ssh -q $MANAGER_HOST -- uname -a >>$EXP_DIR/kernel_version

ssh -q $CLIENT_HOST -- docker run -v /tmp:/tmp \\
    {image_app} \\
    cp /microbench-bin/benchmark /tmp/benchmark

ssh -q $CLIENT_HOST -- /tmp/benchmark \\
    --faas_gateway=$ENTRY_HOST:8080 --bench_case=$BENCH_CASE \\
    --batch_size=$NUM_BATCHSIZE --concurrency=$NUM_CONCURRENCY \\
    --payload_size=1024 --duration=180 >$EXP_DIR/results.log

$HELPER_SCRIPT collect-container-logs --base-dir=$BASE_DIR --log-path=$EXP_DIR/logs

cd /tmp
mkdir -p $EXP_DIR/fn_output
for HOST in $ALL_ENGINE_HOSTS; do
    ssh -q $HOST -- sudo tar -czf /tmp/output.tar.gz /mnt/inmem/boki/output
    scp -q $HOST:/tmp/output.tar.gz /tmp
    tar -zxf /tmp/output.tar.gz && mv mnt $HOST && mv $HOST $EXP_DIR/fn_output
done
cd -

"""

queue_docker_compose_f = """\
  consumer-fn:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "/queuebench-bin/main", "2"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=2
      - GOGC=200
    depends_on:
      - boki-engine
    restart: always

  producer-fn:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "/queuebench-bin/main", "1"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=2
      - GOGC=200
    depends_on:
      - boki-engine
    restart: always

"""

queue_nightcore_config = """\
[
    { "funcName": "slibQueueProducer", "funcId": 1, "minWorkers": 32, "maxWorkers": 32 },
    { "funcName": "slibQueueConsumer", "funcId": 2, "minWorkers": 32, "maxWorkers": 32 }
]
"""

queue_run_once_f = """\
#!/bin/bash
set -euxo pipefail

BASE_DIR=`realpath $(dirname $0)`
ROOT_DIR=`realpath $BASE_DIR/../../..`

EXP_DIR=$BASE_DIR/results/$1

NUM_SHARDS=$2
INTERVAL1=$3
INTERVAL2=$4
NUM_PRODUCER=$5
if [[ $# == 6 ]]; then
    NUM_PBATCHSIZE=$6   # producer batch size
else
    NUM_PBATCHSIZE=1
fi
NUM_CONSUMER=$NUM_SHARDS

HELPER_SCRIPT=$ROOT_DIR/scripts/exp_helper

MANAGER_HOST=`$HELPER_SCRIPT get-docker-manager-host --base-dir=$BASE_DIR`
CLIENT_HOST=`$HELPER_SCRIPT get-client-host --base-dir=$BASE_DIR`
ENTRY_HOST=`$HELPER_SCRIPT get-service-host --base-dir=$BASE_DIR --service=boki-gateway`
ALL_HOSTS=`$HELPER_SCRIPT get-all-server-hosts --base-dir=$BASE_DIR`

$HELPER_SCRIPT generate-docker-compose --base-dir=$BASE_DIR
scp -q $BASE_DIR/docker-compose.yml $MANAGER_HOST:/tmp
scp -q $BASE_DIR/docker-compose-generated.yml $MANAGER_HOST:/tmp

ssh -q $MANAGER_HOST -- docker stack rm boki-experiment

sleep 40

scp -q $ROOT_DIR/scripts/zk_setup.sh $MANAGER_HOST:/tmp/zk_setup.sh
ssh -q $MANAGER_HOST -- sudo mkdir -p /mnt/inmem/store

for host in $ALL_HOSTS; do
    scp -q $BASE_DIR/nightcore_config.json $host:/tmp/nightcore_config.json
done

ALL_ENGINE_HOSTS=`$HELPER_SCRIPT get-machine-with-label --base-dir=$BASE_DIR --machine-label=engine_node`
for HOST in $ALL_ENGINE_HOSTS; do
    scp -q $BASE_DIR/run_launcher $HOST:/tmp/run_launcher
    ssh -q $HOST -- sudo rm -rf /mnt/inmem/boki
    ssh -q $HOST -- sudo mkdir -p /mnt/inmem/boki
    ssh -q $HOST -- sudo mkdir -p /mnt/inmem/boki/output /mnt/inmem/boki/ipc
    ssh -q $HOST -- sudo cp /tmp/run_launcher /mnt/inmem/boki/run_launcher
    ssh -q $HOST -- sudo cp /tmp/nightcore_config.json /mnt/inmem/boki/func_config.json
done

ALL_STORAGE_HOSTS=`$HELPER_SCRIPT get-machine-with-label --base-dir=$BASE_DIR --machine-label=storage_node`
for HOST in $ALL_STORAGE_HOSTS; do
    ssh -q $HOST -- sudo rm -rf   /mnt/storage/logdata
    ssh -q $HOST -- sudo mkdir -p /mnt/storage/logdata
done

ssh -q $MANAGER_HOST -- docker stack deploy \\
    -c /tmp/docker-compose-generated.yml -c /tmp/docker-compose.yml boki-experiment
sleep 60

for HOST in $ALL_ENGINE_HOSTS; do
    ENGINE_CONTAINER_ID=`$HELPER_SCRIPT get-container-id --base-dir=$BASE_DIR --service boki-engine --machine-host $HOST`
    echo 4096 | ssh -q $HOST -- sudo tee /sys/fs/cgroup/cpu,cpuacct/docker/$ENGINE_CONTAINER_ID/cpu.shares
done

QUEUE_PREFIX=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 8 | head -n 1 || true)

sleep 10

rm -rf $EXP_DIR
mkdir -p $EXP_DIR

ssh -q $MANAGER_HOST -- cat /proc/cmdline >>$EXP_DIR/kernel_cmdline
ssh -q $MANAGER_HOST -- uname -a >>$EXP_DIR/kernel_version

ssh -q $CLIENT_HOST -- docker run -v /tmp:/tmp \\
    {image_app} \\
    cp /queuebench-bin/benchmark /tmp/benchmark

ssh -q $CLIENT_HOST -- /tmp/benchmark \\
    --faas_gateway=$ENTRY_HOST:8080 --fn_prefix=slib \\
    --queue_prefix=$QUEUE_PREFIX --num_queues=1 --queue_shards=$NUM_SHARDS \\
    --num_producer=$NUM_PRODUCER --num_consumer=$NUM_CONSUMER \\
    --producer_interval=$INTERVAL1 --consumer_interval=$INTERVAL2 \\
    --producer_bsize=$NUM_PBATCHSIZE \\
    --consumer_fix_shard=true \\
    --payload_size=1024 --duration=180 >$EXP_DIR/results.log

$HELPER_SCRIPT collect-container-logs --base-dir=$BASE_DIR --log-path=$EXP_DIR/logs

cd /tmp
mkdir -p $EXP_DIR/fn_output
for HOST in $ALL_ENGINE_HOSTS; do
    ssh -q $HOST -- sudo tar -czf /tmp/output.tar.gz /mnt/inmem/boki/output
    scp -q $HOST:/tmp/output.tar.gz /tmp
    tar -zxf /tmp/output.tar.gz && mv mnt $HOST && mv $HOST $EXP_DIR/fn_output
done
cd -

"""

retwis_docker_compose_f = """\
  retwis-init:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "/retwisbench-bin/main", "1"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=200
    depends_on:
      - boki-engine
    restart: always

  retwis-register:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "/retwisbench-bin/main", "2"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=200
    depends_on:
      - boki-engine
    restart: always

  retwis-login:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "/retwisbench-bin/main", "3"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=200
    depends_on:
      - boki-engine
    restart: always

  retwis-profile:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "/retwisbench-bin/main", "4"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=200
    depends_on:
      - boki-engine
    restart: always

  retwis-follow:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "/retwisbench-bin/main", "5"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=200
    depends_on:
      - boki-engine
    restart: always

  retwis-post:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "/retwisbench-bin/main", "6"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=200
    depends_on:
      - boki-engine
    restart: always

  retwis-post-list:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "/retwisbench-bin/main", "7"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=200
    depends_on:
      - boki-engine
    restart: always

"""

retwis_nightcore_config = """\
[
    { "funcName": "RetwisInit", "funcId": 1, "minWorkers": 1, "maxWorkers": 1 },
    { "funcName": "RetwisRegister", "funcId": 2, "minWorkers": 24, "maxWorkers": 24 },
    { "funcName": "RetwisLogin", "funcId": 3, "minWorkers": 24, "maxWorkers": 24 },
    { "funcName": "RetwisProfile", "funcId": 4, "minWorkers": 24, "maxWorkers": 24 },
    { "funcName": "RetwisFollow", "funcId": 5, "minWorkers": 24, "maxWorkers": 24 },
    { "funcName": "RetwisPost", "funcId": 6, "minWorkers": 24, "maxWorkers": 24 },
    { "funcName": "RetwisPostList", "funcId": 7, "minWorkers": 24, "maxWorkers": 24 }
]
"""

retwis_run_once_f = """\
#!/bin/bash
set -euxo pipefail

BASE_DIR=`realpath $(dirname $0)`
ROOT_DIR=`realpath $BASE_DIR/../../..`

EXP_DIR=$BASE_DIR/results/$1

CONCURRENCY=$2
NUM_USERS=10000

HELPER_SCRIPT=$ROOT_DIR/scripts/exp_helper

MANAGER_HOST=`$HELPER_SCRIPT get-docker-manager-host --base-dir=$BASE_DIR`
CLIENT_HOST=`$HELPER_SCRIPT get-client-host --base-dir=$BASE_DIR`
ENTRY_HOST=`$HELPER_SCRIPT get-service-host --base-dir=$BASE_DIR --service=boki-gateway`
ALL_HOSTS=`$HELPER_SCRIPT get-all-server-hosts --base-dir=$BASE_DIR`

$HELPER_SCRIPT generate-docker-compose --base-dir=$BASE_DIR
scp -q $BASE_DIR/docker-compose.yml $MANAGER_HOST:/tmp
scp -q $BASE_DIR/docker-compose-generated.yml $MANAGER_HOST:/tmp

ssh -q $MANAGER_HOST -- docker stack rm boki-experiment

sleep 40

scp -q $ROOT_DIR/scripts/zk_setup.sh $MANAGER_HOST:/tmp/zk_setup.sh
ssh -q $MANAGER_HOST -- sudo mkdir -p /mnt/inmem/store

for host in $ALL_HOSTS; do
    scp -q $BASE_DIR/nightcore_config.json $host:/tmp/nightcore_config.json
done

ALL_ENGINE_HOSTS=`$HELPER_SCRIPT get-machine-with-label --base-dir=$BASE_DIR --machine-label=engine_node`
for HOST in $ALL_ENGINE_HOSTS; do
    scp -q $BASE_DIR/run_launcher $HOST:/tmp/run_launcher
    ssh -q $HOST -- sudo rm -rf /mnt/inmem/boki
    ssh -q $HOST -- sudo mkdir -p /mnt/inmem/boki
    ssh -q $HOST -- sudo mkdir -p /mnt/inmem/boki/output /mnt/inmem/boki/ipc
    ssh -q $HOST -- sudo cp /tmp/run_launcher /mnt/inmem/boki/run_launcher
    ssh -q $HOST -- sudo cp /tmp/nightcore_config.json /mnt/inmem/boki/func_config.json
done

ALL_STORAGE_HOSTS=`$HELPER_SCRIPT get-machine-with-label --base-dir=$BASE_DIR --machine-label=storage_node`
for HOST in $ALL_STORAGE_HOSTS; do
    ssh -q $HOST -- sudo rm -rf   /mnt/storage/logdata
    ssh -q $HOST -- sudo mkdir -p /mnt/storage/logdata
done

ssh -q $MANAGER_HOST -- docker stack deploy \\
    -c /tmp/docker-compose-generated.yml -c /tmp/docker-compose.yml boki-experiment
sleep 60

for HOST in $ALL_ENGINE_HOSTS; do
    ENGINE_CONTAINER_ID=`$HELPER_SCRIPT get-container-id --base-dir=$BASE_DIR --service boki-engine --machine-host $HOST`
    echo 4096 | ssh -q $HOST -- sudo tee /sys/fs/cgroup/cpu,cpuacct/docker/$ENGINE_CONTAINER_ID/cpu.shares
done

sleep 10

rm -rf $EXP_DIR
mkdir -p $EXP_DIR

ssh -q $MANAGER_HOST -- cat /proc/cmdline >>$EXP_DIR/kernel_cmdline
ssh -q $MANAGER_HOST -- uname -a >>$EXP_DIR/kernel_version

ssh -q $CLIENT_HOST -- curl -X POST http://$ENTRY_HOST:8080/function/RetwisInit

ssh -q $CLIENT_HOST -- docker run -v /tmp:/tmp \\
    {image_app} \\
    cp /retwisbench-bin/create_users /tmp/create_users

ssh -q $CLIENT_HOST -- /tmp/create_users \\
    --faas_gateway=$ENTRY_HOST:8080 --num_users=$NUM_USERS --concurrency=16

ssh -q $CLIENT_HOST -- docker run -v /tmp:/tmp \\
    {image_app} \\
    cp /retwisbench-bin/benchmark /tmp/benchmark

ssh -q $CLIENT_HOST -- /tmp/benchmark \\
    --faas_gateway=$ENTRY_HOST:8080 --num_users=$NUM_USERS \\
    --percentages=15,30,50,5 \\
    --duration=180 --concurrency=$CONCURRENCY >$EXP_DIR/results.log

$HELPER_SCRIPT collect-container-logs --base-dir=$BASE_DIR --log-path=$EXP_DIR/logs

cd /tmp
mkdir -p $EXP_DIR/fn_output
for HOST in $ALL_ENGINE_HOSTS; do
    ssh -q $HOST -- sudo tar -czf /tmp/output.tar.gz /mnt/inmem/boki/output
    scp -q $HOST:/tmp/output.tar.gz /tmp
    tar -zxf /tmp/output.tar.gz && mv mnt $HOST && mv $HOST $EXP_DIR/fn_output
done
cd -

"""

workflow_hotel_docker_compose_f = """\
  geo-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/geo", "1"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  profile-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/profile", "2"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  rate-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/rate", "3"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  recommendation-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/recommendation", "4"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  user-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/user", "5"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  hotel-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/hotel", "6"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  search-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/search", "7"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  flight-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/flight", "8"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  order-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/order", "9"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  frontend-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/frontend", "10"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  gateway-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/gateway", "11"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

"""

workflow_movie_docker_compose_f = """\
  frontend:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/Frontend", "1"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  cast-info-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/CastInfo", "2"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  review-storage-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/ReviewStorage", "3"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  user-review-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/UserReview", "4"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  movie-review-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/MovieReview", "5"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  compose-review-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/ComposeReview", "6"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  text-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/Text", "7"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  user-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/User", "8"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  unique-id-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/UniqueId", "9"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  rating-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/Rating", "10"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  movie-id-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/MovieId", "11"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  plot-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/Plot", "12"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  movie-info-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/MovieInfo", "13"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  page-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/Page", "14"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=4
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

"""

workflow_singleop_docker_compose_f = """\
  singleop-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/singleop", "1"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  nop-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/nop", "2"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

"""

workflow_txnbench_docker_compose_f = """\
  dbops-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/dbops", "1"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  readonly-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/readonly", "2"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

  writeonly-service:
    image: {image_app}
    entrypoint: ["/tmp/boki/run_launcher", "{bin_path}/writeonly", "3"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
      - TABLE_PREFIX=${{TABLE_PREFIX:?}}
    depends_on:
      - boki-engine
    restart: always

"""

hotel_nightcore_config_f = """\
[
    {{ "funcName": "{baseline_prefix}geo", "funcId": 1, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}profile", "funcId": 2, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}rate", "funcId": 3, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}recommendation", "funcId": 4, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}user", "funcId": 5, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}hotel", "funcId": 6, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}search", "funcId": 7, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}flight", "funcId": 8, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}order", "funcId": 9, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}frontend", "funcId": 10, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}gateway", "funcId": 11, "minWorkers": 8, "maxWorkers": 8 }}
]
"""

movie_nightcore_config_f = """\
[
    {{ "funcName": "{baseline_prefix}Frontend", "funcId": 1, "minWorkers": 16, "maxWorkers": 16 }},
    {{ "funcName": "{baseline_prefix}CastInfo", "funcId": 2, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}ReviewStorage", "funcId": 3, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}UserReview", "funcId": 4, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}MovieReview", "funcId": 5, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}ComposeReview", "funcId": 6, "minWorkers": 16, "maxWorkers": 16 }},
    {{ "funcName": "{baseline_prefix}Text", "funcId": 7, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}User", "funcId": 8, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}UniqueId", "funcId": 9, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}Rating", "funcId": 10, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}MovieId", "funcId": 11, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}Plot", "funcId": 12, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}MovieInfo", "funcId": 13, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}Page", "funcId": 14, "minWorkers": 8, "maxWorkers": 8 }}
]
"""

singleop_nightcore_config_f = """\
[
    {{ "funcName": "{baseline_prefix}singleop", "funcId": 1, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}nop", "funcId": 2, "minWorkers": 8, "maxWorkers": 8 }}
]
"""

txnbench_nightcore_config_f = """\
[
    {{ "funcName": "{baseline_prefix}dbops", "funcId": 1, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}readonly", "funcId": 2, "minWorkers": 8, "maxWorkers": 8 }},
    {{ "funcName": "{baseline_prefix}writeonly", "funcId": 3, "minWorkers": 8, "maxWorkers": 8 }}
]
"""

workflow_run_once_sh_f = """\
#!/bin/bash
set -euxo pipefail

BASE_DIR=`realpath $(dirname $0)`
ROOT_DIR=`realpath $BASE_DIR/../../..`

AWS_REGION=us-east-2

EXP_DIR=$BASE_DIR/results/$1
QPS=$2

HELPER_SCRIPT=$ROOT_DIR/scripts/exp_helper
WRK_DIR=/usr/local/bin

MANAGER_HOST=`$HELPER_SCRIPT get-docker-manager-host --base-dir=$BASE_DIR`
CLIENT_HOST=`$HELPER_SCRIPT get-client-host --base-dir=$BASE_DIR`
ENTRY_HOST=`$HELPER_SCRIPT get-service-host --base-dir=$BASE_DIR --service=boki-gateway`
ALL_HOSTS=`$HELPER_SCRIPT get-all-server-hosts --base-dir=$BASE_DIR`

$HELPER_SCRIPT generate-docker-compose --base-dir=$BASE_DIR
scp -q $BASE_DIR/docker-compose.yml $MANAGER_HOST:/tmp
scp -q $BASE_DIR/docker-compose-generated.yml $MANAGER_HOST:/tmp

ssh -q $MANAGER_HOST -- docker stack rm boki-experiment

sleep 40

TABLE_PREFIX=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 8 | head -n 1 || true)
TABLE_PREFIX="${{TABLE_PREFIX}}-"

ssh -q $CLIENT_HOST -- docker run -v /tmp:/tmp \\
    {image_app} cp -r {bin_path} /tmp
ssh -q $CLIENT_HOST -- docker run -v /tmp:/tmp \\
    {image_app} cp /bokiflow/data/compressed.json /tmp

ssh -q $CLIENT_HOST -- TABLE_PREFIX=$TABLE_PREFIX AWS_REGION=$AWS_REGION \\
    /tmp/{app_dir}/init create {init_mode}
ssh -q $CLIENT_HOST -- TABLE_PREFIX=$TABLE_PREFIX AWS_REGION=$AWS_REGION \\
    /tmp/{app_dir}/init populate {init_mode} /tmp/compressed.json

scp -q $ROOT_DIR/scripts/zk_setup.sh $MANAGER_HOST:/tmp/zk_setup.sh
ssh -q $MANAGER_HOST -- sudo mkdir -p /mnt/inmem/store

for host in $ALL_HOSTS; do
    scp -q $BASE_DIR/nightcore_config.json $host:/tmp/nightcore_config.json
done

ALL_ENGINE_HOSTS=`$HELPER_SCRIPT get-machine-with-label --base-dir=$BASE_DIR --machine-label=engine_node`
for HOST in $ALL_ENGINE_HOSTS; do
    scp -q $BASE_DIR/run_launcher $HOST:/tmp/run_launcher
    ssh -q $HOST -- sudo rm -rf /mnt/inmem/boki
    ssh -q $HOST -- sudo mkdir -p /mnt/inmem/boki
    ssh -q $HOST -- sudo mkdir -p /mnt/inmem/boki/output /mnt/inmem/boki/ipc
    ssh -q $HOST -- sudo cp /tmp/run_launcher /mnt/inmem/boki/run_launcher
    ssh -q $HOST -- sudo cp /tmp/nightcore_config.json /mnt/inmem/boki/func_config.json
done

ALL_STORAGE_HOSTS=`$HELPER_SCRIPT get-machine-with-label --base-dir=$BASE_DIR --machine-label=storage_node`
for HOST in $ALL_STORAGE_HOSTS; do
    ssh -q $HOST -- sudo rm -rf   /mnt/storage/logdata
    ssh -q $HOST -- sudo mkdir -p /mnt/storage/logdata
done

ssh -q $MANAGER_HOST -- TABLE_PREFIX=$TABLE_PREFIX docker stack deploy \\
    -c /tmp/docker-compose-generated.yml -c /tmp/docker-compose.yml boki-experiment
sleep 60

for HOST in $ALL_ENGINE_HOSTS; do
    ENGINE_CONTAINER_ID=`$HELPER_SCRIPT get-container-id --base-dir=$BASE_DIR --service boki-engine --machine-host $HOST`
    echo 4096 | ssh -q $HOST -- sudo tee /sys/fs/cgroup/cpu,cpuacct/docker/$ENGINE_CONTAINER_ID/cpu.shares
done
sleep 10

rm -rf $EXP_DIR
mkdir -p $EXP_DIR

ssh -q $MANAGER_HOST -- cat /proc/cmdline >>$EXP_DIR/kernel_cmdline
ssh -q $MANAGER_HOST -- uname -a >>$EXP_DIR/kernel_version

scp -q $ROOT_DIR/workloads/workflow/{workflow_dir}/benchmark/{bench_dir}/workload.lua $CLIENT_HOST:/tmp

ssh -q $CLIENT_HOST -- {wrk_env} $WRK_DIR/wrk -t 2 -c 2 -d 30 -L -U \\
    -s /tmp/workload.lua \\
    http://$ENTRY_HOST:8080 -R $QPS >$EXP_DIR/wrk_warmup.log

sleep 10

ssh -q $CLIENT_HOST -- {wrk_env} $WRK_DIR/wrk -t 2 -c 2 -d 150 -L -U \\
    -s /tmp/workload.lua \\
    http://$ENTRY_HOST:8080 -R $QPS 2>/dev/null >$EXP_DIR/wrk.log

sleep 10

scp -q $MANAGER_HOST:/mnt/inmem/store/async_results $EXP_DIR
$ROOT_DIR/scripts/compute_latency.py --async-result-file $EXP_DIR/async_results >$EXP_DIR/latency.txt

ssh -q $CLIENT_HOST -- TABLE_PREFIX=$TABLE_PREFIX AWS_REGION=$AWS_REGION \\
    /tmp/{app_dir}/init clean {init_mode}

$HELPER_SCRIPT collect-container-logs --base-dir=$BASE_DIR --log-path=$EXP_DIR/logs

cd /tmp
mkdir -p $EXP_DIR/fn_output
for HOST in $ALL_ENGINE_HOSTS; do
    ssh -q $HOST -- sudo tar -czf /tmp/output.tar.gz /mnt/inmem/boki/output
    scp -q $HOST:/tmp/output.tar.gz /tmp
    tar -zxf /tmp/output.tar.gz && mv mnt $HOST && mv $HOST $EXP_DIR/fn_output
done
cd -

"""


def microbench_config(image_faas, image_app):
    docker_compose_faas_f = docker_compose_faas_boki_f
    docker_compose_app_f = microbench_docker_compose_f
    docker_compose = (docker_compose_common +
                      docker_compose_faas_f.format(image_faas=image_faas,
                                                   n_metalog_replicas=3,
                                                   n_userlog_replicas=3,
                                                   n_index_replicas=8) +
                      docker_compose_app_f.format(image_app=image_app))
    config_json = microbench_nightcore_config
    run_once_sh = microbench_run_once_f.format(image_app=image_app)
    return docker_compose, config_json, run_once_sh


def queue_config(image_faas, image_app):
    docker_compose_faas_f = docker_compose_faas_boki_f
    docker_compose_app_f = queue_docker_compose_f
    docker_compose = (docker_compose_common +
                      docker_compose_faas_f.format(image_faas=image_faas,
                                                   n_metalog_replicas=3,
                                                   n_userlog_replicas=3,
                                                   n_index_replicas=8) +
                      docker_compose_app_f.format(image_app=image_app))
    config_json = queue_nightcore_config
    run_once_sh = queue_run_once_f.format(image_app=image_app)
    return docker_compose, config_json, run_once_sh


def retwis_config(image_faas, image_app):
    docker_compose_faas_f = docker_compose_faas_boki_f
    docker_compose_app_f = retwis_docker_compose_f
    docker_compose = (docker_compose_common +
                      docker_compose_faas_f.format(image_faas=image_faas,
                                                   n_metalog_replicas=3,
                                                   n_userlog_replicas=3,
                                                   n_index_replicas=8) +
                      docker_compose_app_f.format(image_app=image_app))
    config_json = retwis_nightcore_config
    run_once_sh = retwis_run_once_f.format(image_app=image_app)
    return docker_compose, config_json, run_once_sh


def workflow_config(image_faas, image_app, bin_path, db_init_mode, enable_sharedlog):
    # e.g. bin_path = "/beldi-bin/bhotel"
    # then app_name = "bhotel"
    #      workflow_name = "beldi"
    #      bench_name = "hotel"
    bin_path_parts = bin_path.split('/')
    assert len(bin_path_parts) == 3
    assert len(bin_path_parts[1].split('-')) == 2

    workflow_name = bin_path_parts[1].split('-')[0]
    if workflow_name == "bokiflow":
        workflow_dir = "boki"
    else:
        workflow_dir = workflow_name

    app_name = bin_path_parts[2]
    app_dir = app_name

    bench_name = app_name.lstrip('b')
    bench_dir = bench_name

    if db_init_mode == "baseline":
        baseline_prefix = 'b'
        wrk_env = "BASELINE=1"
    else:
        baseline_prefix = ''
        wrk_env = ""

    docker_compose_faas_f = docker_compose_faas_boki_f if enable_sharedlog else docker_compose_faas_nightcore_f
    if bench_name == "hotel":
        docker_compose_app_f = workflow_hotel_docker_compose_f
        config_json_f = hotel_nightcore_config_f
        n_index_replicas = 8
    elif bench_name == 'media':
        docker_compose_app_f = workflow_movie_docker_compose_f
        config_json_f = movie_nightcore_config_f
        n_index_replicas = 8
    elif bench_name == 'singleop':
        docker_compose_app_f = workflow_singleop_docker_compose_f
        config_json_f = singleop_nightcore_config_f
        n_index_replicas = 1
    elif bench_name == 'txnbench':
        docker_compose_app_f = workflow_txnbench_docker_compose_f
        config_json_f = txnbench_nightcore_config_f
        n_index_replicas = 3
    else:
        raise Exception(f'unreachable bench name: {bench_name}')

    docker_compose = (docker_compose_common +
                      docker_compose_faas_f.format(image_faas=image_faas,
                                                   n_metalog_replicas=3,
                                                   n_userlog_replicas=3,
                                                   n_index_replicas=n_index_replicas) +
                      docker_compose_app_f.format(image_app=image_app, bin_path=bin_path))
    config_json = config_json_f.format(baseline_prefix=baseline_prefix)
    run_once_sh = workflow_run_once_sh_f.format(image_app=image_app,
                                                bin_path=bin_path,
                                                app_dir=app_dir,
                                                init_mode=db_init_mode,
                                                workflow_dir=workflow_dir,
                                                bench_dir=bench_dir,
                                                wrk_env=wrk_env)
    return docker_compose, config_json, run_once_sh


def dump_configs(dump_dir, config_generator):
    docker_compose, config_json, run_once_sh = config_generator()
    with open(dump_dir / "docker-compose.yml", "w") as f:
        f.write(docker_compose)
    with open(dump_dir / "nightcore_config.json", "w") as f:
        f.write(config_json)
    with open(dump_dir / "run_once.sh", "w") as f:
        f.write(run_once_sh)
    os.chmod(dump_dir / "run_once.sh", 0o777)


def beldi_hotel_baseline():
    return workflow_config(IMAGE_FAAS, WORKFLOW_IMAGE_APP, bin_path="/beldi-bin/bhotel", db_init_mode="baseline", enable_sharedlog=False)


def beldi_movie_baseline():
    return workflow_config(IMAGE_FAAS, WORKFLOW_IMAGE_APP, bin_path="/beldi-bin/bmedia", db_init_mode="baseline", enable_sharedlog=False)


def beldi_hotel():
    return workflow_config(IMAGE_FAAS, WORKFLOW_IMAGE_APP, bin_path="/beldi-bin/hotel", db_init_mode="beldi", enable_sharedlog=False)


def beldi_movie():
    return workflow_config(IMAGE_FAAS, WORKFLOW_IMAGE_APP, bin_path="/beldi-bin/media", db_init_mode="beldi", enable_sharedlog=False)


def boki_hotel_baseline():
    return workflow_config(IMAGE_FAAS, WORKFLOW_IMAGE_APP, bin_path="/bokiflow-bin/hotel", db_init_mode="cayon", enable_sharedlog=True)


def boki_movie_baseline():
    return workflow_config(IMAGE_FAAS, WORKFLOW_IMAGE_APP, bin_path="/bokiflow-bin/media", db_init_mode="cayon", enable_sharedlog=True)


def boki_hotel_asynclog():
    return workflow_config(IMAGE_FAAS, WORKFLOW_IMAGE_APP, bin_path="/asynclog-bin/hotel", db_init_mode="cayon", enable_sharedlog=True)


def boki_movie_asynclog():
    return workflow_config(IMAGE_FAAS, WORKFLOW_IMAGE_APP, bin_path="/asynclog-bin/media", db_init_mode="cayon", enable_sharedlog=True)


def beldi_singleop_baseline():
    return workflow_config(IMAGE_FAAS, WORKFLOW_IMAGE_APP, bin_path="/beldi-bin/bsingleop", db_init_mode="baseline", enable_sharedlog=False)


def boki_singleop_baseline():
    return workflow_config(IMAGE_FAAS, WORKFLOW_IMAGE_APP, bin_path="/bokiflow-bin/singleop", db_init_mode="cayon", enable_sharedlog=True)


def boki_singleop_asynclog():
    return workflow_config(IMAGE_FAAS, WORKFLOW_IMAGE_APP, bin_path="/asynclog-bin/singleop", db_init_mode="cayon", enable_sharedlog=True)

def boki_txnbench_baseline():
    return workflow_config(IMAGE_FAAS, WORKFLOW_IMAGE_APP, bin_path="/bokiflow-bin/txnbench", db_init_mode="cayon", enable_sharedlog=True)


IMAGE_FAAS = "adjwang/boki:dev"
BENCH_IMAGE_APP = "adjwang/boki-microbench:dev"
QUEUE_IMAGE_APP = "adjwang/boki-queuebench:dev"
RETWIS_IMAGE_APP = "adjwang/boki-retwisbench:dev"
WORKFLOW_IMAGE_APP = "adjwang/boki-beldibench:dev"

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--exp-dir', type=str, required=True,
                        help="e.g. boki-benchmarks/experiments")
    parser.add_argument('--exp-name', type=str, required=True,
                        help="microbench|queue|retwis|workflow")
    args = parser.parse_args()

    if args.exp_name == "microbench":
        microbench_dir = Path(args.exp_dir) / "microbenchmark"
        dump_configs(microbench_dir,
                     partial(microbench_config, IMAGE_FAAS, BENCH_IMAGE_APP))
    elif args.exp_name == "queue":
        queue_dir = Path(args.exp_dir) / "queue"
        dump_configs(queue_dir / "boki",
                     partial(queue_config, IMAGE_FAAS, QUEUE_IMAGE_APP))
    elif args.exp_name == "retwis":
        retwis_dir = Path(args.exp_dir) / "retwis"
        dump_configs(retwis_dir / "boki",
                     partial(retwis_config, IMAGE_FAAS, RETWIS_IMAGE_APP))
    elif args.exp_name == "workflow":
        workflow_dir = Path(args.exp_dir) / "workflow"
        dump_configs(workflow_dir / "beldi-hotel-baseline",
                     beldi_hotel_baseline)
        dump_configs(workflow_dir / "beldi-movie-baseline",
                     beldi_movie_baseline)
        dump_configs(workflow_dir / "beldi-singleop-baseline",
                     beldi_singleop_baseline)
        dump_configs(workflow_dir / "beldi-hotel", beldi_hotel)
        dump_configs(workflow_dir / "beldi-movie", beldi_movie)
        dump_configs(workflow_dir / "boki-hotel-baseline", boki_hotel_baseline)
        dump_configs(workflow_dir / "boki-movie-baseline", boki_movie_baseline)
        dump_configs(workflow_dir / "boki-singleop-baseline",
                     boki_singleop_baseline)
        dump_configs(workflow_dir / "boki-txnbench-baseline",
                     boki_txnbench_baseline)
        dump_configs(workflow_dir / "boki-hotel-asynclog", boki_hotel_asynclog)
        dump_configs(workflow_dir / "boki-movie-asynclog", boki_movie_asynclog)
        dump_configs(workflow_dir / "boki-singleop-asynclog",
                     boki_singleop_asynclog)
    else:
        raise Exception(
            f"unknown exp_name: {args.exp_name}, should be one of microbench|queue|retwis|workflow")
