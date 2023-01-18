#!/bin/bash
set -uxo pipefail

ROOT_DIR=`realpath $(dirname $0)/..`

ln -s $ROOT_DIR/boki /tmp
cp $ROOT_DIR/scripts/zk_setup.sh /tmp
cp $ROOT_DIR/scripts/zk_health_check/zk_health_check /tmp
cp $ROOT_DIR/experiments/queue/boki/nightcore_config.json /tmp
cp $ROOT_DIR/experiments/queue/boki/run_launcher /tmp

cp /tmp/nightcore_config.json /mnt/inmem/boki/func_config.json
cp /tmp/run_launcher /mnt/inmem/boki/run_launcher

rm -rf /mnt/inmem/boki/output
mkdir /mnt/inmem/boki/output

# delete old RocksDB datas
rm -rf /mnt/storage/logdata
