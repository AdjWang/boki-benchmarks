#!/bin/bash
set -uxo pipefail

ROOT_DIR=`realpath $(dirname $0)/..`

# Microbenchmarks
RUN_MICROBENCH=y

# Message queue workload for BokiQueue and Pulsar
RUN_QUEUE_BOKI=y
RUN_QUEUE_PUSLAR=
RUN_QUEUE_SQS=

# Retwis workload for BokiStore and MongoDB
RUN_STORE_BOKI=
RUN_STORE_MONGO=

# Workflow workload for BokiFlow and Beldi
RUN_WORKFLOW_BOKI=y
RUN_WORKFLOW_BELDI=

HELPER_SCRIPT=$ROOT_DIR/scripts/exp_helper

# This IAM role has DynamoDB read/write access
BOKI_MACHINE_IAM=boki-ae-experiments

if [[ ! -z $RUN_MICROBENCH ]] && [[ $RUN_MICROBENCH == "y" ]]; then
echo "====== Start running Microbench experiments ======"

BASE_DIR=$ROOT_DIR/experiments/microbenchmark

$HELPER_SCRIPT start-machines --base-dir=$BASE_DIR --instance-iam-role $BOKI_MACHINE_IAM

$BASE_DIR/run_once.sh i4c100b1 4 100 1
$BASE_DIR/run_once.sh i4c100b10 4 100 10

# $HELPER_SCRIPT stop-machines --base-dir=$BASE_DIR
echo "[DEBUG] early exit"
exit 0

echo "====== Finish running Microbench experiments ======"
else
echo "====== Skip Microbench experiments ======"
fi
echo ""


if [[ ! -z $RUN_QUEUE_BOKI ]] && [[ $RUN_QUEUE_BOKI == "y" ]]; then
echo "====== Start running BokiQueue experiments ======"

BASE_DIR=$ROOT_DIR/experiments/queue/boki

$HELPER_SCRIPT start-machines --base-dir=$BASE_DIR --instance-iam-role $BOKI_MACHINE_IAM

$BASE_DIR/run_once.sh p128c128b1 128 6 1 128 1
$BASE_DIR/run_once.sh p128c128b10 128 6 1 128 10
# $BASE_DIR/run_once.sh p128c32  32  8 1 128
# $BASE_DIR/run_once.sh p32c128  128 3 1 32

# $HELPER_SCRIPT stop-machines --base-dir=$BASE_DIR
echo "[DEBUG] early exit"
exit 0

echo "====== Finish running BokiQueue experiments ======"
else
echo "====== Skip BokiQueue experiments ======"
fi
echo ""


if [[ ! -z $RUN_QUEUE_PUSLAR ]] && [[ $RUN_QUEUE_PUSLAR == "y" ]]; then
echo "====== Start running Pulsar experiments ======"

BASE_DIR=$ROOT_DIR/experiments/queue/pulsar

$HELPER_SCRIPT start-machines --base-dir=$BASE_DIR --instance-iam-role $BOKI_MACHINE_IAM

$BASE_DIR/run_once.sh p128c128 6 128 128
$BASE_DIR/run_once.sh p128c32  8 128 32
$BASE_DIR/run_once.sh p32c128  3 32  128

$HELPER_SCRIPT stop-machines --base-dir=$BASE_DIR

echo "====== Finish running Pulsar experiments ======"
else
echo "====== Skip Pulsar experiments ======"
fi
echo ""


if [[ ! -z $RUN_QUEUE_SQS ]] && [[ $RUN_QUEUE_SQS == "y" ]]; then
echo "====== Start running SQS experiments ======"

BASE_DIR=$ROOT_DIR/experiments/queue/sqs

$HELPER_SCRIPT start-machines --base-dir=$BASE_DIR --instance-iam-role $BOKI_MACHINE_IAM

$BASE_DIR/run_once.sh p128c128 10 128 128
$BASE_DIR/run_once.sh p128c32  24 128 32
$BASE_DIR/run_once.sh p32c128  7  32  128

$HELPER_SCRIPT stop-machines --base-dir=$BASE_DIR

echo "====== Finish running SQS experiments ======"
else
echo "====== Skip SQS experiments ======"
fi
echo ""


if [[ ! -z $RUN_STORE_BOKI ]] && [[ $RUN_STORE_BOKI == "y" ]]; then
echo "====== Start running BokiStore experiments ======"

BASE_DIR=$ROOT_DIR/experiments/retwis/boki

$HELPER_SCRIPT start-machines --base-dir=$BASE_DIR --instance-iam-role $BOKI_MACHINE_IAM

$BASE_DIR/run_once.sh con128 128
$BASE_DIR/run_once.sh con192 192

$HELPER_SCRIPT stop-machines --base-dir=$BASE_DIR

echo "====== Finish running BokiStore experiments ======"
else
echo "====== Skip BokiStore experiments ======"
fi
echo ""


if [[ ! -z $RUN_STORE_MONGO ]] && [[ $RUN_STORE_MONGO == "y" ]]; then
echo "====== Start running MongoDB experiments ======"

BASE_DIR=$ROOT_DIR/experiments/retwis/mongodb

$HELPER_SCRIPT start-machines --base-dir=$BASE_DIR --instance-iam-role $BOKI_MACHINE_IAM

$BASE_DIR/run_once.sh con128 128
$BASE_DIR/run_once.sh con192 192

$HELPER_SCRIPT stop-machines --base-dir=$BASE_DIR

echo "====== Finish running MongoDB experiments ======"
else
echo "====== Skip MongoDB experiments ======"
fi
echo ""


if [[ ! -z $RUN_WORKFLOW_BOKI ]] && [[ $RUN_WORKFLOW_BOKI == "y" ]]; then
echo "====== Start running BokiFlow experiments ======"

BASE_DIR=$ROOT_DIR/experiments/workflow/boki-hotel-baseline

$HELPER_SCRIPT start-machines --base-dir=$BASE_DIR --instance-iam-role $BOKI_MACHINE_IAM

$BASE_DIR/run_once.sh qps100 100
# $BASE_DIR/run_once.sh qps200 200

$HELPER_SCRIPT stop-machines --base-dir=$BASE_DIR
echo "[DEBUG] exit early"
exit 0


BASE_DIR=$ROOT_DIR/experiments/workflow/boki-movie-baseline

$HELPER_SCRIPT start-machines --base-dir=$BASE_DIR --instance-iam-role $BOKI_MACHINE_IAM

$BASE_DIR/run_once.sh qps100 100
$BASE_DIR/run_once.sh qps150 150

$HELPER_SCRIPT stop-machines --base-dir=$BASE_DIR


BASE_DIR=$ROOT_DIR/experiments/workflow/boki-hotel-asynclog

$HELPER_SCRIPT start-machines --base-dir=$BASE_DIR --instance-iam-role $BOKI_MACHINE_IAM

$BASE_DIR/run_once.sh qps100 100
$BASE_DIR/run_once.sh qps200 200

$HELPER_SCRIPT stop-machines --base-dir=$BASE_DIR


BASE_DIR=$ROOT_DIR/experiments/workflow/boki-movie-asynclog

$HELPER_SCRIPT start-machines --base-dir=$BASE_DIR --instance-iam-role $BOKI_MACHINE_IAM

$BASE_DIR/run_once.sh qps100 100
$BASE_DIR/run_once.sh qps150 150

$HELPER_SCRIPT stop-machines --base-dir=$BASE_DIR

echo "====== Finish running BokiFlow experiments ======"
else
echo "====== Skip BokiFlow experiments ======"
fi
echo ""


if [[ ! -z $RUN_WORKFLOW_BELDI ]] && [[ $RUN_WORKFLOW_BELDI == "y" ]]; then
echo "====== Start running Beldi experiments ======"

BASE_DIR=$ROOT_DIR/experiments/workflow/beldi-hotel-baseline

$HELPER_SCRIPT start-machines --base-dir=$BASE_DIR --instance-iam-role $BOKI_MACHINE_IAM

$BASE_DIR/run_once.sh qps100 100
$BASE_DIR/run_once.sh qps200 200

$HELPER_SCRIPT stop-machines --base-dir=$BASE_DIR

BASE_DIR=$ROOT_DIR/experiments/workflow/beldi-movie-baseline

$HELPER_SCRIPT start-machines --base-dir=$BASE_DIR --instance-iam-role $BOKI_MACHINE_IAM

$BASE_DIR/run_once.sh qps100 100
$BASE_DIR/run_once.sh qps150 150

$HELPER_SCRIPT stop-machines --base-dir=$BASE_DIR

echo "====== Finish running Beldi experiments ======"
else
echo "====== Skip Beldi experiments ======"
fi
echo ""
