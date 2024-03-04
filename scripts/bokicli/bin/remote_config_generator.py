# !/usr/bin/env python3
# -*- coding: utf-8 -*-
import os
import sys
from dataclasses import dataclass
from pathlib import Path
PROJECT_DIR = Path(sys.argv[0]).parent.parent
sys.path.append(str(PROJECT_DIR))
import argparse

from templates.docker_func import FuncMeta
from templates.docker_swarm_boki import generate_docker_compose, generate_run_once
import common

@dataclass
class ServConfig:
    data_init_mode: str
    enable_sharedlog: bool
    wrk_env: str
    workflow_bin_dir: str
    workflow_lib_name: str
    workflow_app_name: str
    serv_generator: FuncMeta

REMOTE_SERVICES = {
    'beldi-hotel-baseline': ServConfig(
        data_init_mode="baseline",
        enable_sharedlog=False,
        wrk_env="BASELINE=1",
        workflow_bin_dir="/beldi-bin",
        workflow_lib_name=common.WorkflowLibName.beldi.value[0],
        workflow_app_name=common.WorkflowAppName.hotel.value[0],
        serv_generator=common.WORKFLOW_HOTEL_SERVS,
    ),
    'beldi-movie-baseline': ServConfig(
        data_init_mode="baseline",
        enable_sharedlog=False,
        wrk_env="BASELINE=1",
        workflow_bin_dir="/beldi-bin",
        workflow_lib_name=common.WorkflowLibName.beldi.value[0],
        workflow_app_name=common.WorkflowAppName.media.value[0],
        serv_generator=common.WORKFLOW_MEDIA_SERVS,
    ),
    'boki-hotel-baseline': ServConfig(
        data_init_mode="cayon",
        enable_sharedlog=True,
        wrk_env="",
        workflow_bin_dir="/bokiflow-bin",
        workflow_lib_name=common.WorkflowLibName.boki.value[0],
        workflow_app_name=common.WorkflowAppName.hotel.value[0],
        serv_generator=common.WORKFLOW_HOTEL_SERVS,
    ),
    'boki-movie-baseline': ServConfig(
        data_init_mode="cayon",
        enable_sharedlog=True,
        wrk_env="",
        workflow_bin_dir="/bokiflow-bin",
        workflow_lib_name=common.WorkflowLibName.boki.value[0],
        workflow_app_name=common.WorkflowAppName.media.value[0],
        serv_generator=common.WORKFLOW_MEDIA_SERVS,
    ),
    'boki-finra-baseline': ServConfig(
        data_init_mode="",
        enable_sharedlog=True,
        wrk_env="",
        workflow_bin_dir="/bokiflow-bin",
        workflow_lib_name=common.WorkflowLibName.boki.value[0],
        workflow_app_name=common.WorkflowAppName.finra.value[0],
        serv_generator=common.WORKFLOW_FINRA_SERVS,
    ),
    'boki-hotel-asynclog': ServConfig(
        data_init_mode="cayon",
        enable_sharedlog=True,
        wrk_env="",
        workflow_bin_dir="/asynclog-bin",
        workflow_lib_name=common.WorkflowLibName.asynclog.value[0],
        workflow_app_name=common.WorkflowAppName.hotel.value[0],
        serv_generator=common.WORKFLOW_HOTEL_SERVS,
    ),
    'boki-movie-asynclog': ServConfig(
        data_init_mode="cayon",
        enable_sharedlog=True,
        wrk_env="",
        workflow_bin_dir="/asynclog-bin",
        workflow_lib_name=common.WorkflowLibName.asynclog.value[0],
        workflow_app_name=common.WorkflowAppName.media.value[0],
        serv_generator=common.WORKFLOW_MEDIA_SERVS,
    ),
    'boki-finra-asynclog': ServConfig(
        data_init_mode="",
        enable_sharedlog=True,
        wrk_env="",
        workflow_bin_dir="/asynclog-bin",
        workflow_lib_name=common.WorkflowLibName.boki.value[0],
        workflow_app_name=common.WorkflowAppName.finra.value[0],
        serv_generator=common.WORKFLOW_FINRA_SERVS,
    ),
}

def dump_configs(dump_dir, config: ServConfig, args):
    docker_compose_boki = generate_docker_compose(config.enable_sharedlog,
                                                  args.metalog_reps,
                                                  args.userlog_reps,
                                                  args.index_reps)
    app_prefix = "b" if config.data_init_mode == "baseline" else ""
    docker_compose_func = config.serv_generator.generate_remote_config(config.workflow_bin_dir,
                                                                       app_prefix)
    docker_compose = docker_compose_boki + docker_compose_func

    config_json = config.serv_generator.generate_cluster_config(
        args.metalog_reps, args.userlog_reps, args.index_reps)
    nightcore_config_json = config.serv_generator.generate_nightcore_config(app_prefix)

    bin_path = Path(config.workflow_bin_dir) / (app_prefix + config.workflow_app_name)
    run_once_sh = generate_run_once(
        bin_path, config.data_init_mode, config.workflow_lib_name, config.workflow_app_name,
        ' '.join([args.wrk_env, config.wrk_env]))

    if not Path(dump_dir).exists():
        os.mkdir(dump_dir)
    with open(dump_dir / "docker-compose.yml", "w") as f:
        f.write(docker_compose)
    with open(dump_dir / "config.json", "w") as f:
        f.write(config_json)
    with open(dump_dir / "nightcore_config.json", "w") as f:
        f.write(nightcore_config_json)
    with open(dump_dir / "run_once.sh", "w") as f:
        f.write(run_once_sh)
    os.chmod(dump_dir / "run_once.sh", 0o777)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--workflow-dir', type=str, required=True,
                        help="usually: boki-benchmarks/experiments/workflow")
    parser.add_argument('--metalog-reps', type=int, default=3)
    parser.add_argument('--userlog-reps', type=int, default=3)
    parser.add_argument('--index-reps', type=int, default=8)
    parser.add_argument('--wrk-env', type=str, default="")
    args = parser.parse_args()

    workflow_dir = Path(args.workflow_dir)
    benchmarks = [
        # "beldi-hotel",
        # "beldi-movie",
        # "beldi-hotel-baseline",
        # "beldi-movie-baseline",
        "boki-hotel-baseline",
        # "boki-movie-baseline",
        # "boki-finra-baseline",
        "boki-hotel-asynclog",
        # "boki-movie-asynclog",
        # "boki-finra-asynclog",
    ]
    for bench_name in benchmarks:
        assert bench_name in REMOTE_SERVICES, f'{bench_name}'
        config = REMOTE_SERVICES[bench_name]
        dump_configs(workflow_dir / bench_name, config, args)
