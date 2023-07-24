module cs.utexas.edu/zjia/microbenchmark

go 1.19

require (
	cs.utexas.edu/zjia/faas v0.0.0
	github.com/montanaflynn/stats v0.7.1
)

require (
	github.com/enriquebris/goconcurrentqueue v0.7.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
)

replace cs.utexas.edu/zjia/faas => /home/adjwang/dev/boki-benchmarks/boki/worker/golang

replace github.com/eniac/Beldi => /home/adjwang/dev/boki-benchmarks/workloads/workflow/asynclog
