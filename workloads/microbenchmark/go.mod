module cs.utexas.edu/zjia/microbenchmark

go 1.19

require (
	cs.utexas.edu/zjia/faas v0.0.0
	github.com/montanaflynn/stats v0.7.1
)

replace cs.utexas.edu/zjia/faas => /home/ubuntu/boki-benchmarks/boki/worker/golang

replace github.com/eniac/Beldi => /home/ubuntu/boki-benchmarks/workloads/workflow/asynclog
