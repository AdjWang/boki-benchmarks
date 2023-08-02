import re


def extract_logs(log_file):
    queue_logs = []
    with open(log_file, 'r') as f:
        for line in f:
            logs = re.findall(r'queue.go:(\d+): (\d+) (\d+) (\w+) (.*)', line)
            if len(logs) > 0:
                assert len(logs) == 1
                queue_logs.append(
                    {'line': logs[0][0], 'ts': logs[0][1], 'shard': int(logs[0][2]), 'op': logs[0][3], 'log': logs[0][4]})
    return queue_logs


if __name__ == '__main__':
    producer_stderr = '/tmp/boki-test/mnt/inmem1/boki/output/slibQueueProducer_worker_0.stderr'
    consumer_stderr = '/tmp/boki-test/mnt/inmem1/boki/output/slibQueueConsumer_worker_0.stderr'
    producer_logs = extract_logs(producer_stderr)
    consumer_logs = extract_logs(consumer_stderr)
    # print(producer_logs)
    # print(consumer_logs)

    logs = producer_logs + consumer_logs
    logs = sorted(logs, key=lambda entry: entry['ts'])
    # print(logs)
    with open('/tmp/queue_logs.log', 'w') as f:
        f.writelines(
            [f"{log['line']}:{log['op']+' '+log['log']}\n" for log in logs if log['shard'] == 14 or log['op'] == 'push'])
