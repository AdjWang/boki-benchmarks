# !/usr/bin/env python3
# -*- coding: utf-8 -*-

# global config
TRACER_ENDPOINT = 'http://47.96.165.13:8900/api/v2/spans'

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
    ports:
      - 2181:2181
    # restart: always

"""

# zookeeper intializer
zookeeper_setup = """\
  zookeeper-setup:
    image: zookeeper:3.6.2
    command: /tmp/boki/zk_setup.sh
    depends_on:
       - zookeeper
    volumes:
      - /tmp/zk_setup.sh:/tmp/boki/zk_setup.sh
      - /tmp/zk_health_check:/tmp/boki/zk_health_check
    network_mode: "host"
    # restart: always
    healthcheck:
      test: ["CMD-SHELL", "/tmp/boki/zk_health_check"]
      interval: 3s
      retries: 5
      start_period: 5s
      timeout: 10s

"""

boki_engine = """\
  boki-engine-{node_id}:
    image: zjia/boki:sosp-ae
    hostname: faas-engine-{node_id}
    entrypoint:
      - /boki/engine
      - --tracer_exporter_endpoint={tracer_endpoint}
      - --zookeeper_host=zookeeper:2181
      - --listen_iface=eth0
      - --root_path_for_ipc=/tmp/boki/ipc
      - --func_config_file=/tmp/boki/func_config.json
      - --num_io_workers=4
      - --instant_rps_p_norm=0.8
      - --io_uring_entries=64
      - --io_uring_fd_slots=128
      - --enable_shared_log
      - --slog_engine_enable_cache
      - --slog_engine_cache_cap_mb=512
      - --slog_engine_propagate_auxdata
      - --v=1
    depends_on:
      zookeeper-setup:
        condition: service_healthy
    volumes:
      - /mnt/inmem/boki:/tmp/boki
      - /sys/fs/cgroup:/tmp/root_cgroupfs
    environment:
      - FAAS_NODE_ID={node_id}
      - FAAS_CGROUP_FS_ROOT=/tmp/root_cgroupfs
    ulimits:
      memlock: -1
    # restart: always

"""

boki_controller = """\
  boki-controller:
    image: zjia/boki:sosp-ae
    entrypoint:
      - /boki/controller
      - --zookeeper_host=zookeeper:2181
      - --metalog_replicas=3
      - --userlog_replicas=3
      - --index_replicas=1
      - --v=1
    depends_on:
      zookeeper-setup:
        condition: service_healthy
    # restart: always

"""

boki_gateway = """\
  boki-gateway:
    image: zjia/boki:sosp-ae
    hostname: faas-gateway
    ports:
      - 9000:9000
    entrypoint:
      - /boki/gateway
      - --tracer_exporter_endpoint={tracer_endpoint}
      - --zookeeper_host=zookeeper:2181
      - --listen_iface=eth0
      - --http_port=9000
      - --func_config_file=/tmp/boki/func_config.json
      - --num_io_workers=2
      - --io_uring_entries=64
      - --io_uring_fd_slots=128
      - --lb_per_fn_round_robin
      - --max_running_requests=0
      - --v=1
    depends_on:
      zookeeper-setup:
        condition: service_healthy
    volumes:
      - /tmp/nightcore_config.json:/tmp/boki/func_config.json
    ulimits:
      memlock: -1
    # restart: always

"""

boki_storage = """\
  boki-storage-{node_id}:
    image: zjia/boki:sosp-ae
    hostname: faas-storage-{node_id}
    entrypoint:
      - /boki/storage
      - --tracer_exporter_endpoint={tracer_endpoint}
      - --zookeeper_host=zookeeper:2181
      - --listen_iface=eth0
      - --db_path=/tmp/storage/logdata
      - --num_io_workers=2
      - --io_uring_entries=64
      - --io_uring_fd_slots=128
      - --slog_local_cut_interval_us=300
      - --slog_storage_bgthread_interval_ms=1
      - --slog_storage_backend=rocksdb
      - --slog_storage_cache_cap_mb=512
      - --v=1
    depends_on:
      zookeeper-setup:
        condition: service_healthy
    volumes:
      - /mnt/storage1:/tmp/storage
    environment:
      - FAAS_NODE_ID={node_id}
    ulimits:
      memlock: -1
    # restart: always

"""

boki_sequencer = """\
  boki-sequencer-{node_id}:
    image: zjia/boki:sosp-ae
    hostname: faas-sequencer-{node_id}
    entrypoint:
      - /boki/sequencer
      - --tracer_exporter_endpoint={tracer_endpoint}
      - --zookeeper_host=zookeeper:2181
      - --listen_iface=eth0
      - --num_io_workers=2
      - --io_uring_entries=64
      - --io_uring_fd_slots=128
      - --slog_global_cut_interval_us=300
      - --v=1
    depends_on:
      zookeeper-setup:
        condition: service_healthy
    environment:
      - FAAS_NODE_ID={node_id}
    ulimits:
      memlock: -1
    # restart: always

"""

app_funcs = """\
  consumer-fn:
    image: zjia/boki-queuebench:sosp-ae
    entrypoint: ["/tmp/boki/run_launcher", "/queuebench-bin/main", "2"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=2
      - GOGC=200
    depends_on:
      - boki-engine-1
    # restart: always

  producer-fn:
    image: zjia/boki-queuebench:sosp-ae
    entrypoint: ["/tmp/boki/run_launcher", "/queuebench-bin/main", "1"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=2
      - GOGC=200
    depends_on:
      - boki-engine-1
    # restart: always

  goexample-fn:
    image: zjia/boki-goexample:sosp-ae
    entrypoint: ["/tmp/boki/run_launcher", "/goexample-bin/main", "3"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      - FAAS_GO_MAX_PROC_FACTOR=2
      - GOGC=200
    depends_on:
      - boki-engine-1
    # restart: always

"""

if __name__ == '__main__':
    dc_content = (
        dc_header +
        zookeeper + 
        zookeeper_setup +

        boki_controller.format(tracer_endpoint=TRACER_ENDPOINT) + 
        boki_gateway + 

        boki_engine.format(node_id=1, tracer_endpoint=TRACER_ENDPOINT) + 

        boki_storage.format(node_id=1, tracer_endpoint=TRACER_ENDPOINT) + 
        boki_storage.format(node_id=2, tracer_endpoint=TRACER_ENDPOINT) + 
        boki_storage.format(node_id=3, tracer_endpoint=TRACER_ENDPOINT) + 

        boki_sequencer.format(node_id=1, tracer_endpoint=TRACER_ENDPOINT) + 
        boki_sequencer.format(node_id=2, tracer_endpoint=TRACER_ENDPOINT) + 
        boki_sequencer.format(node_id=3, tracer_endpoint=TRACER_ENDPOINT) + 

        app_funcs
    )

    with open('docker-compose.yml', 'w') as f:
        f.write(dc_content)
