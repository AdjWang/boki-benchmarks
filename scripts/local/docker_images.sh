#!/bin/bash
set -euxo pipefail

ROOT_DIR=`realpath $(dirname $0)/../..`
DOCKERFILE_DIR=$ROOT_DIR/scripts/local/dockerfiles

# Use BuildKit as docker builder
# export DOCKER_BUILDKIT=1
DOCKER_BUILDER=$HOME/.docker/cli-plugins/docker-buildx

function build_boki {
    $DOCKER_BUILDER build -t zjia/boki:sosp-ae \
        -f $DOCKERFILE_DIR/Dockerfile.boki \
        $ROOT_DIR/boki
}

function build_queuebench {
    $DOCKER_BUILDER build -t zjia/boki-queuebench:sosp-ae \
        -f $DOCKERFILE_DIR/Dockerfile.queuebench \
        $ROOT_DIR/workloads/queue
}

function build_retwisbench {
    $DOCKER_BUILDER build -t zjia/boki-retwisbench:sosp-ae \
        -f $DOCKERFILE_DIR/Dockerfile.retwisbench \
        $ROOT_DIR/workloads/retwis
}

function build_beldibench {
    $DOCKER_BUILDER build -t zjia/boki-beldibench:sosp-ae \
        -f $DOCKERFILE_DIR/Dockerfile.beldibench \
        $ROOT_DIR/workloads/workflow
}

function build_goexample {
    $DOCKER_BUILDER build -t zjia/boki-goexample:sosp-ae \
        -f $DOCKERFILE_DIR/Dockerfile.goexample \
        $ROOT_DIR/workloads/goexample
}

function build {
    build_boki
    build_queuebench
    build_retwisbench
    build_beldibench
    build_goexample
}

function push {
    docker push zjia/boki:sosp-ae
    docker push zjia/boki-queuebench:sosp-ae
    docker push zjia/boki-retwisbench:sosp-ae
    docker push zjia/boki-beldibench:sosp-ae
}

case "$1" in
build)
    build
    ;;
push)
    push
    ;;
esac
