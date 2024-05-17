#   zookeeper-setup:
#     image: zookeeper:3.6.2
#     command: /tmp/boki/zk_setup.sh
#     depends_on:
#        - zookeeper
#     volumes:
#       - /tmp/zk_setup.sh:/tmp/boki/zk_setup.sh
#     restart: always
#     healthcheck:
#       test: /tmp/zk_health_check || exit 1
#       interval: 60s
#       retries: 5
#       start_period: 20s
#       timeout: 10s
all:
	go build -buildvcs=false -o zk_health_check main.go