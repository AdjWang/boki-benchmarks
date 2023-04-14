#!/bin/bash
set -euxo pipefail

ROOT_DIR=`realpath $(dirname $0)/../..`
DOCKERFILE_DIR=$ROOT_DIR/scripts/local/dockerfiles

# Use BuildKit as docker builder
# export DOCKER_BUILDKIT=1
DOCKER_BUILDER=$HOME/.docker/cli-plugins/docker-buildx

# NO_CACHE="--no-cache"
NO_CACHE=""

function build_boki {
    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki:dev \
        -f $DOCKERFILE_DIR/Dockerfile.boki \
        $ROOT_DIR/boki
}

function build_queuebench {
    $DOCKER_BUILDER build $NO_CACHE -t adjwang/boki-queuebench:dev \
        -f $DOCKERFILE_DIR/Dockerfile.queuebench \
        $ROOT_DIR/workloads/queue
}

function build_retwisbench {
    $DOCKER_BUILDER build -t adjwang/boki-retwisbench:dev \
        -f $DOCKERFILE_DIR/Dockerfile.retwisbench \
        $ROOT_DIR/workloads/retwis
}

function build_beldibench {
    $DOCKER_BUILDER build -t adjwang/boki-beldibench:dev \
        -f $DOCKERFILE_DIR/Dockerfile.beldibench \
        $ROOT_DIR/workloads/workflow
}

function build_goexample {
    $DOCKER_BUILDER build -t adjwang/boki-goexample:dev \
        -f $DOCKERFILE_DIR/Dockerfile.goexample \
        $ROOT_DIR/workloads/goexample
}

function build {
    build_boki
    build_queuebench
    build_retwisbench
    build_beldibench
    # build_goexample
}

function push {
    docker push adjwang/boki:dev
    docker push adjwang/boki-queuebench:dev
    docker push adjwang/boki-retwisbench:dev
    docker push adjwang/boki-beldibench:dev
}

case "$1" in
build)
    build
    ;;
push)
    push
    ;;
esac
