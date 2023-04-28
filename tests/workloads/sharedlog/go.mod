module cs.utexas.edu/zjia/nightcore-example

go 1.19

require (
	cs.utexas.edu/zjia/faas v0.0.0
	github.com/eniac/Beldi v0.0.0-00010101000000-000000000000
)

require (
	github.com/aws/aws-sdk-go v1.34.6 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/jmespath/go-jmespath v0.3.0 // indirect
	github.com/lithammer/shortuuid v3.0.0+incompatible // indirect
	github.com/mitchellh/mapstructure v1.3.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
)

replace cs.utexas.edu/zjia/faas => /home/adjwang/dev/boki-benchmarks/boki/worker/golang

replace github.com/eniac/Beldi => /home/adjwang/dev/boki-benchmarks/workloads/workflow/asynclog
