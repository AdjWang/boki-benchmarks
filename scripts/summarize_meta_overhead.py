# -*- conding: utf-8 -*-

import re
import sys
from dataclasses import dataclass


@dataclass
class LogAppendMetaInfo:
    tag: str
    original_data_length: int
    new_data_length: int

    @property
    def overhead_len(self):
        return self.new_data_length - self.original_data_length

    @property
    def overhead_per(self):
        return self.overhead_len / self.new_data_length

    def __str__(self):
        return f'overhead len={self.overhead_len}, per={self.overhead_per:.2f}'


def summary(infos):
    def __metrics(datas):
        datas = sorted(datas)
        min_val, max_val = datas[0], datas[-1]
        sum_ = sum(datas)
        avg = sum_ / len(datas)
        return min_val, max_val, avg

    print(f'total: {len(infos)}')

    lens = [i.overhead_len for i in infos]
    min_len, max_len, avg_len = __metrics(lens)
    print(f'len min={min_len}, max={max_len}, avg={int(avg_len)}')

    pers = [i.overhead_per for i in infos]
    min_per, max_per, avg_per = __metrics(pers)
    print(f'per min={min_per:.2f}, max={max_per:.2f}, avg={avg_per:.2f}')


"""
usage: 
    1. uncomment all the TRACE lines in func_worker.go and controlflow.go
       (find them by search all of "// TRACE: report new meta data overhead")
       then run asynclog experiments
    2. select proper RECORD_TYPE below then run
       find <engine log dir>/mnt/inmem* -type f | xargs grep "TRACE" | pythonn3 summarize_meta_overhead.py

expected output:
    # hotel func invoke
    total: 1682
    len min=68, max=1344, avg=100
    per min=0.20, max=0.83, avg=0.28

    # hotel log append
    total: 10230
    len min=120, max=767, avg=206
    per min=0.29, max=0.68, avg=0.53

    # movie func invoke
    total: 8272
    len min=68, max=211, avg=158
    per min=0.21, max=0.44, avg=0.33

    # movie log append
    total: 48880
    len min=120, max=299, avg=173
    per min=0.36, max=0.69, avg=0.56
"""
if __name__ == '__main__':
    RECORD_TYPE = 'FUNC_INVOKE'
    # RECORD_TYPE = 'LOG_APPEND'

    log_append_meta_infos = []
    if RECORD_TYPE == 'LOG_APPEND':
        for line in sys.stdin:
            infos = re.findall(r'log type=(.*), boki data len=(\d+), new data len=(\d+)', line)
            if len(infos) == 0:
                continue
            assert len(infos[0]) == 3, f'{infos[0]}, len={len(infos[0])}'
            tag = infos[0][0]
            original_data_len = int(infos[0][1])
            new_data_len = int(infos[0][2])
            log_append_meta_infos.append(LogAppendMetaInfo(tag, original_data_len, new_data_len))

    elif RECORD_TYPE == 'FUNC_INVOKE':
        for line in sys.stdin:
            infos = re.findall(r'InstanceId=(.*), ctx data len=(\d+), total data len=(\d+)', line)
            if len(infos) == 0:
                continue
            assert len(infos) == 1, f'{infos}, len={len(infos)}'
            assert len(infos[0]) == 3, f'{infos[0]}, len={len(infos[0])}'
            tag = infos[0][0]
            meta_data_len = int(infos[0][1])
            new_data_len = int(infos[0][2])
            original_data_len = new_data_len - meta_data_len
            log_append_meta_infos.append(LogAppendMetaInfo(tag, original_data_len, new_data_len))

    # for info in log_append_meta_infos:
    #     print(info)
    summary(log_append_meta_infos)
