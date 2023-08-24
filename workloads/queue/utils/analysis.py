import re
import numpy as np
import matplotlib.pyplot as plt


def percentile(datas):
    p50 = np.percentile(datas, 50)
    p90 = np.percentile(datas, 90)
    p99 = np.percentile(datas, 99)
    return p50, p90, p99, max(datas), len(datas)


if __name__ == '__main__':
    logs = []
    count = -1
    with open('/tmp/queue.log', 'r') as f:
        if count == -1:
            logs = f.readlines()
        else:
            for i, line in enumerate(f):
                if i >= count:
                    break
                logs.append(line)

    pop_statistics = []
    append_statistics = []
    read_statistics = []
    append_ratio = []
    for log in logs:
        found = re.findall(r'pop=(\d+) append=(\d+) read=(\d+)', log)
        if len(found) > 0:
            assert len(found) == 1
            pop, append, read = found[0]
            pop, append, read = int(pop), int(append), int(read)
            pop_statistics.append(pop)
            append_statistics.append(append)
            read_statistics.append(read)
            append_ratio.append(read/append)

    assert len(pop_statistics) > 0
    p50, p90, p99, p100, count = percentile(append_ratio)
    print(f'p50: {p50}; p90: {p90}; p99: {p99}; p100: {p100}; count: {count}')

    plt.figure()
    x, CDF_counts = np.unique(append_ratio, return_counts = True)
    y = np.cumsum(CDF_counts)/np.sum(CDF_counts)
    plt.plot(x, y)
    plt.savefig('/tmp/queue_prof.png')


    # pop_statistics = []
    # for log in logs:
    #     found = re.findall(r'pop empty=(\d+)', log)
    #     if len(found) > 0:
    #         assert len(found) == 1
    #         pop_statistics.append(int(found[0]))

    # p50, p90, p99, p100, count = percentile(pop_statistics)
    # print(f'p50: {p50}; p90: {p90}; p99: {p99}; p100: {p100}; count: {count}')
