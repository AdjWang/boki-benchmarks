#!/bin/bash
set -uxo pipefail

ssh adjwang@10.0.9.63 -- docker swarm leave --force
ssh adjwang@10.0.9.64 -- docker swarm leave --force
ssh adjwang@10.0.9.65 -- docker swarm leave --force
ssh adjwang@10.0.9.66 -- docker swarm leave --force
ssh adjwang@10.0.9.69 -- docker swarm leave --force
