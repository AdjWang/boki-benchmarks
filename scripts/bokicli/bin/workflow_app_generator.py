""" Generate a workflow application.
Files:
- App at ${WORKFLOW_APP_DIR}/
    - build_all.sh
- App at ${WORKFLOW_APP_DIR}/${WORKFLOW_LIB_NAME}/
    - benchmark/${WORKFLOW_APP_NAME}/workload.lua
    - internal/${WORKFLOW_APP_NAME}/[functions...]
    - Makefile
- Dockerfiles at dockerfiles/
"""

import os
import sys
from pathlib import Path
PROJECT_DIR = Path(sys.argv[0]).parent.parent
sys.path.append(str(PROJECT_DIR))

BOKI_BENCH_DIR = Path(sys.argv[0]).parent.parent.parent.parent
print(BOKI_BENCH_DIR)

import common
import templates.workflow_function as faasfunc


def generate_makefile_target(name, funcs):
    target = [f'{name}: {" ".join(funcs)}']
    for func_name in funcs:
        target.append(f'{func_name}:')
        target.append(f'\tenv GOOS=linux go build -ldflags="-s -w -X github.com/eniac/Beldi/pkg/beldilib.TYPE=BELDI" -o bin/{name}/{func_name} internal/{name}/cmd/{func_name}/main.go')
    return '\n'.join(target)


if __name__ == '__main__':
    mode = common.Mode.LOCAL
    WORKFLOW_LIB_NAME = common.WorkflowLibName.boki.value[0]
    WORKFLOW_APP_NAME = common.WorkflowAppName.finra.value[0]
    functions = [
        'fetchData',
        'lastpx',
        'marginBalance',
        'marketdata',
        'side',
        'trddate',
        'volume',
    ]

    app_dir = BOKI_BENCH_DIR/common.WORKFLOW_APP_DIR / \
        WORKFLOW_LIB_NAME/'internal'/WORKFLOW_APP_NAME
    func_dirs = [app_dir/f'cmd/{func}' for func in functions]
    dirs = [
        app_dir,
        app_dir/'cmd',
        *func_dirs,
    ]
    # make new directories
    for dir in dirs:
        if not dir.exists():
            print(f'mkdir={dir}')
            os.mkdir(dir)
    # create main.go templates
    for func_dir in func_dirs:
        if not Path(func_dir/'main.go').exists():
            print(f'create main.go at {func_dir}')
            with open(func_dir/'main.go', 'w') as f:
                f.write(faasfunc.temp_main)
    # output make file targets
    print(generate_makefile_target(WORKFLOW_APP_NAME, functions))
