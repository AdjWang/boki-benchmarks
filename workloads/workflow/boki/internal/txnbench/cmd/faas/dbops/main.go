package main

import (
	"fmt"
	"time"

	"cs.utexas.edu/zjia/faas"
	slib "cs.utexas.edu/zjia/faas/slib/common"
	"github.com/eniac/Beldi/internal/txnbench/internal/common"
	"github.com/eniac/Beldi/internal/txnbench/internal/dbops"
	"github.com/eniac/Beldi/pkg/cayonlib"
	"github.com/mitchellh/mapstructure"
)

func Handler(env *cayonlib.Env) interface{} {
	var rpcInput common.RPCInput
	cayonlib.CHECK(mapstructure.Decode(env.Input, &rpcInput))
	req := rpcInput.Input.(map[string]interface{})

	logAPITs := time.Now()
	defer func() {
		latency := time.Since(logAPITs).Microseconds()
		slib.AppendTrace(env.FaasCtx, fmt.Sprintf("DBOp-%s", rpcInput.Function), latency)
	}()

	switch rpcInput.Function {
	case common.FnDBReadOnly:
		res := dbops.DBRead(env, req["table"].(string), req["key"].(string))
		return res
	case common.FnDBWriteOnly:
		res := dbops.DBWrite(env, req["table"].(string), req["key"].(string))
		return res
	default:
		panic(fmt.Sprintf("function=%s not found", rpcInput.Function))
	}
}

func main() {
	faas.Serve(cayonlib.CreateFuncHandlerFactory(Handler))
}
