#!/bin/bash

wget https://download.docker.com/linux/static/stable/x86_64/docker-20.10.24.tgz -O /tmp/docker-20.10.24.tgz
cd /tmp && tar -zxf docker-20.10.24.tgz && cd -

# needs sudo
service docker stop
rm /usr/bin/docker
rm /usr/bin/dockerd
cp /tmp/docker/docker /usr/bin/docker
cp /tmp/docker/dockerd /usr/bin/dockerd
service docker start
docker --version
