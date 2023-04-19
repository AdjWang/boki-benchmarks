package cayonlib

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/pkg/errors"

	// "github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws"

	// lambdaSdk "github.com/aws/aws-sdk-go/service/lambda"

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
	AsyncLogCtxPropagator string `mapstructure:"AsyncLogCtxPropagator"`
}

func (iw *InputWrapper) Serialize() []byte {
	stream, err := json.Marshal(*iw)
	CHECK(err)
	return stream
}

// func (iw *InputWrapper) Deserialize(stream []byte) {
// 	err := json.Unmarshal(stream, iw)
// 	CHECK(err)
// }

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

func PrepareEnv(ctx context.Context, iw *InputWrapper, lambdaId string, faasEnv types.Environment) *Env {
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

		FaasCtx: ctx,
		FaasEnv: faasEnv,
		FsmHub:  nil,

		AsyncLogCtx: nil,
	}
	env.FsmHub = NewFsmHub(env)
	if iw.CallerName != "" {
		asyncLogCtx, err := DeserializeAsyncLogContext(env.FaasEnv, []byte(iw.AsyncLogCtxPropagator))
		CHECK(err)
		env.AsyncLogCtx = asyncLogCtx
	} else {
		env.AsyncLogCtx = NewAsyncLogContext(env.FaasEnv)
	}

	return env
}

func SyncInvoke(env *Env, callee string, input interface{}) (interface{}, string) {
	stepFuture, intentLog := AsyncProposeNextStep(env, aws.JSONValue{
		"type":       "PreInvoke",
		"instanceId": shortuuid.New(),
		"callee":     callee,
		"input":      input,
	}, env.AsyncLogCtx.GetLastStepLogMeta())
	// if stepFuture is nil, then intentLog is the recorded step before;
	// if stepFuture is not nil, then intentLog is the newly append step;
	if stepFuture != nil {
		env.AsyncLogCtx.ChainStep(stepFuture.GetMeta())
	} else {
		preInvokeLog := intentLog
		instanceId := preInvokeLog.Data["instanceId"].(string)
		CheckLogDataField(preInvokeLog, "type", "PreInvoke")
		log.Printf("[INFO] Seen PreInvoke log for step %d", preInvokeLog.StepNumber)
		resultLog := FetchStepResultLog(env, preInvokeLog.StepNumber /* catch= */, false)
		if resultLog != nil {
			CheckLogDataField(resultLog, "type", "InvokeResult")
			log.Printf("[INFO] Seen InvokeResult log for step %d", preInvokeLog.StepNumber)
			return resultLog.Data["output"], instanceId
		} else {
			panic("unreachable")
		}
	}
	// However the fsm maybe expired, causing the local step check to be passed
	// while the step is still duplicated. The next check is delegated to the
	// callee. The correctness is gauranteed by (Catch(env) and
	// read-your-write consistency and first-write-win step assertation rule).

	// A concurrent step check can be placed here to speed up failing execute
	// flow, but would introcude many tricky racing problems difficult to tackle with.
	// Since the failure step log is believed to be at low ratio, so this optimization
	// is trivial and thus deprecated.

	// the new step log would:
	// applied if cond is true
	// discard if cond is false
	instanceId := intentLog.Data["instanceId"].(string)
	stepNumber := intentLog.StepNumber

	iw := InputWrapper{
		CallerName:  env.LambdaId,
		CallerId:    env.InstanceId,
		CallerStep:  stepNumber,
		Async:       false,
		InstanceId:  instanceId,
		Input:       input,
		TxnId:       env.TxnId,
		Instruction: env.Instruction,

		AsyncLogCtxPropagator: "", // assign later
	}
	if iw.Instruction == "EXECUTE" {
		env.AsyncLogCtx.ChainStep(LibAsyncAppendLog(env, TransactionStreamTag(env.LambdaId, env.TxnId),
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
			},
			func(cond types.CondHandle) {
				cond.AddDep(env.AsyncLogCtx.GetLastStepLogMeta())
			}).GetMeta())
	}
	asyncLogCtxData, err := env.AsyncLogCtx.Serialize()
	CHECK(err)
	iw.AsyncLogCtxPropagator = string(asyncLogCtxData)
	ASSERT(iw.AsyncLogCtxPropagator != "", fmt.Sprintf("invalid AsyncLogCtxPropagator from: %+v", env.AsyncLogCtx))

	payload := iw.Serialize()
	res, err := env.FaasEnv.InvokeFunc(env.FaasCtx, callee, payload)
	CHECK(errors.Wrapf(err, "InvokeFunc callee: %v, res: %v, payload: %+v", callee, res, iw))
	ow := OutputWrapper{}
	ow.Deserialize(res)
	switch ow.Status {
	case "Success":
		return ow.Output, iw.InstanceId
	default:
		panic("never happens")
	}
}

func ProposeInvoke(env *Env, callee string, input interface{}) (types.Future[uint64], *IntentLogEntry) {
	stepFuture, intentLog := AsyncProposeNextStep(env, aws.JSONValue{
		"type":       "PreInvoke",
		"instanceId": shortuuid.New(),
		"callee":     callee,
		"input":      input,
	}, env.AsyncLogCtx.GetLastStepLogMeta())
	if stepFuture != nil {
		env.AsyncLogCtx.ChainStep(stepFuture.GetMeta())
	}
	return stepFuture, intentLog
}
func AssignedSyncInvoke(env *Env, callee string, stepFuture types.Future[uint64], intentLog *IntentLogEntry) (interface{}, string) {
	if stepFuture == nil {
		preInvokeLog := intentLog
		instanceId := preInvokeLog.Data["instanceId"].(string)
		CheckLogDataField(preInvokeLog, "type", "PreInvoke")
		log.Printf("[INFO] Seen PreInvoke log for step %d", preInvokeLog.StepNumber)
		resultLog := FetchStepResultLog(env, preInvokeLog.StepNumber /* catch= */, false)
		if resultLog != nil {
			CheckLogDataField(resultLog, "type", "InvokeResult")
			log.Printf("[INFO] Seen InvokeResult log for step %d", preInvokeLog.StepNumber)
			return resultLog.Data["output"], instanceId
		} else {
			panic("unreachable")
		}
	}

	instanceId := intentLog.Data["instanceId"].(string)
	stepNumber := intentLog.StepNumber
	input := intentLog.Data["input"]

	iw := InputWrapper{
		CallerName:  env.LambdaId,
		CallerId:    env.InstanceId,
		CallerStep:  stepNumber,
		Async:       false,
		InstanceId:  instanceId,
		Input:       input,
		TxnId:       env.TxnId,
		Instruction: env.Instruction,

		AsyncLogCtxPropagator: "", // assign later
	}
	if iw.Instruction == "EXECUTE" {
		env.AsyncLogCtx.ChainStep(LibAsyncAppendLog(env, TransactionStreamTag(env.LambdaId, env.TxnId),
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
			},
			func(cond types.CondHandle) {
				cond.AddDep(env.AsyncLogCtx.GetLastStepLogMeta())
			}).GetMeta())
	}
	asyncLogCtxData, err := env.AsyncLogCtx.Serialize()
	CHECK(err)
	iw.AsyncLogCtxPropagator = string(asyncLogCtxData)
	ASSERT(iw.AsyncLogCtxPropagator != "", fmt.Sprintf("invalid AsyncLogCtxPropagator from: %+v", env.AsyncLogCtx))

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
	stepFuture, intentLog := AsyncProposeNextStep(env, aws.JSONValue{
		"type":       "PreInvoke",
		"instanceId": shortuuid.New(),
		"callee":     callee,
		"input":      input,
	}, env.AsyncLogCtx.GetLastStepLogMeta())
	if stepFuture != nil {
		env.AsyncLogCtx.ChainStep(stepFuture.GetMeta())
	} else {
		preInvokeLog := intentLog
		instanceId := preInvokeLog.Data["instanceId"].(string)
		CheckLogDataField(preInvokeLog, "type", "PreInvoke")
		log.Printf("[INFO] Seen PreInvoke log for step %d", preInvokeLog.StepNumber)
		resultLog := FetchStepResultLog(env, preInvokeLog.StepNumber /* catch= */, false)
		if resultLog != nil {
			CheckLogDataField(resultLog, "type", "InvokeResult")
			log.Printf("[INFO] Seen InvokeResult log for step %d", preInvokeLog.StepNumber)
			return instanceId
		} else {
			panic("unreachable")
		}
	}

	instanceId := intentLog.Data["instanceId"].(string)
	stepNumber := intentLog.StepNumber

	asyncLogCtxData, err := env.AsyncLogCtx.Serialize()
	CHECK(err)

	iw := InputWrapper{
		CallerName: env.LambdaId,
		CallerId:   env.InstanceId,
		CallerStep: stepNumber,
		Async:      true,
		InstanceId: instanceId,
		Input:      input,

		AsyncLogCtxPropagator: string(asyncLogCtxData),
	}
	payload := iw.Serialize()
	err = env.FaasEnv.InvokeFuncAsync(env.FaasCtx, callee, payload)
	CHECK(err)
	return iw.InstanceId
}

func wrapperInternal(f func(*Env) interface{}, iw *InputWrapper, env *Env) (OutputWrapper, error) {
	if TYPE == "BASELINE" {
		panic("Baseline type not supported")
	}

	if iw.CallerName != "" {
		ASSERT(env.AsyncLogCtx.GetLastStepLogMeta().IsValid(),
			fmt.Sprintf("last step meta: %+v should be valid", env.AsyncLogCtx.GetLastStepLogMeta()))
	} else {
		ASSERT(!env.AsyncLogCtx.GetLastStepLogMeta().IsValid(),
			fmt.Sprintf("last step meta: %+v should be invalid", env.AsyncLogCtx.GetLastStepLogMeta()))
	}

	var intentLogFuture types.Future[uint64]
	if iw.Async == false || iw.CallerName == "" {
		intentLogFuture = LibAsyncAppendLog(env, IntentLogTag, IntentLogTagMeta(), aws.JSONValue{
			"InstanceId": env.InstanceId,
			"DONE":       false,
			"ASYNC":      iw.Async,
			"INPUT":      iw.Input,
			"ST":         time.Now().Unix(),
		}, func(cond types.CondHandle) {
			cond.AddDep(env.AsyncLogCtx.GetLastStepLogMeta())
		})
	} else {
		intentLogFuture = LibAsyncAppendLog(env, IntentLogTag, IntentLogTagMeta(), aws.JSONValue{
			"InstanceId": env.InstanceId,
			"ST":         time.Now().Unix(),
		}, func(cond types.CondHandle) {
			cond.AddDep(env.AsyncLogCtx.GetLastStepLogMeta())
		})
	}
	env.AsyncLogCtx.ChainFuture(intentLogFuture.GetMeta())
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
	// assert true only for bokiflow that a user function must have steps
	ASSERT(env.AsyncLogCtx.GetLastStepLogMeta().IsValid(),
		fmt.Sprintf("last step meta: %+v should be valid", env.AsyncLogCtx.GetLastStepLogMeta()))

	if iw.CallerName != "" {
		env.AsyncLogCtx.ChainStep(AsyncLogStepResult(env, iw.CallerId, iw.CallerStep, aws.JSONValue{
			"type":   "InvokeResult",
			"output": output,
		}, env.AsyncLogCtx.GetLastStepLogMeta()).GetMeta())
	}
	env.AsyncLogCtx.ChainFuture(LibAsyncAppendLog(env, IntentLogTag, IntentLogTagMeta(), aws.JSONValue{
		"InstanceId": env.InstanceId,
		"DONE":       true,
		"TS":         time.Now().Unix(),
	}, func(cond types.CondHandle) {
		cond.AddDep(env.AsyncLogCtx.GetLastStepLogMeta())
	}).GetMeta())

	// clear pending logs at the end of the workflow
	if iw.CallerName == "" {
		err := env.AsyncLogCtx.Sync(gSyncTimeout)
		CHECK(err)
	}

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
	env := PrepareEnv(ctx, iw, w.fnName, w.env)
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
