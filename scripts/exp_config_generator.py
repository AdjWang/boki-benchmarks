# !/usr/bin/env python3
# -*- coding: utf-8 -*-
import os
from pathlib import Path
import argparse

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
      - --metalog_replicas=3
      - --userlog_replicas=3
      - --index_replicas=8
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

docker_compose_app_hotel_f = """\
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

docker_compose_app_movie_f = """\
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

hotel_config_f = """\
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

movie_config_f = """\
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

run_once_sh_f = """\
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
    {image_app} \\
    cp -r {bin_path} /tmp && cp /bokiflow/data /tmp

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

"""


def config_common(image_faas, image_app, bin_path, db_init_mode, enable_sharedlog):
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

    docker_compose_faas_f = docker_compose_faas_sharedlog_f if enable_sharedlog else docker_compose_faas_nightcore_f
    docker_compose_app_f = docker_compose_app_hotel_f if bench_name == "hotel" else docker_compose_app_movie_f

    docker_compose = (docker_compose_common +
                      docker_compose_faas_f.format(image_faas=image_faas) +
                      docker_compose_app_f.format(image_app=image_app, bin_path=bin_path))
    config_json = hotel_config_f.format(baseline_prefix=baseline_prefix)
    run_once_sh = run_once_sh_f.format(image_app=image_app,
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
    return config_common(IMAGE_FAAS, IMAGE_APP, bin_path="/beldi-bin/bhotel", db_init_mode="baseline", enable_sharedlog=False)


def beldi_movie_baseline():
    return config_common(IMAGE_FAAS, IMAGE_APP, bin_path="/beldi-bin/bmedia", db_init_mode="baseline", enable_sharedlog=False)


def beldi_hotel():
    return config_common(IMAGE_FAAS, IMAGE_APP, bin_path="/beldi-bin/hotel", db_init_mode="beldi", enable_sharedlog=False)


def beldi_movie():
    return config_common(IMAGE_FAAS, IMAGE_APP, bin_path="/beldi-bin/media", db_init_mode="beldi", enable_sharedlog=False)


def boki_hotel_baseline():
    return config_common(IMAGE_FAAS, IMAGE_APP, bin_path="/bokiflow-bin/hotel", db_init_mode="cayon", enable_sharedlog=True)


def boki_movie_baseline():
    return config_common(IMAGE_FAAS, IMAGE_APP, bin_path="/bokiflow-bin/media", db_init_mode="cayon", enable_sharedlog=True)


def boki_hotel_asynclog():
    return config_common(IMAGE_FAAS, IMAGE_APP, bin_path="/asynclog-bin/hotel", db_init_mode="cayon", enable_sharedlog=True)


def boki_movie_asynclog():
    return config_common(IMAGE_FAAS, IMAGE_APP, bin_path="/asynclog-bin/media", db_init_mode="cayon", enable_sharedlog=True)


IMAGE_FAAS = "adjwang/boki:dev"
IMAGE_APP = "adjwang/boki-beldibench:dev"

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--workflow-dir', type=str, required=True,
                        help="usually: boki-benchmarks/experiments/workflow")
    args = parser.parse_args()

    workflow_dir = Path(args.workflow_dir)
    dump_configs(workflow_dir / "beldi-hotel-baseline", beldi_hotel_baseline)
    dump_configs(workflow_dir / "beldi-movie-baseline", beldi_movie_baseline)
    dump_configs(workflow_dir / "beldi-hotel", beldi_hotel)
    dump_configs(workflow_dir / "beldi-movie", beldi_movie)
    dump_configs(workflow_dir / "boki-hotel-baseline", boki_hotel_baseline)
    dump_configs(workflow_dir / "boki-movie-baseline", boki_movie_baseline)
    dump_configs(workflow_dir / "boki-hotel-asynclog", boki_hotel_asynclog)
    dump_configs(workflow_dir / "boki-movie-asynclog", boki_movie_asynclog)
