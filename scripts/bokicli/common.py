from enum import Enum

class Mode(Enum):
    # local docker-compose cluster for debugging
    LOCAL = 0,
    # remote docker swarm cluster on AWS for benchmarking
    REMOTE = 1,

class WorkflowLibName(Enum):
    beldi = "beldi",
    boki = "boki",
    asynclog = "asynclog",
    test = "test",
    optimal = "optimal",

class WorkflowAppName(Enum):
    singleop = "singleop",
    hotel = "hotel",
    media = "media",
    bhotel = "bhotel",
    bmedia = "bmedia",
    finra = "finra",


WORKFLOW_EXP_DIR = "experiments/workflow"
WORKFLOW_EXP_APP_NAME = "boki-finra-baseline"
WORKFLOW_APP_DIR = "workloads/workflow"

ZOOKEEPER_ENDPOINT = 'zookeeper:2181'
# BIN_ENV = """- LD_PRELOAD=/boki/libprofiler.so
#       - CPUPROFILE=/tmp/gperf/prof.out"""
BIN_ENV = ''

VERBOSE = 0
IO_URING_ENTRIES = 64
IO_URING_FD_SLOTS = 1024
LOCAL_FUNC_ENV = "- DBENV=LOCAL"
TABLE_PREFIX = "23333333-"
# HALFMOON_LOGGING_MODE = "read"
HALFMOON_LOGGING_MODE = "write"

IMAGE_TESTS = "adjwang/boki-tests:dev"

IMAGE_FAAS = "adjwang/boki:dev"
IMAGE_APP = "adjwang/boki-beldibench:dev"
# IMAGE_FAAS = "zjia/boki:sosp-ae"
# IMAGE_APP = "zjia/boki-beldibench:sosp-ae"
