import re
import numpy as np
import matplotlib.pyplot as plt


def percentile(datas):
    def __percentile(data_list, p):
        p = np.percentile(data_list, p, axis=0)
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
    print(f'{title} p30: {p30}; p50: {p50}; p70: {p70}; p90: {p90}; p99: {p99}; p100: {p100}; count: {count}')


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

    read_count_statistics = []
    apply_count_statistics = []
    for log in logs:
        found = re.findall(r'pop=(\d+) append=(\d+) read=(\d+) count r/a=\((\d+) (\d+)\)', log)
        if len(found) > 0:
            assert len(found) == 1
            pop, append, read, read_count, apply_count = found[0]
            pop, append, read, read_count, apply_count = int(pop), int(append), int(read), int(read_count), int(apply_count)
            pop_statistics.append(pop)
            append_statistics.append(append)
            read_statistics.append((read, read_count))
            append_ratio.append(read/append)

            read_count_statistics.append(read_count)
            apply_count_statistics.append(apply_count)

    print_percentile('pop', pop_statistics)
    print_percentile('append', append_statistics)
    print_percentile('read', read_statistics)
    print_percentile('aar', append_ratio)

    print_percentile('read_count', read_count_statistics)
    print_percentile('apply_count', apply_count_statistics)

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
