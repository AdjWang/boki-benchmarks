from enum import Enum
from templates.docker_func import FuncMeta

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

IMAGE_FAAS = "adjwang/boki:dev"
IMAGE_APP = "adjwang/boki-beldibench:dev"
# IMAGE_FAAS = "zjia/boki:sosp-ae"
# IMAGE_APP = "zjia/boki-beldibench:sosp-ae"

WORKFLOW_HOTEL_SERVS = FuncMeta(
    app_name="hotel",
    image_name=IMAGE_APP,
    engine_mnt_dir_local="/tmp/boki-test",
    func_names=["geo",
                "profile",
                "rate",
                "recommendation",
                "user",
                "hotel",
                "search",
                "flight",
                "order",
                "frontend",
                "gateway"],
    worker_min_max={"geo": (8, 8),
                    "profile": (8, 8),
                    "rate": (8, 8),
                    "recommendation": (8, 8),
                    "user": (8, 8),
                    "hotel": (8, 8),
                    "search": (8, 8),
                    "flight": (8, 8),
                    "order": (8, 8),
                    "frontend": (8, 8),
                    "gateway": (8, 8)},
    func_envs_local=dict(TABLE_PREFIX=TABLE_PREFIX,
                         DBENV="LOCAL"),
    func_envs_remote=dict(TABLE_PREFIX="${TABLE_PREFIX:?}"),
)

WORKFLOW_MEDIA_SERVS = FuncMeta(
    app_name="media",
    image_name=IMAGE_APP,
    engine_mnt_dir_local="/tmp/boki-test",
    func_names=["Frontend",
                "CastInfo",
                "ReviewStorage",
                "UserReview",
                "MovieReview",
                "ComposeReview",
                "Text",
                "User",
                "UniqueId",
                "Rating",
                "MovieId",
                "Plot",
                "MovieInfo",
                "Page"],
    worker_min_max={"Frontend": (16, 16),
                    "CastInfo": (8, 8),
                    "ReviewStorage": (8, 8),
                    "UserReview": (8, 8),
                    "MovieReview": (8, 8),
                    "ComposeReview": (64, 64),
                    "Text": (8, 8),
                    "User": (8, 8),
                    "UniqueId": (8, 8),
                    "Rating": (8, 8),
                    "MovieId": (8, 8),
                    "Plot": (8, 8),
                    "MovieInfo": (8, 8),
                    "Page": (8, 8)},
    func_envs_local=dict(TABLE_PREFIX=TABLE_PREFIX,
                         DBENV="LOCAL"),
    func_envs_remote=dict(TABLE_PREFIX="${TABLE_PREFIX:?}"),
)

WORKFLOW_FINRA_SERVS = FuncMeta(
    app_name="finra",
    image_name=IMAGE_APP,
    engine_mnt_dir_local="/tmp/boki-test",
    func_names=["fetchData",
                "lastpx",
                "marginBalance",
                "marketdata",
                "side",
                "trddate",
                "volume"],
    worker_min_max={"fetchData": (8, 8),
                    "lastpx": (8, 8),
                    "marginBalance": (8, 8),
                    "marketdata": (8, 8),
                    "side": (8, 8),
                    "trddate": (8, 8),
                    "volume": (8, 8)},
    func_envs_local=dict(TABLE_PREFIX=TABLE_PREFIX,
                         DBENV="LOCAL"),
    func_envs_remote=dict(TABLE_PREFIX="${TABLE_PREFIX:?}"),
)
