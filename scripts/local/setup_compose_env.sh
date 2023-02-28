#!/bin/bash
set -uxo pipefail

INDEX_REPLICATION=3
USERLOG_REPLICATION=3
METALOG_REPLICATION=3

ROOT_DIR=`realpath $(dirname $0)/../..`

# remove old files and folders
rm -rf /tmp/boki
rm -rf /tmp/zk_setup.sh
rm -rf /tmp/zk_health_check
rm -rf /tmp/nightcore_config.json
rm -rf /tmp/run_launcher

ln -s $ROOT_DIR/boki /tmp
cp $ROOT_DIR/scripts/local/zk_setup.sh /tmp
cp $ROOT_DIR/scripts/local/zk_health_check/zk_health_check /tmp
cp $ROOT_DIR/experiments/queue/boki/nightcore_config.json /tmp
cp $ROOT_DIR/experiments/queue/boki/run_launcher /tmp

# engine nodes
for node_i in `seq 1 $INDEX_REPLICATION`; do
    rm -rf /mnt/inmem$node_i
    mkdir /mnt/inmem$node_i
    mkdir /mnt/inmem$node_i/boki
    mkdir /mnt/inmem$node_i/gperf

    cp /tmp/nightcore_config.json /mnt/inmem$node_i/boki/func_config.json
    cp /tmp/run_launcher /mnt/inmem$node_i/boki/run_launcher

    rm -rf /mnt/inmem$node_i/boki/output
    mkdir /mnt/inmem$node_i/boki/output

    rm -rf /mnt/inmem$node_i/boki/ipc
    mkdir /mnt/inmem$node_i/boki/ipc
done

# storage nodes
for node_i in `seq 1 $USERLOG_REPLICATION`; do
    # delete old RocksDB datas
    rm -rf /mnt/storage$node_i/logdata
    rm -rf /mnt/storage$node_i/gperf
    mkdir /mnt/storage$node_i/gperf
done

# sequencer nodes
for node_i in `seq 1 $METALOG_REPLICATION`; do
    rm -rf /mnt/sequencer$node_i
    mkdir /mnt/sequencer$node_i
    mkdir /mnt/sequencer$node_i/gperf
done
