from typing import List, Dict, Callable
from dataclasses import dataclass, field
import json

from common import WorkflowLibName

@dataclass
class FuncMeta:
    app_name: str
    image_name: str
    engine_mnt_dir_local: str
    func_names: List[str]
    worker_min_max: Dict[str, tuple]
    func_envs_local_getter: Callable[[WorkflowLibName], Dict[str, str]]
    func_envs_remote_getter: Callable[[WorkflowLibName], Dict[str, str]]

    service_names: List[str] = field(init=False)
    func_bins: List[str] = field(default=None)
    workflow_lib_name: WorkflowLibName = field(default=None)
    # declaring common indents already have, not which should add to all lines
    # 2 spaces per indent
    base_indents: int = 1
    func_template_local = """\
  {service_name}-{node_id}:
    image: {image_name}
    networks:
      - boki-net
    entrypoint: ["/tmp/boki/run_launcher", "{func_bin}", "{func_id}"]
    volumes:
      - {mnt_dir}/mnt/inmem{node_id}/boki:/tmp/boki
    environment:
      {func_envs}
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
    depends_on:
      - boki-engine-{node_id}
    # restart: always
"""
    func_template_remote = """\
  {service_name}:
    image: {image_name}
    entrypoint: ["/tmp/boki/run_launcher", "{func_bin}", "{func_id}"]
    volumes:
      - /mnt/inmem/boki:/tmp/boki
    environment:
      {func_envs}
      - FAAS_GO_MAX_PROC_FACTOR=8
      - GOGC=1000
    depends_on:
      - boki-engine
    restart: always
"""

    def __post_init__(self):
        self.service_names = [f"{fn_name}-service" for fn_name in self.func_names]

    @property
    def services_count(self):
        return len(self.service_names)
    
    def set_workflow_lib_name(self, name: WorkflowLibName):
        self.workflow_lib_name = name

    def generate_local_config(
        self,
        image_fn_bin_dir: str,
        engines: int,
        app_prefix: str = "",
    ) -> str:
        if self.func_bins is None:
            func_bins = [f'{image_fn_bin_dir}/{app_prefix}{self.app_name}/{fn_name}'
                         for fn_name in self.func_names]
        else:
            assert len(self.func_bins) == len(self.func_names)
            func_bins = [f'{image_fn_bin_dir}/{app_prefix}{self.app_name}/{fn_name}'
                         for fn_name in self.func_bins]
        assert self.workflow_lib_name is not None
        func_envs = ('  '*(self.base_indents+2)) \
                   .join([f'- {k}={v}\n' for k, v in self.func_envs_local_getter(self.workflow_lib_name).items()]) \
                   .strip('\n')
        func_templates = []
        for engine_id in range(1, engines+1):
            for idx in range(self.services_count):
                func_id = idx + 1
                func = self.func_template_local.format(mnt_dir=self.engine_mnt_dir_local,
                                                       node_id=engine_id,
                                                       service_name=self.service_names[idx],
                                                       image_name=self.image_name,
                                                       func_bin=func_bins[idx],
                                                       func_id=func_id,
                                                       func_envs=func_envs)
                func_templates.append(func)
        return '\n'.join(func_templates)

    def generate_remote_config(self, image_fn_bin_dir: str, app_prefix: str="") -> str:
        func_bins = [f'{image_fn_bin_dir}/{app_prefix}{self.app_name}/{fn_name}'
                     for fn_name in self.func_names]
        assert self.workflow_lib_name is not None
        func_envs = ('  '*(self.base_indents+2)) \
                   .join([f'- {k}={v}\n' for k, v in self.func_envs_remote_getter(self.workflow_lib_name).items()]) \
                   .strip('\n')
        func_templates = []
        for idx in range(self.services_count):
            func_id = idx + 1
            func = self.func_template_remote.format(service_name=self.service_names[idx],
                                                    image_name=self.image_name,
                                                    func_bin=func_bins[idx],
                                                    func_id=func_id,
                                                    func_envs=func_envs)
            func_templates.append(func)
        return '\n'.join(func_templates)

    def generate_nightcore_config(self, func_name_prefix: str="") -> str:
        config = []
        for idx in range(self.services_count):
            func_name = func_name_prefix + self.func_names[idx]
            func_id = idx + 1
            entry = dict(funcName=func_name,
                         funcId=func_id,
                         minWorkers=self.worker_min_max[func_name][0],
                         maxWorkers=self.worker_min_max[func_name][1])
            config.append(entry)
        return json.dumps(config, indent=4)

    def generate_cluster_config(self, rep_sequencer, rep_storage, rep_engine) -> str:
        sequencers = dict()
        for idx in range(1, rep_sequencer+1):
            sequencers[f"bokiexp-sequencer-{idx}"] = \
                { "type": "c5d.2xlarge", "role": "worker", "labels": ["sequencer_node=true"] }
        storages = dict()
        for idx in range(1, rep_storage+1):
            storages[f"bokiexp-storage-{idx}"] = \
                { "type": "c5d.2xlarge", "role": "worker", "mount_instance_storage": "nvme1n1", "labels": ["storage_node=true"] }
        engines = dict()
        for idx in range(1, rep_engine+1):
            engines[f"bokiexp-engine-{idx}"] = \
                { "type": "c5d.2xlarge", "role": "worker", "labels": [ "engine_node=true" ] }
        func_servs = dict()
        for idx, func_serv_name in enumerate(self.service_names, 1):
            func_servs[func_serv_name] = \
                { "placement_label": "engine_node", "replicas": rep_engine, "need_aws_env": True, "mount_certs": True }
        config = dict(
            machines={
                "bokiexp-gateway": {"type": "c5d.2xlarge", "role": "manager"},
                "bokiexp-client": {"type": "c5d.xlarge", "role": "client"},
                **sequencers,
                **storages,
                **engines,
            },
            services={
                "zookeeper": { "placement": "bokiexp-gateway" },
                "zookeeper-setup": { "placement": "bokiexp-gateway" },
                "boki-controller": { "placement": "bokiexp-gateway" },
                "boki-gateway": { "placement": "bokiexp-gateway" },
                "boki-storage": { "placement_label": "storage_node", "replicas": rep_storage },
                "boki-sequencer": { "placement_label": "sequencer_node", "replicas": rep_sequencer },
                "boki-engine": { "placement_label": "engine_node", "replicas": rep_engine },
                **func_servs,
            },
            aws_region="us-east-2",
        )
        return json.dumps(config, indent=4)

if __name__ == '__main__':
    def __get_boki_workflow_func_envs_local(workflow_lib_name: WorkflowLibName):
        if workflow_lib_name == WorkflowLibName.optimal:
            return dict(TABLE_PREFIX="2333",
                        LoggingMode="read")
        else:
            return dict(TABLE_PREFIX="2333")

    def __get_boki_workflow_func_envs_remote(workflow_lib_name: WorkflowLibName):
        if workflow_lib_name == WorkflowLibName.optimal:
            return dict(TABLE_PREFIX="${TABLE_PREFIX:?}",
                        LoggingMode="${LoggingMode:?}")
        else:
            return dict(TABLE_PREFIX="${TABLE_PREFIX:?}")

    bokiflow_hotel_funcs = FuncMeta(
        app_name="hotel",
        image_name="adjwang/boki-beldibench:dev",
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
        func_envs_local_getter=__get_boki_workflow_func_envs_local,
        func_envs_remote_getter=__get_boki_workflow_func_envs_remote,
    )
    bokiflow_hotel_funcs.set_workflow_lib_name(WorkflowLibName.optimal)

    # funcs_per_engine = bokiflow_hotel_funcs.generate_local_config(
    #     image_fn_bin_dir="/optimal-bin", engines=1
    # )
    funcs_per_engine = bokiflow_hotel_funcs.generate_remote_config(
        image_fn_bin_dir="/optimal-bin"
    )
    print(funcs_per_engine)

    # nightcore_config = bokiflow_hotel_funcs.generate_nightcore_config()
    # print(nightcore_config)

    # cluster_config = bokiflow_hotel_funcs.generate_cluster_config(rep_sequencer=3, rep_storage=3, rep_engine=8)
    # print(cluster_config)
