#!/usr/bin/python3
import os
import json
import argparse

import numpy as np

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
    p50 = np.percentile(filtered, 50) / 1000.0
    p99 = np.percentile(filtered, 99) / 1000.0
    p99_9 = np.percentile(filtered, 99.9) / 1000.0
    p_success = success_count / len(results[skip:])
    return p50, p99, p99_9, p_success

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--async-result-file', type=str, default=None)
    parser.add_argument('--warmup-ratio', type=float, default=1.0/6)
    parser.add_argument('--outlier-factor', type=int, default=30)
    args = parser.parse_args()

    results = parse_async_results(args.async_result_file)
    if len(results) == 0:
        print('no async results')
        exit(0)
    p50, p99, p99_9, p_success = compute_latency(results,
                               warmup_ratio=args.warmup_ratio,
                               outlier_ratio=args.outlier_factor)
    print('p50 latency: %.2f ms' % p50)
    print('p99 latency: %.2f ms' % p99)
    print('p99_9 latency: %.2f ms' % p99_9)
    print('p_success: %.2f' % p_success)
