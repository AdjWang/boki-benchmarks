# !/usr/bin/env python3
# -*- coding: utf-8 -*-
import sys
from pathlib import Path
PROJECT_DIR = Path(sys.argv[0]).parent.parent
sys.path.append(str(PROJECT_DIR))
import common

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

docker_compose_faas_sharedlog_f = """\
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
      - --metalog_replicas={metalog_reps}
      - --userlog_replicas={userlog_reps}
      - --index_replicas={index_reps}
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

run_once_sh_no_data_f = """\
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

ssh -q $CLIENT_HOST -- mkdir -p /tmp/app
ssh -q $CLIENT_HOST -- docker run -v /tmp:/tmp \\
    {image_app} cp -r {app_bin_dir}/. /tmp/app

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

scp -q $ROOT_DIR/workloads/workflow/{workflow_lib_name}/benchmark/{workflow_app_name}/workload.lua $CLIENT_HOST:/tmp

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

run_once_sh_with_data_f = """\
#!/bin/bash
set -euxo pipefail

BASE_DIR=`realpath $(dirname $0)`
ROOT_DIR=`realpath $BASE_DIR/../../..`

AWS_REGION=us-east-2

EXP_DIR=$BASE_DIR/results/$1
QPS=$2
LOGMODE=$3

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

ssh -q $CLIENT_HOST -- mkdir -p /tmp/app
ssh -q $CLIENT_HOST -- docker run -v /tmp:/tmp \\
    {image_app} cp -r {app_bin_dir}/. /tmp/app
ssh -q $CLIENT_HOST -- docker run -v /tmp:/tmp \\
    {image_app} cp /bokiflow/data/compressed.json /tmp

ssh -q $CLIENT_HOST -- TABLE_PREFIX=$TABLE_PREFIX AWS_REGION=$AWS_REGION LoggingMode=$LOGMODE \\
    /tmp/app/init create {init_mode}
ssh -q $CLIENT_HOST -- TABLE_PREFIX=$TABLE_PREFIX AWS_REGION=$AWS_REGION LoggingMode=$LOGMODE \\
    /tmp/app/init populate {init_mode} /tmp/compressed.json

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

ssh -q $MANAGER_HOST -- TABLE_PREFIX=$TABLE_PREFIX LoggingMode=$LOGMODE docker stack deploy \\
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

scp -q $ROOT_DIR/workloads/workflow/{workflow_lib_name}/benchmark/{workflow_app_name}/workload.lua $CLIENT_HOST:/tmp

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

ssh -q $CLIENT_HOST -- TABLE_PREFIX=$TABLE_PREFIX AWS_REGION=$AWS_REGION LoggingMode=$LOGMODE \\
    /tmp/app/init clean {init_mode}

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


def generate_docker_compose(enable_sharedlog,
                            metalog_reps=3,
                            userlog_reps=3,
                            index_reps=8):
    docker_compose_faas_f = docker_compose_faas_sharedlog_f if enable_sharedlog else docker_compose_faas_nightcore_f
    docker_compose = (docker_compose_common +
                      docker_compose_faas_f.format(image_faas=common.IMAGE_FAAS,
                                                   metalog_reps=metalog_reps,
                                                   userlog_reps=userlog_reps,
                                                   index_reps=index_reps))
    return docker_compose


def generate_run_once(fn_bin_dir, data_init_mode, workflow_lib_name, workflow_app_name, wrk_env):
    template = run_once_sh_no_data_f if data_init_mode == "" else run_once_sh_with_data_f
    run_once_sh = template.format(image_app=common.IMAGE_APP,
                                  app_bin_dir=fn_bin_dir,
                                  init_mode=data_init_mode,
                                  workflow_lib_name=workflow_lib_name,
                                  workflow_app_name=workflow_app_name,
                                  wrk_env=wrk_env)
    return run_once_sh
