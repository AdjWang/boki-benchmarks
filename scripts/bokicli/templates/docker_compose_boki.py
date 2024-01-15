# !/usr/bin/env python3
# -*- coding: utf-8 -*-
import os
import argparse
import sys
from pathlib import Path
PROJECT_DIR = Path(sys.argv[0]).parent.parent
sys.path.append(str(PROJECT_DIR))
import common

# dc: docker compose file

network_config = """\
networks:
  boki-net:
    driver: bridge

"""

dc_header = """\
version: "3.8"

"""

# services

# dynamodb
# docker run -d -p 8000:8000 amazon/dynamodb-local
dynamodb = """\
  db:
    image: amazon/dynamodb-local
    hostname: dynamodb
    networks:
      - boki-net
    ports:
      - 8000:8000
    # restart: always

"""

dynamodb_setup_hotel_f = """\
  db-setup:
    image: adjwang/boki-beldibench:dev
    networks:
      - boki-net
    command: bash -c "
        {workflow_bin_dir}/{baseline_prefix}hotel/init clean {benchmark_mode} &&
        {workflow_bin_dir}/{baseline_prefix}hotel/init create {benchmark_mode} &&
        {workflow_bin_dir}/{baseline_prefix}hotel/init populate {benchmark_mode} &&
        sleep infinity
      "
    environment:
      {func_env}
      - TABLE_PREFIX={table_prefix}
    depends_on:
       - db
    healthcheck:
      test: ["CMD-SHELL", "{workflow_bin_dir}/{baseline_prefix}hotel/init health_check {benchmark_mode}"]
      interval: 10s
      retries: 5
      start_period: 10s
      timeout: 60s

"""

dynamodb_setup_media_f = """\
  db-setup:
    image: adjwang/boki-beldibench:dev
    networks:
      - boki-net
    command: bash -c "
        {workflow_bin_dir}/{baseline_prefix}media/init clean {benchmark_mode} &&
        {workflow_bin_dir}/{baseline_prefix}media/init create {benchmark_mode} &&
        {workflow_bin_dir}/{baseline_prefix}media/init populate {benchmark_mode} /bokiflow/data/compressed.json &&
        sleep infinity
      "
    environment:
      {func_env}
      - TABLE_PREFIX={table_prefix}
    depends_on:
       - db
    healthcheck:
      test: ["CMD-SHELL", "{workflow_bin_dir}/{baseline_prefix}media/init health_check {benchmark_mode}"]
      interval: 10s
      retries: 5
      start_period: 10s
      timeout: 60s

"""

# zookeeper
zookeeper = """\
  zookeeper:
    image: zookeeper:3.6.2
    hostname: zookeeper
    networks:
      - boki-net
    ports:
      - 2181:2181
    restart: always

"""

zookeeper_dep_db_setup = """\
      db-setup:
        condition: service_healthy
"""

# zookeeper intializer
zookeeper_setup_f = """\
  zookeeper-setup:
    image: zookeeper:3.6.2
    command: /tmp/boki/zk_setup.sh
    depends_on:
      zookeeper:
        condition: service_started
{zookeeper_dep_db_setup}
    volumes:
      - {workdir}/config/zk_setup.sh:/tmp/boki/zk_setup.sh
      - {workdir}/config/zk_health_check:/tmp/boki/zk_health_check
    networks:
      - boki-net
    restart: always
    healthcheck:
      test: ["CMD-SHELL", "/tmp/boki/zk_health_check"]
      interval: 3s
      retries: 5
      start_period: 5s
      timeout: 10s

"""

boki_engine_f = """\
  boki-engine-{node_id}:
    image: adjwang/boki:dev
    hostname: faas-engine-{node_id}
    networks:
      - boki-net
    entrypoint:
      - /boki/engine
      - --zookeeper_host={zookeeper_endpoint}
      - --listen_iface=eth0
      - --root_path_for_ipc=/tmp/boki/ipc
      - --func_config_file=/tmp/boki/func_config.json
      - --num_io_workers=4
      - --instant_rps_p_norm=0.8
      - --io_uring_entries={io_uring_entries}
      - --io_uring_fd_slots={io_uring_fd_slots}
      - --enable_shared_log
      - --slog_engine_enable_cache
      - --slog_engine_cache_cap_mb=512
      - --slog_engine_propagate_auxdata
      - --v={verbose}
    depends_on:
      zookeeper-setup:
        condition: service_healthy
    volumes:
      - {workdir}/mnt/inmem{node_id}/boki:/tmp/boki
      - {workdir}/mnt/inmem{node_id}/gperf:/tmp/gperf
      - /sys/fs/cgroup:/tmp/root_cgroupfs
    environment:
      {bin_env}
      - FAAS_NODE_ID={node_id}
      - FAAS_CGROUP_FS_ROOT=/tmp/root_cgroupfs
    ulimits:
      memlock: -1
    # restart: always

"""

boki_controller_f = """\
  boki-controller:
    image: adjwang/boki:dev
    networks:
      - boki-net
    entrypoint:
      - /boki/controller
      - --zookeeper_host={zookeeper_endpoint}
      - --metalog_replicas={metalog_reps}
      - --userlog_replicas={userlog_reps}
      - --index_replicas={index_reps}
      - --v={verbose}
    depends_on:
      zookeeper-setup:
        condition: service_healthy
    # restart: always

"""

boki_gateway_f = """\
  boki-gateway:
    image: adjwang/boki:dev
    hostname: faas-gateway
    networks:
      - boki-net
    ports:
      - 9000:9000
    entrypoint:
      - /boki/gateway
      - --zookeeper_host={zookeeper_endpoint}
      - --listen_iface=eth0
      - --http_port=9000
      - --func_config_file=/tmp/boki/func_config.json
      - --async_call_result_path=/tmp/store/async_results
      - --num_io_workers=2
      - --io_uring_entries={io_uring_entries}
      - --io_uring_fd_slots={io_uring_fd_slots}
      - --lb_per_fn_round_robin
      - --max_running_requests=0
      - --v={verbose}
    depends_on:
      zookeeper-setup:
        condition: service_healthy
    volumes:
      - {workdir}/config/nightcore_config.json:/tmp/boki/func_config.json 
      - {workdir}/mnt/inmem_gateway/store:/tmp/store
    ulimits:
      memlock: -1
    # restart: always

"""

boki_storage_f = """\
  boki-storage-{node_id}:
    image: adjwang/boki:dev
    hostname: faas-storage-{node_id}
    networks:
      - boki-net
    entrypoint:
      - /boki/storage
      - --zookeeper_host={zookeeper_endpoint}
      - --listen_iface=eth0
      - --db_path=/tmp/storage/logdata
      - --num_io_workers=2
      - --io_uring_entries={io_uring_entries}
      - --io_uring_fd_slots={io_uring_fd_slots}
      - --slog_local_cut_interval_us=300
      - --slog_storage_bgthread_interval_ms=1
      - --slog_storage_backend=rocksdb
      - --slog_storage_cache_cap_mb=512
      - --v={verbose}
    depends_on:
      zookeeper-setup:
        condition: service_healthy
    volumes:
      - {workdir}/mnt/storage{node_id}:/tmp/storage
      - {workdir}/mnt/storage{node_id}/gperf:/tmp/gperf
    environment:
      {bin_env}
      - FAAS_NODE_ID={node_id}
    ulimits:
      memlock: -1
    # restart: always

"""

boki_sequencer_f = """\
  boki-sequencer-{node_id}:
    image: adjwang/boki:dev
    hostname: faas-sequencer-{node_id}
    networks:
      - boki-net
    entrypoint:
      - /boki/sequencer
      - --zookeeper_host={zookeeper_endpoint}
      - --listen_iface=eth0
      - --num_io_workers=2
      - --io_uring_entries={io_uring_entries}
      - --io_uring_fd_slots={io_uring_fd_slots}
      - --slog_global_cut_interval_us=300
      - --v={verbose}
    depends_on:
      zookeeper-setup:
        condition: service_healthy
    volumes:
      - {workdir}/mnt/sequencer{node_id}/gperf:/tmp/gperf
    environment:
      {bin_env}
      - FAAS_NODE_ID={node_id}
    ulimits:
      memlock: -1
    # restart: always

"""

def generate_docker_compose(func_config, work_dir, metalog_reps, userlog_reps, index_reps):
    baseline_prefix = 'b' if func_config.unsafe_baseline else ''
    dc_content = ''.join([
        dc_header,
        network_config,

        "services:\n",
        func_config.db,
        func_config.db_setup_f.format(workflow_bin_dir=func_config.workflow_bin_dir,
                                      func_env=common.LOCAL_FUNC_ENV,
                                      table_prefix=common.TABLE_PREFIX,
                                      benchmark_mode=(
            'baseline' if func_config.unsafe_baseline else 'beldi'),
            baseline_prefix=baseline_prefix),

        zookeeper,
        zookeeper_setup_f.format(
            zookeeper_dep_db_setup=zookeeper_dep_db_setup if func_config.db_setup_f != "" else "",
            workdir=work_dir),

        boki_controller_f.format(
            bin_env='',
            zookeeper_endpoint=common.ZOOKEEPER_ENDPOINT,
            metalog_reps=metalog_reps,
            userlog_reps=userlog_reps,
            index_reps=index_reps,
            verbose=common.VERBOSE
        ),
        boki_gateway_f.format(
            workdir=work_dir,
            bin_env='',
            zookeeper_endpoint=common.ZOOKEEPER_ENDPOINT,
            io_uring_entries=common.IO_URING_ENTRIES,
            io_uring_fd_slots=common.IO_URING_FD_SLOTS,
            verbose=common.VERBOSE
        ),

        *[boki_engine_f.format(
            workdir=work_dir,
            node_id=i,
            bin_env=common.BIN_ENV,
            zookeeper_endpoint=common.ZOOKEEPER_ENDPOINT,
            io_uring_entries=common.IO_URING_ENTRIES,
            io_uring_fd_slots=common.IO_URING_FD_SLOTS,
            verbose=common.VERBOSE
        ) for i in range(1, 1+index_reps)],

        *[boki_storage_f.format(
            workdir=work_dir,
            node_id=i,
            bin_env=common.BIN_ENV,
            zookeeper_endpoint=common.ZOOKEEPER_ENDPOINT,
            io_uring_entries=common.IO_URING_ENTRIES,
            io_uring_fd_slots=common.IO_URING_FD_SLOTS,
            verbose=common.VERBOSE
        ) for i in range(1, 1+userlog_reps)],

        *[boki_sequencer_f.format(
            workdir=work_dir,
            node_id=i,
            bin_env=common.BIN_ENV,
            zookeeper_endpoint=common.ZOOKEEPER_ENDPOINT,
            io_uring_entries=common.IO_URING_ENTRIES,
            io_uring_fd_slots=common.IO_URING_FD_SLOTS,
            verbose=common.VERBOSE
        ) for i in range(1, 1+metalog_reps)],
    ])
    return dc_content


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--metalog-reps', type=int, default=3)
    parser.add_argument('--userlog-reps', type=int, default=3)
    parser.add_argument('--index-reps', type=int, default=3)
    parser.add_argument('--test-case', type=str, default='boki-hotel-baseline')
    parser.add_argument('--table-prefix', type=str, default="23333")
    parser.add_argument('--workdir', type=str, default='/tmp')
    parser.add_argument('--output', type=str, default='/tmp')
    args = parser.parse_args()

    # no beldi-hotel and beldi-movie here, compare to boki is enough
    AVAILABLE_TEST_CASES = {
        'beldi-hotel-baseline': dict(
            db=dynamodb,
            db_setup_f=dynamodb_setup_hotel_f,
            baseline=True,
            workflow_bin_dir="/beldi-bin",
        ),
        'beldi-movie-baseline': dict(
            db=dynamodb,
            db_setup_f=dynamodb_setup_media_f,
            baseline=True,
            workflow_bin_dir="/beldi-bin",
        ),
        'boki-hotel-baseline': dict(
            db=dynamodb,
            db_setup_f=dynamodb_setup_hotel_f,
            baseline=False,
            workflow_bin_dir="/bokiflow-bin",
        ),
        'boki-movie-baseline': dict(
            db=dynamodb,
            db_setup_f=dynamodb_setup_hotel_f,
            baseline=False,
            workflow_bin_dir="/bokiflow-bin",
        ),
        'boki-hotel-asynclog': dict(
            db=dynamodb,
            db_setup_f=dynamodb_setup_hotel_f,
            baseline=False,
            workflow_bin_dir="/asynclog-bin",
        ),
        'boki-movie-asynclog': dict(
            db=dynamodb,
            db_setup_f=dynamodb_setup_hotel_f,
            baseline=False,
            workflow_bin_dir="/asynclog-bin",
        ),
        'sharedlog': dict(
            db="",
            db_setup_f="",
            baseline=False,
            workflow_bin_dir="/test-bin",
        ),
    }

    # argument assertations
    if args.test_case not in AVAILABLE_TEST_CASES:
        raise Exception("invalid test case: '{}', need to be one of: {}".format(
                        args.test_case, list(AVAILABLE_TEST_CASES.keys())))
    if args.test_case.startswith('boki-') and args.table_prefix == "":
        raise Exception("table prefix of workflow is not allowed to be empty")

    config = AVAILABLE_TEST_CASES[args.test_case]

    dc_content = generate_docker_compose(
        config, args.workdir, args.table_prefix, args.metalog_reps, args.userlog_reps, args.index_reps)
    print(dc_content)
