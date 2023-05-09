package cayonlib

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

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

	LogTracerPropagator string `mapstructure:"LogTracerPropagator"`
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
	Status     string
	Output     interface{}
	TraceLog   string
	TraceTotal string
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
		CHECK(json.Unmarshal([]byte(body.(string)), &iw))
	} else {
		CHECK(mapstructure.Decode(raw, &iw))
	}
	return &iw
}

func PrepareEnv(iw *InputWrapper, lambdaId string) *Env {
	// s := strings.Split(lambdacontext.FunctionName, "-")
	// lambdaId := s[len(s)-1]
	if iw.InstanceId == "" {
		iw.InstanceId = shortuuid.New()
	}
	env := &Env{
		LambdaId:    lambdaId,
		InstanceId:  iw.InstanceId,
		StepNumber:  0,
		Input:       iw.Input,
		TxnId:       iw.TxnId,
		Instruction: iw.Instruction,

		LogTracer: nil,
	}
	if iw.CallerName != "" {
		logTracer, err := DeserializeLogTracer([]byte(iw.LogTracerPropagator))
		CHECK(err)
		env.LogTracer = logTracer
	} else {
		env.LogTracer = NewLogTracer()
	}
	return env
}

func SyncInvoke(env *Env, callee string, input interface{}) (interface{}, string) {
	env.LogTracer.TraceStart()
	newLog, preInvokeLog := ProposeNextStep(env, aws.JSONValue{
		"type":       "PreInvoke",
		"instanceId": shortuuid.New(),
		"callee":     callee,
		"input":      input,
	})
	instanceId := preInvokeLog.Data["instanceId"].(string)
	if !newLog {
		CheckLogDataField(preInvokeLog, "type", "PreInvoke")
		log.Printf("[INFO] Seen PreInvoke log for step %d", preInvokeLog.StepNumber)
		resultLog := FetchStepResultLog(env, preInvokeLog.StepNumber /* catch= */, false)
		if resultLog != nil {
			CheckLogDataField(resultLog, "type", "InvokeResult")
			log.Printf("[INFO] Seen InvokeResult log for step %d", preInvokeLog.StepNumber)
			env.LogTracer.TraceEnd()
			return resultLog.Data["output"], instanceId
		}
	}
	env.LogTracer.TraceEnd()

	iw := InputWrapper{
		CallerName:  env.LambdaId,
		CallerId:    env.InstanceId,
		CallerStep:  preInvokeLog.StepNumber,
		Async:       false,
		InstanceId:  instanceId,
		Input:       input,
		TxnId:       env.TxnId,
		Instruction: env.Instruction,

		LogTracerPropagator: "", // assign later
	}
	env.LogTracer.TraceStart()
	if iw.Instruction == "EXECUTE" {
		LibAppendLog(env, TransactionStreamTag(env.LambdaId, env.TxnId), &TxnLogEntry{
			LambdaId: env.LambdaId,
			TxnId:    env.TxnId,
			Callee:   callee,
			WriteOp:  aws.JSONValue{},
		})
	}
	env.LogTracer.TraceEnd()
	logTracerData, err := env.LogTracer.Serialize()
	CHECK(err)
	iw.LogTracerPropagator = string(logTracerData)
	ASSERT(iw.LogTracerPropagator != "", fmt.Sprintf("invalid LogTracerPropagator from: %+v", env.LogTracer))
	log.Printf("[DEBUG] sync invoke: %v %v", iw.LogTracerPropagator, env.LogTracer.DebugString())

	payload := iw.Serialize()
	res, err := env.FaasEnv.InvokeFunc(env.FaasCtx, callee, payload)
	CHECK(err)
	ow := OutputWrapper{}
	ow.Deserialize(res)
	switch ow.Status {
	case "Success":
		tLog, err := strconv.ParseInt(ow.TraceLog, 10, 64)
		CHECK(err)
		env.LogTracer.TraceAdd(time.Duration(tLog) * time.Microsecond)
		tTotal, err := strconv.ParseInt(ow.TraceTotal, 10, 64)
		CHECK(err)
		env.LogTracer.TraceAddTotal(time.Duration(tTotal) * time.Microsecond)
		return ow.Output, iw.InstanceId
	default:
		panic("never happens")
	}
}

func ProposeInvoke(env *Env, callee string) *IntentLogEntry {
	env.LogTracer.TraceStart()
	defer env.LogTracer.TraceEnd()

	newLog, preInvokeLog := ProposeNextStep(env, aws.JSONValue{
		"type":       "PreInvoke",
		"instanceId": shortuuid.New(),
		"callee":     callee,
	})
	if !newLog {
		CheckLogDataField(preInvokeLog, "type", "PreInvoke")
		CheckLogDataField(preInvokeLog, "callee", callee)
		log.Printf("[INFO] Seen PreInvoke log for step %d", preInvokeLog.StepNumber)
	}
	return preInvokeLog
}

func AssignedSyncInvoke(env *Env, callee string, input interface{}, preInvokeLog *IntentLogEntry) (interface{}, string) {
	env.LogTracer.TraceStart()
	CheckLogDataField(preInvokeLog, "type", "PreInvoke")
	CheckLogDataField(preInvokeLog, "callee", callee)

	instanceId := preInvokeLog.Data["instanceId"].(string)

	resultLog := FetchStepResultLog(env, preInvokeLog.StepNumber /* catch= */, false)
	if resultLog != nil {
		CheckLogDataField(resultLog, "type", "InvokeResult")
		log.Printf("[INFO] Seen InvokeResult log for step %d", preInvokeLog.StepNumber)
		env.LogTracer.TraceEnd()
		return resultLog.Data["output"], instanceId
	}

	iw := InputWrapper{
		CallerName:  env.LambdaId,
		CallerId:    env.InstanceId,
		CallerStep:  preInvokeLog.StepNumber,
		Async:       false,
		InstanceId:  instanceId,
		Input:       input,
		TxnId:       env.TxnId,
		Instruction: env.Instruction,

		LogTracerPropagator: "", // assign later
	}
	if iw.Instruction == "EXECUTE" {
		LibAppendLog(env, TransactionStreamTag(env.LambdaId, env.TxnId), &TxnLogEntry{
			LambdaId: env.LambdaId,
			TxnId:    env.TxnId,
			Callee:   callee,
			WriteOp:  aws.JSONValue{},
		})
	}
	env.LogTracer.TraceEnd()
	logTracerData, err := env.LogTracer.Serialize()
	CHECK(err)
	iw.LogTracerPropagator = string(logTracerData)
	ASSERT(iw.LogTracerPropagator != "", fmt.Sprintf("invalid LogTracerPropagator from: %+v", env.LogTracer))

	payload := iw.Serialize()
	res, err := env.FaasEnv.InvokeFunc(env.FaasCtx, callee, payload)
	CHECK(err)
	ow := OutputWrapper{}
	ow.Deserialize(res)
	switch ow.Status {
	case "Success":
		return ow.Output, iw.InstanceId
	default:
		panic("never happens")
	}
}

func AsyncInvoke(env *Env, callee string, input interface{}) string {
	env.LogTracer.TraceStart()
	newLog, preInvokeLog := ProposeNextStep(env, aws.JSONValue{
		"type":       "PreInvoke",
		"instanceId": shortuuid.New(),
		"callee":     callee,
		"input":      input,
	})
	instanceId := preInvokeLog.Data["instanceId"].(string)
	if !newLog {
		CheckLogDataField(preInvokeLog, "type", "PreInvoke")
		log.Printf("[INFO] Seen PreInvoke log for step %d", preInvokeLog.StepNumber)
		resultLog := FetchStepResultLog(env, preInvokeLog.StepNumber /* catch= */, false)
		if resultLog != nil {
			CheckLogDataField(resultLog, "type", "InvokeResult")
			log.Printf("[INFO] Seen InvokeResult log for step %d", preInvokeLog.StepNumber)
			env.LogTracer.TraceEnd()
			return instanceId
		}
	}
	env.LogTracer.TraceEnd()

	logTracerData, err := env.LogTracer.Serialize()
	CHECK(err)
	iw := InputWrapper{
		CallerName: env.LambdaId,
		CallerId:   env.InstanceId,
		CallerStep: preInvokeLog.StepNumber,
		Async:      true,
		InstanceId: instanceId,
		Input:      input,

		LogTracerPropagator: string(logTracerData),
	}

	/*
		Should we handle this?
		LibWrite(env.LogTable, pk, map[expression.NameBuilder]expression.OperandBuilder{
			expression.Name("RET"): expression.Value(1),
		})
	*/

	payload := iw.Serialize()
	err = env.FaasEnv.InvokeFuncAsync(env.FaasCtx, callee, payload)
	CHECK(err)
	return iw.InstanceId
}

func getAllTxnLogs(env *Env) []*TxnLogEntry {
	tag := TransactionStreamTag(env.LambdaId, env.TxnId)
	seqNum := uint64(0)
	results := make([]*TxnLogEntry, 0)
	for {
		logEntry, err := env.FaasEnv.SharedLogReadNext(env.FaasCtx, tag, seqNum)
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
	start := time.Now()

	if TYPE == "BASELINE" {
		panic("Baseline type not supported")
	}

	env.LogTracer.TraceStart()
	if iw.Async == false || iw.CallerName == "" {
		LibAppendLog(env, IntentLogTag, aws.JSONValue{
			"InstanceId": env.InstanceId,
			"DONE":       false,
			"ASYNC":      iw.Async,
			"INPUT":      iw.Input,
			"ST":         time.Now().Unix(),
		})
	} else {
		LibAppendLog(env, IntentLogTag, aws.JSONValue{
			"InstanceId": env.InstanceId,
			"ST":         time.Now().Unix(),
		})
	}
	env.LogTracer.TraceEnd()
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

	env.LogTracer.TraceStart()
	if iw.CallerName != "" {
		LogStepResult(env, iw.CallerId, iw.CallerStep, aws.JSONValue{
			"type":   "InvokeResult",
			"output": output,
		})
	}
	LibAppendLog(env, IntentLogTag, aws.JSONValue{
		"InstanceId": env.InstanceId,
		"DONE":       true,
		"TS":         time.Now().Unix(),
	})
	env.LogTracer.TraceEnd()

	elapsed := time.Since(start)
	env.LogTracer.TraceAddTotal(elapsed)
	return OutputWrapper{
		Status:     "Success",
		Output:     output,
		TraceLog:   env.LogTracer.LogString(),
		TraceTotal: env.LogTracer.TotalString(),
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

	env.LogTracer.TraceStart()
	env.Fsm = NewIntentFsm(env.InstanceId)
	env.Fsm.Catch(env)
	env.LogTracer.TraceEnd()

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
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

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
