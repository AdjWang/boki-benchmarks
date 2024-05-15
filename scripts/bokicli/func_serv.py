from common import WorkflowLibName, TABLE_PREFIX, IMAGE_TESTS, IMAGE_APP, HALFMOON_LOGGING_MODE
from templates.docker_func import FuncMeta

def __get_finra_func_envs_local(workflow_lib_name: WorkflowLibName):
    return dict(TABLE_PREFIX=TABLE_PREFIX,
                DBENV="LOCAL")

def __get_finra_func_envs_remote(workflow_lib_name: WorkflowLibName):
    return dict(TABLE_PREFIX="${TABLE_PREFIX:?}")


def __get_workflow_func_envs_local(workflow_lib_name: WorkflowLibName):
    if workflow_lib_name == WorkflowLibName.optimal:
        return dict(TABLE_PREFIX=TABLE_PREFIX,
                    DBENV="LOCAL",
                    LoggingMode=HALFMOON_LOGGING_MODE)
    else:
        return dict(TABLE_PREFIX=TABLE_PREFIX,
                    DBENV="LOCAL")

def __get_workflow_func_envs_remote(workflow_lib_name: WorkflowLibName):
    if workflow_lib_name == WorkflowLibName.optimal:
        return dict(TABLE_PREFIX="${TABLE_PREFIX:?}",
                    LoggingMode="${LoggingMode:?}")
    else:
        return dict(TABLE_PREFIX="${TABLE_PREFIX:?}")


SLOG_TEST_SERVS = FuncMeta(
    app_name="sharedlog",
    image_name=IMAGE_TESTS,
    engine_mnt_dir_local="/tmp/boki-test",
    func_names=["Foo",
                "Bar",
                "BasicLogOp",
                "AsyncLogOp",
                "AsyncLogOpChild",
                "Bench"],
    worker_min_max={"Foo": (1, 32),
                    "Bar": (1, 32),
                    "BasicLogOp": (1, 32),
                    "AsyncLogOp": (1, 32),
                    "AsyncLogOpChild": (1, 32),
                    "Bench": (1, 32)},
    func_bins=["sharedlog_basic",
               "sharedlog_basic",
               "sharedlog_basic",
               "sharedlog_basic",
               "sharedlog_basic",
               "sharedlog_basic"],
    func_envs_local_getter=__get_workflow_func_envs_local,
    func_envs_remote_getter=__get_workflow_func_envs_remote,
)

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
    func_envs_local_getter=__get_workflow_func_envs_local,
    func_envs_remote_getter=__get_workflow_func_envs_remote,
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
    func_envs_local_getter=__get_workflow_func_envs_local,
    func_envs_remote_getter=__get_workflow_func_envs_remote,
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
    func_envs_local_getter=__get_finra_func_envs_local,
    func_envs_remote_getter=__get_finra_func_envs_remote,
)

WORKFLOW_OPTIMAL_SINGLEOP_SERVS = FuncMeta(
    app_name="singleop",
    image_name=IMAGE_APP,
    engine_mnt_dir_local="/tmp/boki-test",
    func_names=["nop",
                "singleop",
                "prewarm"],
    worker_min_max={"nop": (8, 8),
                    "singleop": (8, 8),
                    "prewarm": (1, 1)},
    func_envs_local_getter=__get_workflow_func_envs_local,
    func_envs_remote_getter=__get_workflow_func_envs_remote,
)