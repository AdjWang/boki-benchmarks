package main

import (
	"sync"

	"cs.utexas.edu/zjia/faas"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/eniac/Beldi/pkg/cayonlib"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type RPCInput struct {
	Function string
	Input    interface{}
}

func handler(env *cayonlib.Env, event aws.JSONValue) aws.JSONValue {
	nParallel := int(event["n_parallel"].(float64))
	outputs := make([]aws.JSONValue, 0, nParallel)
	// fan out
	wg := sync.WaitGroup{}
	for i := 0; i < nParallel; i++ {
		funcs := []string{"marketdata", "lastpx", "side", "trddate", "volume"}
		arg := aws.JSONValue{
			"reqId": env.InstanceId,
			"body": aws.JSONValue{
				"portfolioType": "S&P",
				"portfolio":     "1234",
			}}
		groupOutputMu := sync.Mutex{}
		groupOutput := aws.JSONValue{}
		outputs = append(outputs, groupOutput)
		for _, funcName := range funcs {
			fn := cayonlib.ProposeInvoke(env, funcName)
			wg.Add(1)
			go func(callee string) {
				defer wg.Done()
				output, _ := cayonlib.AssignedSyncInvoke(env, callee, RPCInput{
					Function: callee,
					Input:    arg,
				}, fn)
				groupOutputMu.Lock()
				groupOutput[callee] = output
				groupOutputMu.Unlock()
			}(funcName)
		}
	}
	wg.Wait()
	// fan in
	output, _ := cayonlib.SyncInvoke(env, "marginBalance", RPCInput{
		Function: "marginBalance",
		Input:    aws.JSONValue{"reqId": env.InstanceId, "body": outputs},
	})
	return aws.JSONValue(output.(map[string]interface{}))
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

	return handler(env, aws.JSONValue(req["body"].(map[string]interface{})))
}

func main() {
	faas.Serve(cayonlib.CreateFuncHandlerFactory(Handler))
}
