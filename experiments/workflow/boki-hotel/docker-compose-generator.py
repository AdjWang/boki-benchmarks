import argparse
import re

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--docker-compose', type=str, default=None, required=True)
    parser.add_argument('--tracer-host', type=str, default=None, required=True)
    parser.add_argument('--zookeeper-host', type=str, default=None, required=True)
    parser.add_argument('--output', type=str, default='docker-compose.yml')
    args = parser.parse_args()

    with open(args.docker_compose, 'r') as f:
        template = f.read()
    
    template = re.sub(r'%pytemp_tracer_host%', args.tracer_host, template)
    template = re.sub(r'%pytemp_zookeeper_host%', args.zookeeper_host, template)
    
    with open(args.output, 'w') as f:
        f.write(template)
