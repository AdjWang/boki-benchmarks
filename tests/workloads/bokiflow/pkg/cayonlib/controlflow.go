package cayonlib

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/pkg/errors"

	// "github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"

	// lambdaSdk "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/golang/snappy"
	"github.com/lithammer/shortuuid"
	"github.com/mitchellh/mapstructure"

	"cs.utexas.edu/zjia/faas/types"

	"context"
	// "strings"
	"time"
)

type InputWrapper struct {
	CallerName  string      `mapstructure:"CallerName"`
	CallerId    string      `mapstructure:"CallerId"`
	CallerStep  int32       `mapstructure:"CallerStep"`
	InstanceId  string      `mapstructure:"InstanceId"`
	Input       interface{} `mapstructure:"Input"`
	TxnId       string      `mapstructure:"TxnId"`
	Instruction string      `mapstructure:"Instruction"`
	Async       bool        `mapstructure:"Async"`

	// TODO: more graceful way to propagate
	// Currently mapstructure+json would lose the type of the field, like
	// []byte or types.FutureMeta, use string instead.
	AsyncLogPropagator        string `mapstructure:"AsyncLogPropagator"`
	LastStepLogMetaPropagator string `mapstructure:"LastStepLogMetaPropagator"`
	lastStepLogMeta           types.FutureMeta
}

func (iw *InputWrapper) Serialize() []byte {
	stream, err := json.Marshal(*iw)
	CHECK(err)
	return stream
}

func (iw *InputWrapper) Deserialize(stream []byte) {
	err := json.Unmarshal(stream, iw)
	CHECK(err)
}

type StackTraceCall struct {
	Label string `json:"label"`
	Line  int    `json:"line"`
	Path  string `json:"path"`
}

func (ie *InvokeError) Deserialize(stream []byte) {
	err := json.Unmarshal(stream, ie)
	CHECK(err)
	if ie.ErrorMessage == "" {
		panic(errors.New("never happen"))
	}
}

type InvokeError struct {
	ErrorMessage string           `json:"errorMessage"`
	ErrorType    string           `json:"errorType"`
	StackTrace   []StackTraceCall `json:"stackTrace"`
}

type OutputWrapper struct {
	Status string
	Output interface{}
}

func (ow *OutputWrapper) Serialize() []byte {
	stream, err := json.Marshal(*ow)
	CHECK(err)
	return stream
}

func (ow *OutputWrapper) Deserialize(stream []byte) {
	err := json.Unmarshal(stream, ow)
	CHECK(err)
	if ow.Status != "Success" && ow.Status != "Failure" {
		ie := InvokeError{}
		ie.Deserialize(stream)
		panic(ie)
	}
}

func ParseInput(raw interface{}) *InputWrapper {
	var iw InputWrapper
	if body, ok := raw.(map[string]interface{})["body"]; ok {
		CHECK(errors.Wrapf(json.Unmarshal([]byte(body.(string)), &iw), "invalid json raw=%+v", raw))
	} else {
		CHECK(errors.Wrapf(mapstructure.Decode(raw, &iw), "invalid mapstructure raw=%+v", raw))
	}
	return &iw
}

func PrepareEnv(iw *InputWrapper, lambdaId string) *Env {
	// s := strings.Split(lambdacontext.FunctionName, "-")
	// lambdaId := s[len(s)-1]
	if iw.InstanceId == "" {
		iw.InstanceId = shortuuid.New()
	}
	return &Env{
		LambdaId:    lambdaId,
		InstanceId:  iw.InstanceId,
		StepNumber:  0,
		Input:       iw.Input,
		TxnId:       iw.TxnId,
		Instruction: iw.Instruction,
	}
}

// func SyncInvoke(env *Env, callee string, input interface{}) (interface{}, string) {
// 	newLog, preInvokeLog := ProposeNextStep(env, aws.JSONValue{
// 		"type":       "PreInvoke",
// 		"instanceId": shortuuid.New(),
// 		"callee":     callee,
// 		"input":      input,
// 	})
// 	instanceId := preInvokeLog.Data["instanceId"].(string)
// 	if !newLog {
// 		CheckLogDataField(preInvokeLog, "type", "PreInvoke")
// 		log.Printf("[INFO] Seen PreInvoke log for step %d", preInvokeLog.StepNumber)
// 		resultLog := FetchStepResultLog(env, preInvokeLog.StepNumber /* catch= */, false)
// 		if resultLog != nil {
// 			CheckLogDataField(resultLog, "type", "InvokeResult")
// 			log.Printf("[INFO] Seen InvokeResult log for step %d", preInvokeLog.StepNumber)
// 			return resultLog.Data["output"], instanceId
// 		}
// 	}

// 	iw := InputWrapper{
// 		CallerName:  env.LambdaId,
// 		CallerId:    env.InstanceId,
// 		CallerStep:  preInvokeLog.StepNumber,
// 		Async:       false,
// 		InstanceId:  instanceId,
// 		Input:       input,
// 		TxnId:       env.TxnId,
// 		Instruction: env.Instruction,
// 	}
// 	if iw.Instruction == "EXECUTE" {
// 		LibAppendLog(env, TransactionStreamTag(env.LambdaId, env.TxnId), &TxnLogEntry{
// 			LambdaId: env.LambdaId,
// 			TxnId:    env.TxnId,
// 			Callee:   callee,
// 			WriteOp:  aws.JSONValue{},
// 		})
// 	}
// 	payload := iw.Serialize()
// 	res, err := env.FaasEnv.InvokeFunc(env.FaasCtx, callee, payload)
// 	CHECK(err)
// 	ow := OutputWrapper{}
// 	ow.Deserialize(res)
// 	switch ow.Status {
// 	case "Success":
// 		return ow.Output, iw.InstanceId
// 	default:
// 		panic("never happens")
// 	}
// }

type invokeRetValPair struct {
	output     interface{}
	instanceId string
}

func SyncInvoke(env *Env, callee string, input interface{}) (interface{}, string) {
	stepFuture, intentLog := AsyncProposeNextStep(env, aws.JSONValue{
		"type":       "PreInvoke",
		"instanceId": shortuuid.New(),
		"callee":     callee,
		"input":      input,
	}, /*deps*/ []types.FutureMeta{})
	// if stepFuture is nil, then intentLog is the recorded step before;
	// if stepFuture is not nil, then intentLog is the newly append step;
	fnGetLoggedStepResult := func(preInvokeLog *IntentLogEntry) (interface{}, string, bool) {
		instanceId := preInvokeLog.Data["instanceId"].(string)

		CheckLogDataField(preInvokeLog, "type", "PreInvoke")
		log.Printf("[INFO] Seen PreInvoke log for step %d", preInvokeLog.StepNumber)
		resultLog := FetchStepResultLog(env, preInvokeLog.StepNumber /* catch= */, false)
		if resultLog != nil {
			CheckLogDataField(resultLog, "type", "InvokeResult")
			log.Printf("[INFO] Seen InvokeResult log for step %d", preInvokeLog.StepNumber)
			return resultLog.Data["output"], instanceId, true
		} else {
			return nil, "", false
		}
	}
	if stepFuture != nil {
		env.FaasEnv.AsyncLogCtx().Chain(stepFuture.GetMeta())
	} else {
		if output, instanceId, ok := fnGetLoggedStepResult(intentLog); ok {
			return output, instanceId
		} else {
			panic("unreachable")
		}
	}
	// However the fsm maybe expired, so we still need to resolve it
	// to grab the last view of the step log here.
	// retValChan := make(chan invokeRetValPair)
	// DEBUG
	// go func(env *Env, future types.Future[uint64]) {
	// 	// sync
	// 	err := future.Await(10 * time.Second)
	// 	CHECK(err)
	// 	// resolve cond
	// 	lastStepLog, err := env.FaasEnv.AsyncSharedLogRead(env.FaasCtx, future.GetMeta())
	// 	CHECK(err)

	// 	logState := ResolveLog(env, lastStepLog)
	// 	if logState.State == LogEntryState_PENDING {
	// 		panic("impossible")
	// 	} else if logState.State == LogEntryState_DISCARDED {
	// 		if output, instanceId, ok := fnGetLoggedStepResult(intentLog); ok {
	// 			retValChan <- invokeRetValPair{output, instanceId}
	// 		}
	// 	} else if logState.State == LogEntryState_APPLIED {
	// 		// carry on
	// 	} else {
	// 		panic("unreachable")
	// 	}
	// }(env, stepFuture)

	future := stepFuture
	// go func(env *Env, future types.Future[uint64]) {
	// newLog would:
	// exert if cond is true
	// drop if cond is false
	instanceId := intentLog.Data["instanceId"].(string)
	stepNumber := intentLog.StepNumber

	asyncLogCtxData, err := env.FaasEnv.AsyncLogCtx().Serialize()
	CHECK(err)
	futureMetaData, err := future.GetMeta().Serialize()
	CHECK(err)

	iw := InputWrapper{
		CallerName:  env.LambdaId,
		CallerId:    env.InstanceId,
		CallerStep:  stepNumber,
		Async:       false,
		InstanceId:  instanceId,
		Input:       input,
		TxnId:       env.TxnId,
		Instruction: env.Instruction,

		AsyncLogPropagator:        string(asyncLogCtxData),
		LastStepLogMetaPropagator: string(futureMetaData),
	}
	if iw.Instruction == "EXECUTE" {
		env.FaasEnv.AsyncLogCtx().Chain(LibAsyncAppendLog(env, TransactionStreamTag(env.LambdaId, env.TxnId),
			[]types.TagMeta{
				{
					FsmType: FsmType_TRANSACTIONSTREAM,
					TagKeys: []string{env.LambdaId, env.TxnId},
				},
			},
			&TxnLogEntry{
				LambdaId: env.LambdaId,
				TxnId:    env.TxnId,
				Callee:   callee,
				WriteOp:  aws.JSONValue{},
			}, func(cond types.CondHandle) { cond.AddDep(future.GetMeta()) }).GetMeta())
	}
	payload := iw.Serialize()
	res, err := env.FaasEnv.InvokeFunc(env.FaasCtx, callee, payload)
	CHECK(errors.Wrapf(err, "InvokeFunc callee: %v, res: %v, payload: %+v", callee, res, iw))
	ow := OutputWrapper{}
	ow.Deserialize(res)
	switch ow.Status {
	case "Success":
		// retValChan <- invokeRetValPair{ow.Output, iw.InstanceId}
		return ow.Output, iw.InstanceId
	default:
		panic("never happens")
	}
	// }(env, stepFuture)

	// retVal := <-retValChan
	// return retVal.output, retVal.instanceId
}

// TODO: can implement by deps now
//
//	func ProposeInvoke(env *Env, callee string) *IntentLogEntry {
//		newLog, preInvokeLog := ProposeNextStep(env, aws.JSONValue{
//			"type":       "PreInvoke",
//			"instanceId": shortuuid.New(),
//			"callee":     callee,
//		})
//		if !newLog {
//			CheckLogDataField(preInvokeLog, "type", "PreInvoke")
//			CheckLogDataField(preInvokeLog, "callee", callee)
//			log.Printf("[INFO] Seen PreInvoke log for step %d", preInvokeLog.StepNumber)
//		}
//		return preInvokeLog
//	}
func ProposeInvoke(env *Env, callee string, input interface{}) (types.Future[uint64], *IntentLogEntry) {
	stepFuture, intentLog := AsyncProposeNextStep(env, aws.JSONValue{
		"type":       "PreInvoke",
		"instanceId": shortuuid.New(),
		"callee":     callee,
		"input":      input,
	}, /*deps*/ []types.FutureMeta{})
	if stepFuture != nil {
		env.FaasEnv.AsyncLogCtx().Chain(stepFuture.GetMeta())
	}
	return stepFuture, intentLog
}
func AssignedSyncInvoke(env *Env, callee string, stepFuture types.Future[uint64], intentLog *IntentLogEntry) (interface{}, string) {
	fnGetLoggedStepResult := func(preInvokeLog *IntentLogEntry) (interface{}, string, bool) {
		instanceId := preInvokeLog.Data["instanceId"].(string)

		CheckLogDataField(preInvokeLog, "type", "PreInvoke")
		log.Printf("[INFO] Seen PreInvoke log for step %d", preInvokeLog.StepNumber)
		resultLog := FetchStepResultLog(env, preInvokeLog.StepNumber /* catch= */, false)
		if resultLog != nil {
			CheckLogDataField(resultLog, "type", "InvokeResult")
			log.Printf("[INFO] Seen InvokeResult log for step %d", preInvokeLog.StepNumber)
			return resultLog.Data["output"], instanceId, true
		} else {
			return nil, "", false
		}
	}
	if stepFuture != nil {
		env.FaasEnv.AsyncLogCtx().Chain(stepFuture.GetMeta())
	} else {
		if output, instanceId, ok := fnGetLoggedStepResult(intentLog); ok {
			return output, instanceId
		} else {
			panic("unreachable")
		}
	}
	// However the fsm maybe expired, so we still need to resolve it
	// to grab the last view of the step log here.
	retValChan := make(chan invokeRetValPair)
	go func(env *Env, future types.Future[uint64]) {
		// sync
		err := future.Await(10 * time.Second)
		CHECK(err)
		// resolve cond
		lastStepLog, err := env.FaasEnv.AsyncSharedLogRead(env.FaasCtx, future.GetMeta())
		CHECK(err)

		logState := ResolveLog(env, lastStepLog)
		if logState.State == LogEntryState_PENDING {
			panic("impossible")
		} else if logState.State == LogEntryState_DISCARDED {
			if output, instanceId, ok := fnGetLoggedStepResult(intentLog); ok {
				retValChan <- invokeRetValPair{output, instanceId}
			}
		} else if logState.State == LogEntryState_APPLIED {
			// carry on
		} else {
			panic("unreachable")
		}
	}(env, stepFuture)

	go func(env *Env, future types.Future[uint64]) {
		// newLog would:
		// exert if cond is true
		// drop if cond is false
		instanceId := intentLog.Data["instanceId"].(string)
		stepNumber := intentLog.StepNumber
		input := intentLog.Data["input"]

		asyncLogCtxData, err := env.FaasEnv.AsyncLogCtx().Serialize()
		CHECK(err)
		futureMetaData, err := future.GetMeta().Serialize()
		CHECK(err)

		iw := InputWrapper{
			CallerName:  env.LambdaId,
			CallerId:    env.InstanceId,
			CallerStep:  stepNumber,
			Async:       false,
			InstanceId:  instanceId,
			Input:       input,
			TxnId:       env.TxnId,
			Instruction: env.Instruction,

			AsyncLogPropagator:        string(asyncLogCtxData),
			LastStepLogMetaPropagator: string(futureMetaData),
		}
		if iw.Instruction == "EXECUTE" {
			env.FaasEnv.AsyncLogCtx().Chain(LibAsyncAppendLog(env, TransactionStreamTag(env.LambdaId, env.TxnId),
				[]types.TagMeta{
					{
						FsmType: FsmType_TRANSACTIONSTREAM,
						TagKeys: []string{env.LambdaId, env.TxnId},
					},
				},
				&TxnLogEntry{
					LambdaId: env.LambdaId,
					TxnId:    env.TxnId,
					Callee:   callee,
					WriteOp:  aws.JSONValue{},
				}, func(cond types.CondHandle) { cond.AddDep(future.GetMeta()) }).GetMeta())
		}
		payload := iw.Serialize()
		res, err := env.FaasEnv.InvokeFunc(env.FaasCtx, callee, payload)
		CHECK(err)
		ow := OutputWrapper{}
		ow.Deserialize(res)
		switch ow.Status {
		case "Success":
			retValChan <- invokeRetValPair{ow.Output, iw.InstanceId}
		default:
			panic("never happens")
		}
	}(env, stepFuture)

	retVal := <-retValChan
	return retVal.output, retVal.instanceId
}

// func AssignedSyncInvoke(env *Env, callee string, input interface{}, preInvokeLog *IntentLogEntry) (interface{}, string) {
// 	CheckLogDataField(preInvokeLog, "type", "PreInvoke")
// 	CheckLogDataField(preInvokeLog, "callee", callee)

// 	instanceId := preInvokeLog.Data["instanceId"].(string)

// 	resultLog := FetchStepResultLog(env, preInvokeLog.StepNumber /* catch= */, false)
// 	if resultLog != nil {
// 		CheckLogDataField(resultLog, "type", "InvokeResult")
// 		log.Printf("[INFO] Seen InvokeResult log for step %d", preInvokeLog.StepNumber)
// 		return resultLog.Data["output"], instanceId
// 	}

// 	iw := InputWrapper{
// 		CallerName:  env.LambdaId,
// 		CallerId:    env.InstanceId,
// 		CallerStep:  preInvokeLog.StepNumber,
// 		Async:       false,
// 		InstanceId:  instanceId,
// 		Input:       input,
// 		TxnId:       env.TxnId,
// 		Instruction: env.Instruction,
// 	}
// 	if iw.Instruction == "EXECUTE" {
// 		LibAppendLog(env, TransactionStreamTag(env.LambdaId, env.TxnId), &TxnLogEntry{
// 			LambdaId: env.LambdaId,
// 			TxnId:    env.TxnId,
// 			Callee:   callee,
// 			WriteOp:  aws.JSONValue{},
// 		})
// 	}
// 	payload := iw.Serialize()
// 	res, err := env.FaasEnv.InvokeFunc(env.FaasCtx, callee, payload)
// 	CHECK(err)
// 	ow := OutputWrapper{}
// 	ow.Deserialize(res)
// 	switch ow.Status {
// 	case "Success":
// 		return ow.Output, iw.InstanceId
// 	default:
// 		panic("never happens")
// 	}
// }

func AsyncInvoke(env *Env, callee string, input interface{}) string {
	stepFuture, intentLog := AsyncProposeNextStep(env, aws.JSONValue{
		"type":       "PreInvoke",
		"instanceId": shortuuid.New(),
		"callee":     callee,
		"input":      input,
	}, /*deps*/ []types.FutureMeta{})
	// if stepFuture is nil, then intentLog is the recorded step before;
	// if stepFuture is not nil, then intentLog is the newly append step;
	fnGetLoggedStepResult := func(preInvokeLog *IntentLogEntry) (interface{}, string, bool) {
		instanceId := preInvokeLog.Data["instanceId"].(string)

		CheckLogDataField(preInvokeLog, "type", "PreInvoke")
		log.Printf("[INFO] Seen PreInvoke log for step %d", preInvokeLog.StepNumber)
		resultLog := FetchStepResultLog(env, preInvokeLog.StepNumber /* catch= */, false)
		if resultLog != nil {
			CheckLogDataField(resultLog, "type", "InvokeResult")
			log.Printf("[INFO] Seen InvokeResult log for step %d", preInvokeLog.StepNumber)
			return resultLog.Data["output"], instanceId, true
		} else {
			return nil, "", false
		}
	}
	if stepFuture != nil {
		env.FaasEnv.AsyncLogCtx().Chain(stepFuture.GetMeta())
	} else {
		if _, instanceId, ok := fnGetLoggedStepResult(intentLog); ok {
			return instanceId
		} else {
			panic("unreachable")
		}
	}
	// However the fsm maybe expired, so we still need to resolve it
	// to grab the last view of the step log here.
	retValChan := make(chan invokeRetValPair)
	// DEBUG
	// go func(env *Env, future types.Future[uint64]) {
	// 	// sync
	// 	err := future.Await(10 * time.Second)
	// 	CHECK(err)
	// 	// resolve cond
	// 	lastStepLog, err := env.FaasEnv.AsyncSharedLogRead(env.FaasCtx, future.GetMeta())
	// 	CHECK(err)

	// 	logState := ResolveLog(env, lastStepLog)
	// 	if logState.State == LogEntryState_PENDING {
	// 		panic("impossible")
	// 	} else if logState.State == LogEntryState_DISCARDED {
	// 		if _, instanceId, ok := fnGetLoggedStepResult(intentLog); ok {
	// 			retValChan <- invokeRetValPair{nil, instanceId}
	// 		}
	// 	} else if logState.State == LogEntryState_APPLIED {
	// 		// carry on
	// 	} else {
	// 		panic("unreachable")
	// 	}
	// }(env, stepFuture)

	go func(env *Env, future types.Future[uint64]) {
		// newLog would:
		// exert if cond is true
		// drop if cond is false
		instanceId := intentLog.Data["instanceId"].(string)
		stepNumber := intentLog.StepNumber

		asyncLogCtxData, err := env.FaasEnv.AsyncLogCtx().Serialize()
		CHECK(err)
		futureMetaData, err := future.GetMeta().Serialize()
		CHECK(err)

		iw := InputWrapper{
			CallerName: env.LambdaId,
			CallerId:   env.InstanceId,
			CallerStep: stepNumber,
			Async:      false,
			InstanceId: instanceId,
			Input:      input,

			AsyncLogPropagator:        string(asyncLogCtxData),
			LastStepLogMetaPropagator: string(futureMetaData),
		}
		payload := iw.Serialize()
		err = env.FaasEnv.InvokeFuncAsync(env.FaasCtx, callee, payload)
		CHECK(err)
		retValChan <- invokeRetValPair{nil, iw.InstanceId}
	}(env, stepFuture)

	retVal := <-retValChan
	return retVal.instanceId
}

func getAllTxnLogs(env *Env) []*TxnLogEntry {
	tag := TransactionStreamTag(env.LambdaId, env.TxnId)
	seqNum := uint64(0)
	results := make([]*TxnLogEntry, 0)
	for {
		logEntry, err := env.FaasEnv.AsyncSharedLogReadNext(env.FaasCtx, tag, seqNum)
		CHECK(err)
		if logEntry == nil {
			break
		}
		decoded, err := snappy.Decode(nil, logEntry.Data)
		CHECK(err)
		var txnLog TxnLogEntry
		err = json.Unmarshal(decoded, &txnLog)
		CHECK(err)
		if txnLog.LambdaId == env.LambdaId && txnLog.TxnId == env.TxnId {
			txnLog.SeqNum = logEntry.SeqNum
			results = append(results, &txnLog)
		}
		seqNum = logEntry.SeqNum + 1
	}
	return results
}

func TPLCommit(env *Env) {
	txnLogs := getAllTxnLogs(env)
	for _, txnLog := range txnLogs {
		if txnLog.Callee != "" {
			continue
		}
		tablename := txnLog.WriteOp["tablename"].(string)
		key := txnLog.WriteOp["key"].(string)
		update := map[expression.NameBuilder]expression.OperandBuilder{}
		for kk, vv := range txnLog.WriteOp["value"].(map[string]interface{}) {
			update[expression.Name(kk)] = expression.Value(vv)
		}
		Write(env, tablename, key, update)
		Unlock(env, tablename, key)
	}
	for _, txnLog := range txnLogs {
		if txnLog.Callee != "" {
			log.Printf("[INFO] Commit transaction %s for callee %s", env.TxnId, txnLog.Callee)
			SyncInvoke(env, txnLog.Callee, aws.JSONValue{})
		}
	}
}

func TPLAbort(env *Env) {
	txnLogs := getAllTxnLogs(env)
	for _, txnLog := range txnLogs {
		if txnLog.Callee != "" {
			continue
		}
		tablename := txnLog.WriteOp["tablename"].(string)
		key := txnLog.WriteOp["key"].(string)
		Unlock(env, tablename, key)
	}
	for _, txnLog := range txnLogs {
		if txnLog.Callee != "" {
			log.Printf("[INFO] Abort transaction %s for callee %s", env.TxnId, txnLog.Callee)
			SyncInvoke(env, txnLog.Callee, aws.JSONValue{})
		}
	}
}

func wrapperInternal(f func(*Env) interface{}, iw *InputWrapper, env *Env) (OutputWrapper, error) {
	if TYPE == "BASELINE" {
		panic("Baseline type not supported")
	}

	var intentCond func(types.CondHandle)
	if iw.CallerName != "" {
		intentCond = func(cond types.CondHandle) {
			cond.AddDep(iw.lastStepLogMeta)
		}
	} else {
		intentCond = func(cond types.CondHandle) {}
	}

	var intentLogFuture types.Future[uint64]
	if iw.Async == false || iw.CallerName == "" {
		intentLogFuture = LibAsyncAppendLog(env, IntentLogTag, IntentLogTagMeta(), aws.JSONValue{
			"InstanceId": env.InstanceId,
			"DONE":       false,
			"ASYNC":      iw.Async,
			"INPUT":      iw.Input,
			"ST":         time.Now().Unix(),
		}, intentCond)
	} else {
		intentLogFuture = LibAsyncAppendLog(env, IntentLogTag, IntentLogTagMeta(), aws.JSONValue{
			"InstanceId": env.InstanceId,
			"ST":         time.Now().Unix(),
		}, intentCond)
	}
	//ok := LibPut(env.IntentTable, aws.JSONValue{"InstanceId": env.InstanceId},
	//	aws.JSONValue{"DONE": false, "ASYNC": iw.Async})
	//if !ok {
	//	res := LibRead(env.IntentTable, aws.JSONValue{"InstanceId": env.InstanceId}, []string{"RET"})
	//	output, exist := res["RET"]
	//	if exist {
	//		return OutputWrapper{
	//			Status: "Success",
	//			Output: output,
	//		}, nil
	//	}
	//}

	if err := env.FaasEnv.AsyncLogCtx().Sync(time.Second); err != nil {
		return OutputWrapper{
			Status: "Failure",
			Output: 0,
		}, err
	}
	env.FaasEnv.AsyncLogCtx().Truncate()
	// speed up by skipping intent log sync
	env.FaasEnv.AsyncLogCtx().Chain(intentLogFuture.GetMeta())

	// verify
	if iw.CallerName != "" {
		lastStepLog, err := env.FaasEnv.AsyncSharedLogRead(env.FaasCtx, iw.lastStepLogMeta)
		CHECK(err)
		// DEBUG
		if lastStepLog == nil {
			panic(fmt.Sprintf("read nothing from last step: %+v", iw.lastStepLogMeta))
		}

		logState := ResolveLog(env, lastStepLog)
		if logState.State == LogEntryState_PENDING {
			panic("impossible")
		} else if logState.State == LogEntryState_DISCARDED {
			return OutputWrapper{
				Status: "Failure",
				Output: 0,
			}, nil
		} else if logState.State == LogEntryState_APPLIED {
			// carry on
		} else {
			panic("unreachable")
		}
	}

	var output interface{}
	if env.Instruction == "COMMIT" {
		TPLCommit(env)
		output = 0
	} else if env.Instruction == "ABORT" {
		TPLAbort(env)
		output = 0
	} else {
		output = f(env)
	}

	if iw.CallerName != "" {
		env.FaasEnv.AsyncLogCtx().Chain(AsyncLogStepResult(env, iw.CallerId, iw.CallerStep, aws.JSONValue{
			"type":   "InvokeResult",
			"output": output,
		}, func(cond types.CondHandle) {
			cond.AddDep(iw.lastStepLogMeta)
		}).GetMeta())
	}
	env.FaasEnv.AsyncLogCtx().Chain(LibAsyncAppendLog(env, IntentLogTag, IntentLogTagMeta(), aws.JSONValue{
		"InstanceId": env.InstanceId,
		"DONE":       true,
		"TS":         time.Now().Unix(),
	}, func(cond types.CondHandle) {
		cond.AddDep(intentLogFuture.GetMeta())
	}).GetMeta())

	return OutputWrapper{
		Status: "Success",
		Output: output,
	}, nil
}

type funcHandlerWrapper struct {
	fnName  string
	handler func(env *Env) interface{}
	env     types.Environment
}

func (w *funcHandlerWrapper) Call(ctx context.Context, input []byte) ([]byte, error) {
	var jsonInput map[string]interface{}
	err := json.Unmarshal(input, &jsonInput)
	if err != nil {
		return nil, err
	}
	iw := ParseInput(jsonInput)
	env := PrepareEnv(iw, w.fnName)
	env.FaasCtx = ctx
	env.FaasEnv = w.env

	if iw.CallerName != "" {
		err = env.FaasEnv.NewAsyncLogCtx([]byte(iw.AsyncLogPropagator))
		CHECK(err)
		iw.lastStepLogMeta, err = types.DeserializeFutureMeta([]byte(iw.LastStepLogMetaPropagator))
		CHECK(err)
	}

	env.Fsm = NewIntentFsm(env.InstanceId)
	env.Fsm.Catch(env)
	ow, err := wrapperInternal(w.handler, iw, env)
	if err != nil {
		return nil, err
	}
	return ow.Serialize(), nil
}

type funcHandlerFactory struct {
	handler func(env *Env) interface{}
}

func (f *funcHandlerFactory) New(env types.Environment, funcName string) (types.FuncHandler, error) {
	return &funcHandlerWrapper{
		fnName:  funcName,
		handler: f.handler,
		env:     env,
	}, nil
}

func (f *funcHandlerFactory) GrpcNew(env types.Environment, service string) (types.GrpcFuncHandler, error) {
	return nil, fmt.Errorf("Not implemented")
}

func CreateFuncHandlerFactory(f func(env *Env) interface{}) types.FuncHandlerFactory {
	return &funcHandlerFactory{handler: f}
}
