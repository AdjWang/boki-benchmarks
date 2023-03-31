# !/usr/bin/env python3
# -*- coding: utf-8 -*-
import os
import argparse

# dc: docker compose file

dc_header = """\
version: "3.8"
services:
"""

# services

# zookeeper
zookeeper = """\
  zookeeper:
    image: zookeeper:3.6.2
    hostname: zookeeper
    networks:
      - boki-net
    ports:
      - 2181:2181
    # restart: always

"""

# zookeeper intializer
zookeeper_setup_f = """\
  zookeeper-setup:
    image: zookeeper:3.6.2
    command: /tmp/boki/zk_setup.sh
    depends_on:
       - zookeeper
    volumes:
      - {workdir}/config/zk_setup.sh:/tmp/boki/zk_setup.sh
      - {workdir}/config/zk_health_check:/tmp/boki/zk_health_check
    network_mode: "host"
    # restart: always
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

sharedlog_funcs_f = """\
  sharedlog-Foo-{node_id}:
    image: adjwang/boki-tests:dev
    networks:
      - boki-net
    entrypoint: ["/tmp/boki/run_launcher", "/test-bin/sharedlog/nightcore_basic", "1"]
    volumes:
      - {workdir}/mnt/inmem{node_id}/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=2
      - GOGC=200
    depends_on:
      - boki-engine-{node_id}
    # restart: always

  sharedlog-Bar-{node_id}:
    image: adjwang/boki-tests:dev
    networks:
      - boki-net
    entrypoint: ["/tmp/boki/run_launcher", "/test-bin/sharedlog/nightcore_basic", "2"]
    volumes:
      - {workdir}/mnt/inmem{node_id}/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=2
      - GOGC=200
    depends_on:
      - boki-engine-{node_id}
    # restart: always

  sharedlog-BasicLogOp-{node_id}:
    image: adjwang/boki-tests:dev
    networks:
      - boki-net
    entrypoint: ["/tmp/boki/run_launcher", "/test-bin/sharedlog/sharedlog_basic", "3"]
    volumes:
      - {workdir}/mnt/inmem{node_id}/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=2
      - GOGC=200
    depends_on:
      - boki-engine-{node_id}
    # restart: always

  sharedlog-AsyncLogOp-{node_id}:
    image: adjwang/boki-tests:dev
    networks:
      - boki-net
    entrypoint: ["/tmp/boki/run_launcher", "/test-bin/sharedlog/sharedlog_basic", "4"]
    volumes:
      - {workdir}/mnt/inmem{node_id}/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=2
      - GOGC=200
    depends_on:
      - boki-engine-{node_id}
    # restart: always

  sharedlog-Bench-{node_id}:
    image: adjwang/boki-tests:dev
    networks:
      - boki-net
    entrypoint: ["/tmp/boki/run_launcher", "/test-bin/sharedlog/sharedlog_basic", "5"]
    volumes:
      - {workdir}/mnt/inmem{node_id}/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=2
      - GOGC=200
    depends_on:
      - boki-engine-{node_id}
    # restart: always

"""

network_config = """\
networks:
  boki-net:
    driver: bridge

"""

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--metalog-reps', type=int, default=3)
    parser.add_argument('--userlog-reps', type=int, default=3)
    parser.add_argument('--index-reps', type=int, default=3)
    parser.add_argument('--test-case', type=str, required=True)
    parser.add_argument('--workdir', type=str, default='/tmp')
    parser.add_argument('--output', type=str, default='/tmp')
    args = parser.parse_args()

    # global config
    ZOOKEEPER_ENDPOINT = 'zookeeper:2181'
    # BIN_ENV = """- LD_PRELOAD=/boki/libprofiler.so
    #       - CPUPROFILE=/tmp/gperf/prof.out"""
    BIN_ENV = ''

    WORK_DIR = args.workdir
    METALOG_REPLICAS = args.metalog_reps
    USERLOG_REPLICAS = args.userlog_reps
    INDEX_REPLICAS = args.index_reps
    VERBOSE = 1
    IO_URING_ENTRIES = 64
    IO_URING_FD_SLOTS = 128

    AVAILABLE_TEST_CASES = {
        'sharedlog': sharedlog_funcs_f,
    }
    if args.test_case not in AVAILABLE_TEST_CASES:
        raise Exception("invalid test case: '{}', need to be one of: {}".format(
                        args.test_case, list(AVAILABLE_TEST_CASES.keys())))
    app_funcs_f = AVAILABLE_TEST_CASES[args.test_case]

    dc_content = ''.join([
        dc_header,
        zookeeper,
        zookeeper_setup_f.format(workdir=WORK_DIR),

        boki_controller_f.format(
            bin_env='',
            zookeeper_endpoint=ZOOKEEPER_ENDPOINT,
            metalog_reps=METALOG_REPLICAS,
            userlog_reps=USERLOG_REPLICAS,
            index_reps=INDEX_REPLICAS,
            verbose=VERBOSE
        ),
        boki_gateway_f.format(
            workdir=WORK_DIR,
            bin_env='',
            zookeeper_endpoint=ZOOKEEPER_ENDPOINT,
            io_uring_entries=IO_URING_ENTRIES,
            io_uring_fd_slots=IO_URING_FD_SLOTS,
            verbose=VERBOSE
        ),

        *[boki_engine_f.format(
            workdir=WORK_DIR,
            node_id=i,
            bin_env=BIN_ENV,
            zookeeper_endpoint=ZOOKEEPER_ENDPOINT,
            io_uring_entries=IO_URING_ENTRIES,
            io_uring_fd_slots=IO_URING_FD_SLOTS,
            verbose=VERBOSE
        ) for i in range(1, 1+INDEX_REPLICAS)],

        *[boki_storage_f.format(
            workdir=WORK_DIR,
            node_id=i,
            bin_env=BIN_ENV,
            zookeeper_endpoint=ZOOKEEPER_ENDPOINT,
            io_uring_entries=IO_URING_ENTRIES,
            io_uring_fd_slots=IO_URING_FD_SLOTS,
            verbose=VERBOSE
        ) for i in range(1, 1+USERLOG_REPLICAS)],

        *[boki_sequencer_f.format(
            workdir=WORK_DIR,
            node_id=i,
            bin_env=BIN_ENV,
            zookeeper_endpoint=ZOOKEEPER_ENDPOINT,
            io_uring_entries=IO_URING_ENTRIES,
            io_uring_fd_slots=IO_URING_FD_SLOTS,
            verbose=VERBOSE
        ) for i in range(1, 1+METALOG_REPLICAS)],

        *[app_funcs_f.format(
            workdir=WORK_DIR,
            node_id=i
        ) for i in range(1, 1+INDEX_REPLICAS)],
        network_config,
    ])

    with open(os.path.join(args.output, 'docker-compose.yml'), 'w') as f:
        f.write(dc_content)
