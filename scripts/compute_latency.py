#!/usr/bin/python3
import base64
import json
import argparse

import numpy as np

def parse_async_results(file_path):
    results = []
    with open(file_path) as fin:
        for line in fin:
            results.append(json.loads(line.strip()))
    return results

def parse_log_trace_time(raw_data):
    json_data = base64.b64decode(raw_data)
    output = json.loads(json_data)
    return output.get('Trace')

def parse_trace_data(async_result_file_path, warmup_ratio=1.0/6, outlier_ratio=30):
    success_count = 0
    traces = []
    queueing_delays = []
    latencies = []
    results = parse_async_results(async_result_file_path)
    skip = int(len(results) * warmup_ratio)
    for entry in results[skip:]:
        success_flag = entry['success']
        recv_ts = entry['recvTs']
        dispatch_ts = entry['dispatchTs']
        finish_ts = entry['finishedTs']
        trace_us = parse_log_trace_time(entry['output'])
        if trace_us:
            traces.append(int(trace_us) / (finish_ts - dispatch_ts))
        if success_flag:
            success_count += 1
        if dispatch_ts > recv_ts:
            queueing_delays.append(dispatch_ts - recv_ts)
        latencies.append(finish_ts - dispatch_ts)

    threshold = np.median(latencies) * outlier_ratio
    ignored = np.sum(np.array(latencies > threshold))
    # filtered = list(filter(lambda x: x < threshold, latencies))
    filtered_latencies = latencies
    filtered_traces = traces
    return filtered_latencies, filtered_traces, success_count / len(results[skip:])

def get_percentages(datas):
    p50 = np.percentile(datas, 50)
    p99 = np.percentile(datas, 99)
    p99_9 = np.percentile(datas, 99.9)
    return p50, p99, p99_9

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--async-result-file', type=str, default=None)
    parser.add_argument('--warmup-ratio', type=float, default=1.0/6)
    parser.add_argument('--outlier-factor', type=int, default=30)
    args = parser.parse_args()

    latencies, log_traces, success_percentage = parse_trace_data(args.async_result_file,
                                                                 warmup_ratio=args.warmup_ratio,
                                                                 outlier_ratio=args.outlier_factor)
    p50, p99, p99_9 = get_percentages(latencies)
    print('p50 latency: %.2f ms' % (p50/1000.0))
    print('p99 latency: %.2f ms' % (p99/1000.0))
    print('p99_9 latency: %.2f ms' % (p99_9/1000.0))

    if len(log_traces) > 0:
        p50, p99, p99_9 = get_percentages(log_traces)
        print('p50 log percentage: %.2f%%' % (p50*100))
        print('p99 log percentage: %.2f%%' % (p99*100))
        print('p99_9 log percentage: %.2f%%' % (p99_9*100))
    else:
        print('no log traces')

    print('p_success: %.2f' % success_percentage)
