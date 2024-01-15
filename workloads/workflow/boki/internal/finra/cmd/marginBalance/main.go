package main

import (
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

func checkMarginBalance(portfolioData []data.Trade, marketData map[string]interface{}, portfolio string) bool {
	marginAccountBalanceData, err := data.ReadMarginBalance()
	cayonlib.CHECK(errors.Wrap(err, "ReadMarginBalance failed"))
	marginAccountBalance := marginAccountBalanceData[portfolio]
	portfolioMarketValue := float64(0)
	for _, trade := range portfolioData {
		security := trade.Security
		qty, err := trade.LastQty.Int64()
		cayonlib.CHECK(errors.Wrapf(err, "Convert LastQty to Int64 failed. LastQty=%+v", trade.LastQty))
		portfolioMarketValue += float64(qty) * marketData[security].(float64)
	}
	// Maintenance Margin should be atleast 25% of market value for "long" securities
	// https://www.finra.org/rules-guidance/rulebooks/finra-rules/4210#the-rule
	result := false
	if float64(marginAccountBalance) >= 0.25*portfolioMarketValue {
		result = true
	}
	return result
}

func handler(events []aws.JSONValue) aws.JSONValue {
	var marketData map[string]interface{}
	var portfolio string
	validFormat := true
	for _, event := range events {
		for _, fnOutput := range event {
			body := aws.JSONValue(fnOutput.(map[string]interface{}))["body"].(map[string]interface{})
			if _, found := body["marketData"]; found {
				marketData = body["marketData"].(map[string]interface{})
			} else if _, found := body["valid"]; found {
				portfolio = body["portfolio"].(string)
				validFormat = validFormat && body["valid"].(bool)
			} else {
				panic("unreachable")
			}
		}
	}
	portfolios, err := data.ReadPortfolios()
	cayonlib.CHECK(errors.Wrap(err, "ReadPortfolios failed"))
	portfolioData := portfolios[portfolio]
	marginSatisfied := checkMarginBalance(portfolioData, marketData, portfolio)
	return aws.JSONValue{
		"statusCode": 200,
		"body": aws.JSONValue{
			"validFormat":     validFormat,
			"marginSatisfied": marginSatisfied,
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

	// cast []interface{} to []aws.JSONValue
	reqBodyMap := req["body"].([]interface{})
	reqBodyJson := make([]aws.JSONValue, 0, len(reqBodyMap))
	for _, reqBody := range reqBodyMap {
		reqBodyJson = append(reqBodyJson, aws.JSONValue(reqBody.(map[string]interface{})))
	}
	return handler(reqBodyJson)
}

func main() {
	faas.Serve(cayonlib.CreateFuncHandlerFactory(Handler))
}
