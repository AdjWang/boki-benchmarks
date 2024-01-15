""" Generate workflow experiment configurations.
Files:
- Exp at ${WORKFLOW_EXP_DIR}/${EXP_APP_NAME}/
    - config.json
    - docker-compose.yml
    - nightcore_config.json
    - run_launcher
    - run_once.sh
"""

import argparse
import os
import sys
from dataclasses import dataclass
from pathlib import Path
PROJECT_DIR = Path(sys.argv[0]).parent.parent
sys.path.append(str(PROJECT_DIR))

BOKI_BENCH_DIR = Path(sys.argv[0]).parent.parent.parent.parent
print(BOKI_BENCH_DIR)

import common
from templates.docker_func import FuncMeta
from templates.docker_compose_boki import (
    dynamodb,
    dynamodb_setup_hotel_f,
    dynamodb_setup_media_f,
    generate_docker_compose)

@dataclass
class ServConfig:
    db: str
    db_setup_f: str
    unsafe_baseline: bool
    workflow_bin_dir: str
    workflow_lib_name: str
    serv_generator: FuncMeta

LOCAL_SERVICES = {
    'beldi-hotel-baseline': ServConfig(
        db=dynamodb,
        db_setup_f=dynamodb_setup_hotel_f,
        unsafe_baseline=True,
        workflow_bin_dir="/beldi-bin",
        workflow_lib_name=common.WorkflowLibName.beldi.value[0],
        serv_generator=common.WORKFLOW_HOTEL_SERVS,
    ),
    'beldi-movie-baseline': ServConfig(
        db=dynamodb,
        db_setup_f=dynamodb_setup_media_f,
        unsafe_baseline=True,
        workflow_bin_dir="/beldi-bin",
        workflow_lib_name=common.WorkflowLibName.beldi.value[0],
        serv_generator=common.WORKFLOW_MEDIA_SERVS,
    ),
    'boki-hotel-baseline': ServConfig(
        db=dynamodb,
        db_setup_f=dynamodb_setup_hotel_f,
        unsafe_baseline=False,
        workflow_bin_dir="/bokiflow-bin",
        workflow_lib_name=common.WorkflowLibName.boki.value[0],
        serv_generator=common.WORKFLOW_HOTEL_SERVS,
    ),
    'boki-movie-baseline': ServConfig(
        db=dynamodb,
        db_setup_f=dynamodb_setup_media_f,
        unsafe_baseline=False,
        workflow_bin_dir="/bokiflow-bin",
        workflow_lib_name=common.WorkflowLibName.boki.value[0],
        serv_generator=common.WORKFLOW_MEDIA_SERVS,
    ),
    'boki-finra-baseline': ServConfig(
        db="",
        db_setup_f="",
        unsafe_baseline=False,
        workflow_bin_dir="/bokiflow-bin",
        workflow_lib_name=common.WorkflowLibName.boki.value[0],
        serv_generator=common.WORKFLOW_FINRA_SERVS,
    ),
    'boki-hotel-asynclog': ServConfig(
        db=dynamodb,
        db_setup_f=dynamodb_setup_hotel_f,
        unsafe_baseline=False,
        workflow_bin_dir="/asynclog-bin",
        workflow_lib_name=common.WorkflowLibName.boki.value[0],
        serv_generator=common.WORKFLOW_HOTEL_SERVS,
    ),
    'boki-movie-asynclog': ServConfig(
        db=dynamodb,
        db_setup_f=dynamodb_setup_media_f,
        unsafe_baseline=False,
        workflow_bin_dir="/asynclog-bin",
        workflow_lib_name=common.WorkflowLibName.boki.value[0],
        serv_generator=common.WORKFLOW_MEDIA_SERVS,
    ),
    'boki-finra-asynclog': ServConfig(
        db="",
        db_setup_f="",
        unsafe_baseline=False,
        workflow_bin_dir="/asynclog-bin",
        workflow_lib_name=common.WorkflowLibName.boki.value[0],
        serv_generator=common.WORKFLOW_FINRA_SERVS,
    ),
    'sharedlog': ServConfig(
        db="",
        db_setup_f="",
        unsafe_baseline=False,
        workflow_bin_dir="/test-bin",
        workflow_lib_name=common.WorkflowLibName.test.value[0],
        # TODO: add services generator
        serv_generator=None,
    ),
}


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--metalog-reps', type=int, default=3)
    parser.add_argument('--userlog-reps', type=int, default=3)
    parser.add_argument('--index-reps', type=int, default=3)
    parser.add_argument('--test-case', type=str, required=True)
    parser.add_argument('--workdir', type=str, default='/tmp')
    parser.add_argument('--output', type=str, default='/tmp')
    args = parser.parse_args()

    # argument assertations
    if args.test_case not in LOCAL_SERVICES:
        raise Exception("invalid test case: '{}', need to be one of: {}".format(
                        args.test_case, list(LOCAL_SERVICES.keys())))
    if args.test_case.startswith('boki-') and common.TABLE_PREFIX == "":
        raise Exception("table prefix of workflow is not allowed to be empty")

    config = LOCAL_SERVICES[args.test_case]
    baseline_prefix = 'b' if config.unsafe_baseline else ''

    dc_boki = generate_docker_compose(
        config, args.workdir, args.metalog_reps, args.userlog_reps, args.index_reps)
    # print(dc_boki)
    # in all our benchmarks, each engine contains an index
    dc_serv = config.serv_generator.generate_local_config(
        image_fn_bin_dir=config.workflow_bin_dir, engines=args.index_reps)
    # print(dc_serv)

    with open(os.path.join(args.output, 'docker-compose.yml'), 'w') as f:
        f.write(dc_boki+dc_serv)
