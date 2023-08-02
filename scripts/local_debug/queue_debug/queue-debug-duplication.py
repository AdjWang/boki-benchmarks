from pathlib import Path
import sys
import re
from collections import Counter


def extract_pops(log_file):
    client_count = 0
    client_pops = []
    with open(log_file, 'r') as f:
        for line in f:
            pops = re.findall(r'Consumer messages: \d+ \[(.*)\]', line)
            if len(pops) > 0:
                assert len(pops) == 1
                seqnums = [int(i) for i in pops[0].split(' ')]
                client_pops.append(
                    {'client': client_count, 'seqnums': seqnums})
                client_count += 1
    return client_pops


if __name__ == '__main__':
    script_dir = Path(sys.argv[0]).parent
    bench_output = script_dir / 'output2.log'
    client_pops = extract_pops(bench_output)

    seqnums = [seqnum for i in client_pops for seqnum in i['seqnums']]
    seqnums = sorted(seqnums)
    print(seqnums)

    producers = 1
    abnormal = [(seqnum, count) for seqnum, count in Counter(seqnums).items() if count != producers]

    print(abnormal)
