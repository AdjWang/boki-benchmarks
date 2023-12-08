package main

import (
	// "github.com/aws/aws-lambda-go/lambda"

	"time"

	"github.com/eniac/Beldi/internal/txnbench/internal/common"
	"github.com/eniac/Beldi/pkg/cayonlib"

	"cs.utexas.edu/zjia/faas"
	slib "cs.utexas.edu/zjia/faas/slib/common"
)

func Handler(env *cayonlib.Env) interface{} {
	logAPITs := time.Now()
	defer func() {
		latency := time.Since(logAPITs).Microseconds()
		slib.AppendTrace(env.FaasCtx, "Overall", latency)
	}()

	tsBeginTxn := time.Now()
	cayonlib.BeginTxn(env)
	slib.AppendTrace(env.FaasCtx, "BeginTxn", time.Since(tsBeginTxn).Microseconds())

	tsDBOps := time.Now()
	input := map[string]string{
		"table": common.TableReadOnly,
		"key":   common.DefaultKey,
	}
	res, _ := cayonlib.SyncInvoke(env, common.FnDBOps, common.RPCInput{
		Function: common.FnDBReadOnly,
		Input:    input,
	})
	slib.AppendTrace(env.FaasCtx, "DBOps", time.Since(tsDBOps).Microseconds())

	if !res.(bool) {
		tsAbortTxn := time.Now()
		cayonlib.AbortTxn(env)
		slib.AppendTrace(env.FaasCtx, "AbortTxn", time.Since(tsAbortTxn).Microseconds())
		return "ReadOnly Txn Fails"
	}

	tsCommitTxn := time.Now()
	cayonlib.CommitTxn(env)
	slib.AppendTrace(env.FaasCtx, "CommitTxn", time.Since(tsCommitTxn).Microseconds())
	return "ReadOnly Txn Success"
}

func main() {
	faas.Serve(cayonlib.CreateFuncHandlerFactory(Handler))
}
