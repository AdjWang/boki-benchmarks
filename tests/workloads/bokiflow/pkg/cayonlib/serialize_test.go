package cayonlib

import (
	"encoding/json"
	"testing"

	"cs.utexas.edu/zjia/faas/types"
)

//	type InputWrapper struct {
//		CallerName  string      `mapstructure:"CallerName"`
//		CallerId    string      `mapstructure:"CallerId"`
//		CallerStep  int32       `mapstructure:"CallerStep"`
//		InstanceId  string      `mapstructure:"InstanceId"`
//		Input       interface{} `mapstructure:"Input"`
//		TxnId       string      `mapstructure:"TxnId"`
//		Instruction string      `mapstructure:"Instruction"`
//		Async       bool        `mapstructure:"Async"`
//
//		AsyncLogPropagator interface{}      `mapstructure:"AsyncLogPropagator"`
//		LastStepLogMeta    types.FutureMeta `mapstructure:"LastStepLogMeta"`
//	}
func TestInputWrapperSerialize(t *testing.T) {
	dummyAsyncLogCtx := DebugNewAsyncLogContext()
	dummyFutureMeta := types.FutureMeta{
		LocalId: 0,
	}
	dummyAsyncLogCtx.ChainStep(dummyFutureMeta)
	asyncLogCtxData, err := dummyAsyncLogCtx.Serialize()
	if err != nil {
		t.Fatal(err)
	}

	iw := InputWrapper{
		CallerName:  "dummyCallerName",
		CallerId:    "dummyCallerId",
		CallerStep:  0,
		Async:       false,
		InstanceId:  "dummyInstanceId",
		Input:       nil,
		TxnId:       "dummyTxnId",
		Instruction: "dummyInstruction",

		AsyncLogCtxPropagator: string(asyncLogCtxData),
	}
	t.Logf("iw=%+v", iw)
	input := iw.Serialize()

	{
		// from controlflow.go:func (w *funcHandlerWrapper) Call(ctx context.Context, input []byte) ([]byte, error) {
		var jsonInput map[string]interface{}
		err := json.Unmarshal(input, &jsonInput)
		if err != nil {
			t.Fatal(err)
		}
		iw := ParseInput(jsonInput)
		t.Logf("iw=%+v", iw)
	}
}
