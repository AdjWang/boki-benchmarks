import os
import re
import numpy as np


def from_seg(segs):
    prof_infos = []
    for seg in segs:
        seg_info = re.findall(r'ts=(\d+) tip=(.*)', seg)
        assert len(seg_info) == 1
        prof_infos.append({'ts': int(seg_info[0][0]), 'tip': seg_info[0][1]})

    diffs = dict()
    for i in range(len(prof_infos)-1):
        op1 = prof_infos[i]
        op2 = prof_infos[i+1]
        diffs[f"{op1['tip']}-{op2['tip']}"] = (op2['ts']-op1['ts'])

    t_sum = sum([t for t in diffs.values()])
    outputs = {'sum': t_sum}
    for op, dur in diffs.items():
        outputs[op] = dur / t_sum
    return outputs


def extract_prof_infos(logs):
    infos = []
    for log in logs:
        t = re.findall(r'prof info=(.*)', log)
        if len(t) > 0:
            assert len(t) == 1
            info_line = t[0].strip(';').split(';')
            infos.append(from_seg(info_line))
    return infos


def summary_infos(infos, key):
    data = [i[key] for i in infos if key in i]
    if key != 'sum':
        return get_avgrage(data)
    else:
        return get_percentile(data)


def extract_prof_summaries(logs, dir):
    infos = []
    for log in logs:
        t = re.findall(rf'dir={dir} hop_times=0 elapsed=(\d+)ns', log)
        if len(t) > 0:
            assert len(t) == 1
            infos.append({'dir': 'kReadNextU', 'elapsed': int(t[0])})
    return infos


def get_avgrage(datas):
    if len(datas) == 0:
        return 0
    return sum(datas)/len(datas)


def get_percentile(datas, factor=1000.0):
    if len(datas) == 0:
        return None, None, None, None
    p50 = np.percentile(datas, 50) / factor
    p99 = np.percentile(datas, 99) / factor
    p99_9 = np.percentile(datas, 99.9) / factor
    return p50, p99, p99_9, len(datas)


if __name__ == '__main__':
    engine_logs = os.popen(
        "docker logs $(docker ps -a | grep engine | awk '{print $1}') 2>&1").readlines()
    print(len(engine_logs))

    summaries = extract_prof_summaries(engine_logs, 'kReadNext')
    durations = [i['elapsed'] for i in summaries]
    print(get_percentile(durations))

    summaries = extract_prof_summaries(engine_logs, 'kReadNextU')
    durations = [i['elapsed'] for i in summaries]
    print(get_percentile(durations))

    # infos = extract_prof_infos(engine_logs)
    # print(infos[0].keys())
    # # infos = sorted(infos, key=lambda info: info['sum'])
    # for key in ['sum', 'start-get index', 'get index-get view', 'get view-get range', 'get range-end']:
    #     print(key, summary_infos(infos, key))
