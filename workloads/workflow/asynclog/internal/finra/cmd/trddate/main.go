package main

import (
	"cs.utexas.edu/zjia/faas"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/eniac/Beldi/internal/finra/internal/data"
	"github.com/eniac/Beldi/pkg/cayonlib"
	"github.com/itchyny/timefmt-go"
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
		trddate := trade.TradeDate
		// Tag ID: 75, Tag Name: TradeDate, Format: YYMMDD
		if len(trddate) == 6 {
			_, err := timefmt.Parse(trddate, "%y%m%d")
			if err != nil {
				valid = false
				break
			}
		} else {
			valid = false
			break
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

	return handler(req["body"].(map[string]interface{}))
}

func main() {
	faas.Serve(cayonlib.CreateFuncHandlerFactory(Handler))
}
