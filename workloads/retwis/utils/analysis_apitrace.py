import re
import numpy as np
import matplotlib.pyplot as plt
from collections import defaultdict
from multiprocessing import Pool
from functools import partial
from tqdm import tqdm
from dataclasses import dataclass

PARALLEL = 1

def percentile(datas):
    def __percentile(data_list, p):
        p = np.percentile(data_list, p, axis=0, method='inverted_cdf')
        if isinstance(p, np.ndarray):
            p = p.tolist()
        return p
    p30 = __percentile(datas, 30)
    p50 = __percentile(datas, 50)
    p70 = __percentile(datas, 70)
    p90 = __percentile(datas, 90)
    p99 = __percentile(datas, 99)
    return p30, p50, p70, p90, p99, max(datas), len(datas)


def print_percentile(title, datas):
    assert len(datas) > 0
    p30, p50, p70, p90, p99, p100, count = percentile(datas)
    if not isinstance(p30, np.float64):
        print(f'{title} p30: {p30}; p50: {p50}; p70: {p70}; p90: {p90}; p99: {p99}; p100: {p100}; count: {count}')
    else:
        print(f'{title} p30: {p30:.3f}; p50: {p50:.3f}; p70: {p70:.3f}; p90: {p90:.3f}; p99: {p99:.3f}; p100: {p100:.3f}; count: {count}')


def gather_info(logs, extractor):
    if PARALLEL > 1:
        with Pool(PARALLEL) as pool:
            results = pool.map(extractor, tqdm(logs, "regexp"))
            results = [i for i in tqdm(results, "filter") if i != None]
    else:
        results = [extractor(log) for log in tqdm(logs, "regexp")]
        results = [i for i in tqdm(results, "filter") if i != None]
    return results

@dataclass
class SingleInvocationInfo:
    func_name: str
    latency: int
    n_ops: int
    max_latency: int
    min_latency: int

    def __post_init__(self):
        self.latency = int(self.latency)
        self.n_ops = int(self.n_ops)
        self.max_latency = int(self.max_latency)
        self.min_latency = int(self.min_latency)


FN_SET = ('Post', 'PostList', 'Login', 'Profile')
LOG_FN_SET = ('Append', 'ReadPrev', 'ReadNext', 'ReadNextB', 'SetAuxData')

class InvocationInfo:
    def __init__(self, entries):
        self.sub_invocations = dict()
        for entry in entries:
            info = SingleInvocationInfo(*entry)
            if info.func_name in FN_SET:
                self.func_name = info.func_name
                self.latency = info.latency
                self.n_ops = info.n_ops
                self.max_latency = info.max_latency
                self.min_latency = info.min_latency
            else:
                assert info.func_name in LOG_FN_SET
                self.sub_invocations[info.func_name] = info
    
    def ratio_of(self, fn_name):
        if isinstance(fn_name, list):
            return sum([self.ratio_of(single_op_name) for single_op_name in fn_name])
        else:
            assert fn_name not in FN_SET
            assert fn_name in LOG_FN_SET
            if fn_name in self.sub_invocations:
                return self.sub_invocations[fn_name].latency / self.latency
            else:
                return 0.0
    
    def ratio_of_slog(self):
        return sum([i.latency for i in self.sub_invocations.values()]) / self.latency


if __name__ == '__main__':
    logs = []
    count = -1
    with open('/tmp/retwis.log', 'r') as f:
        if count == -1:
            logs = f.readlines()
        else:
            for i, line in enumerate(f):
                if i >= count:
                    break
                logs.append(line)

    # 2023/10/18 09:40:16 tracer.go:42: [APITRACE] map[PostList:2342(n=1 max=2342 min=2342) ReadPrev:2082(n=4 max=767 min=224) SetAuxData:63(n=2 max=33 min=30)]
    def __extract_get_query_ratio(entry):
        res1 = re.findall(r'.*?map\[(.*)\]', entry)
        if len(res1) == 0:
            return None
        assert len(res1) == 1
        record_line = res1[0]
        entries = re.findall(r'(\w+):(\d+)\(n=(\d+) max=(\d+) min=(\d+)\)', record_line)
        line = InvocationInfo(entries)
        return (line.latency, line.ratio_of_slog(), line.ratio_of('Append'), line.ratio_of(['ReadPrev', 'ReadNext']), line.ratio_of('SetAuxData'))
    keys = ('latency', 'slog ratio', 'Append ratio', 'Read ratio', 'SetAuxData ratio')
    query_stat = gather_info(logs, __extract_get_query_ratio)
    query_stat = list(zip(*query_stat))     # transpose
    if len(query_stat) > 0:
        assert len(keys) == len(query_stat)
        print('LogAPIStat')
        for idx, v in enumerate(query_stat):
            print_percentile(keys[idx], v)
        print('')

    # plt.figure()
    # x, CDF_counts = np.unique(append_ratio, return_counts = True)
    # y = np.cumsum(CDF_counts)/np.sum(CDF_counts)
    # plt.plot(x, y)
    # plt.savefig('/tmp/queue_prof.png')


    # pop_statistics = []
    # for log in logs:
    #     found = re.findall(r'pop empty=(\d+)', log)
    #     if len(found) > 0:
    #         assert len(found) == 1
    #         pop_statistics.append(int(found[0]))

    # p50, p90, p99, p100, count = percentile(pop_statistics)
    # print(f'p50: {p50}; p90: {p90}; p99: {p99}; p100: {p100}; count: {count}')
