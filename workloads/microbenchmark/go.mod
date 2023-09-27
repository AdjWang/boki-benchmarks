module cs.utexas.edu/zjia/microbenchmark

go 1.19

require (
	cs.utexas.edu/zjia/faas v0.0.0
	github.com/montanaflynn/stats v0.7.1
)

require (
	github.com/iceber/iouring-go v0.0.0-20230403020409-002cfd2e2a90 // indirect
	golang.org/x/sys v0.0.0-20200923182605-d9f96fdee20d // indirect
)

replace cs.utexas.edu/zjia/faas => /home/ubuntu/boki-benchmarks/boki/worker/golang

replace github.com/eniac/Beldi => /home/ubuntu/boki-benchmarks/workloads/workflow/asynclog
