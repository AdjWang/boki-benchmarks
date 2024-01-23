module github.com/eniac/Beldi

go 1.14

require (
	cs.utexas.edu/zjia/faas v0.0.0
	github.com/aws/aws-lambda-go v1.19.1
	github.com/aws/aws-sdk-go v1.34.6
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/google/uuid v1.1.1 // indirect
	github.com/hailocab/go-geoindex v0.0.0-20160127134810-64631bfe9711
	github.com/lithammer/shortuuid v3.0.0+incompatible
	github.com/mitchellh/mapstructure v1.3.3
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0 // indirect
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb // indirect
)

replace cs.utexas.edu/zjia/faas => /home/ubuntu/boki-benchmarks/boki/worker/golang

replace cs.utexas.edu/zjia/faas/slib => /home/ubuntu/boki-benchmarks/boki/slib
