import json
import numpy as np
from matplotlib import pyplot as plt


def parse_async_results(file_path):
    results = []
    with open(file_path) as fin:
        for line in fin:
            results.append(json.loads(line.strip()))
    return results


def compute_latency(results, warmup_ratio=1.0/6, outlier_ratio=30):
    success_count = 0
    queueing_delays = []
    latencies = []
    skip = int(len(results) * warmup_ratio)
    for entry in results[skip:]:
        success_flag = entry['success']
        recv_ts = entry['recvTs']
        dispatch_ts = entry['dispatchTs']
        finish_ts = entry['finishedTs']
        if success_flag:
            success_count += 1
        if dispatch_ts > recv_ts:
            queueing_delays.append(dispatch_ts - recv_ts)
        latencies.append(finish_ts - dispatch_ts)
    threshold = np.median(latencies) * outlier_ratio
    ignored = np.sum(np.array(latencies > threshold))
    # filtered = list(filter(lambda x: x < threshold, latencies))
    filtered = latencies
    
    plt.plot(filtered)
    plt.show()

if __name__ == '__main__':
    async_results_file = '/home/ubuntu/boki-benchmarks/experiments/workflow/boki-movie-baseline/results/qps700/async_results'
    results = parse_async_results(async_results_file)
    compute_latency(results)
