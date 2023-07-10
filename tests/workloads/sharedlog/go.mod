module cs.utexas.edu/zjia/nightcore-example

go 1.19

require (
	cs.utexas.edu/zjia/faas v0.0.0
	cs.utexas.edu/zjia/faas/slib v0.0.0-00010101000000-000000000000
	github.com/eniac/Beldi v0.0.0-00010101000000-000000000000
	github.com/pkg/errors v0.9.1
)

require (
	github.com/Jeffail/gabs/v2 v2.6.0 // indirect
	github.com/aws/aws-sdk-go v1.34.6 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-redis/redis/v8 v8.8.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/jmespath/go-jmespath v0.3.0 // indirect
	github.com/lithammer/shortuuid v3.0.0+incompatible // indirect
	github.com/mitchellh/mapstructure v1.3.3 // indirect
	go.opentelemetry.io/otel v0.19.0 // indirect
	go.opentelemetry.io/otel/metric v0.19.0 // indirect
	go.opentelemetry.io/otel/trace v0.19.0 // indirect
)

replace cs.utexas.edu/zjia/faas => /home/adjwang/dev/boki-benchmarks/boki/worker/golang

replace github.com/eniac/Beldi => /home/adjwang/dev/boki-benchmarks/workloads/workflow/asynclog

replace cs.utexas.edu/zjia/faas/slib => /home/adjwang/dev/boki-benchmarks/boki/slib
