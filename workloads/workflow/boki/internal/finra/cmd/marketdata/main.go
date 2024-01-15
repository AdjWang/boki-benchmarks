package main

import (
	"cs.utexas.edu/zjia/faas"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/eniac/Beldi/internal/finra/internal/yfinance"
	"github.com/eniac/Beldi/pkg/cayonlib"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type RPCInput struct {
	Function string
	Input    interface{}
}

func handler(event aws.JSONValue) aws.JSONValue {
	portfolioType := event["portfolioType"].(string)

	tickersForPortfolioTypes := make(map[string][]string)
	types := make([]string, 3)
	types[0] = "GOOG"
	types[1] = "AMZN"
	types[2] = "MSFT"
	tickersForPortfolioTypes["S&P"] = types
	tickers := tickersForPortfolioTypes[portfolioType]

	prices := make(map[string]float64)
	for _, ticker := range tickers {
		// external service
		price, err := yfinance.GetLastClosingPrice(ticker)
		cayonlib.CHECK(errors.Wrapf(err, "GetLastClosingPrice failed. Ticker=%s", ticker))

		prices[ticker] = price
	}
	// prices = {'GOOG': 1732.38, 'AMZN': 3185.27, 'MSFT': 221.02}
	return aws.JSONValue{
		"statusCode": 200,
		"body": aws.JSONValue{
			"marketData": prices,
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
