#!/bin/bash
set -euxo pipefail

docker stop $(docker ps | grep zookeeper | awk '{print $1}')
# docker rm $(docker ps -a | grep zookeeper | awk '{print $1}')
