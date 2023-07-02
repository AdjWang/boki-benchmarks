package cayonlib

import (
	"encoding/json"
	"testing"
)

func TestInputWrapperSerialize(t *testing.T) {
	dummyAsyncLogCtx := DebugNewAsyncLogContext()
	dummyAsyncLogCtx.ChainStep(0 /*LocalId*/)
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
