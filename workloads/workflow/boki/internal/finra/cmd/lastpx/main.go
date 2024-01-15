package main

import (
	"log"
	"strings"

	"cs.utexas.edu/zjia/faas"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/eniac/Beldi/internal/finra/internal/data"
	"github.com/eniac/Beldi/pkg/cayonlib"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type RPCInput struct {
	Function string
	Input    interface{}
}

func handler(event aws.JSONValue) aws.JSONValue {
	portfolio := event["portfolio"].(string)
	portfolios, err := data.ReadPortfolios()
	cayonlib.CHECK(errors.Wrap(err, "ReadPortfolios failed"))
	data := portfolios[portfolio]
	valid := true
	for _, trade := range data {
		px := trade.LastPx.String()
		if strings.Contains(px, ".") {
			group := strings.Split(px, ".")
			a := group[0]
			b := group[1]
			if !((len(a) == 3 && len(b) == 6) ||
				(len(a) == 4 && len(b) == 5) ||
				(len(a) == 5 && len(b) == 4) ||
				(len(a) == 5 && len(b) == 3)) {
				log.Printf("%s: %dv%d", px, len(a), len(b))
				valid = false
				break
			}
		}
	}
	return aws.JSONValue{
		"statusCode": 200,
		"body": aws.JSONValue{
			"valid":     valid,
			"portfolio": portfolio,
		}}
}

func Handler(env *cayonlib.Env) interface{} {
	var rpcInput RPCInput
	err := mapstructure.Decode(env.Input, &rpcInput)
	cayonlib.CHECK(errors.Wrapf(err, "Decode input failed. Input=%+v", env.Input))
	req := rpcInput.Input.(map[string]interface{})

	// logAPITs := time.Now()
	// defer func() {
	// 	latency := time.Since(logAPITs).Microseconds()
	// 	slib.AppendTrace(env.FaasCtx, fmt.Sprintf("xxx-%s", rpcInput.Function), latency)
	// }()

	return handler(aws.JSONValue(req["body"].(map[string]interface{})))
}

func main() {
	faas.Serve(cayonlib.CreateFuncHandlerFactory(Handler))
}
