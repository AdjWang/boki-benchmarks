#!/bin/bash

# build builder container
# boki/dockerfiles/build_image.sh build

# check submodules
git fetch --recurse-submodules -j16

# build boki dependencies
docker run --rm -v $HOME/dev/boki-benchmarks/boki:/boki adjwang/boki-buildenv:dev bash -c "cd /boki && ./build_deps.sh"
