package microbenchmark

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	statestore "cs.utexas.edu/zjia/faas/slib/asyncstatestore"
	"cs.utexas.edu/zjia/faas/types"
)

type SingleOpInput struct {
	OpType string `json:"optype"`
}

type SingleOpOutput struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	ResultVar string `json:"resvar"`
}

type singleOpHandler struct {
	env types.Environment
}

func NewSlibSingleOpHandler(env types.Environment) types.FuncHandler {
	return &singleOpHandler{
		env: env,
	}
}

func singleOpReadSlib(ctx context.Context, env types.Environment, input *SingleOpInput) (*SingleOpOutput, error) {
	store := statestore.CreateEnv(ctx, env)
	timelineObj := store.Object("tempvar")
	if value, _ := timelineObj.Get("value"); !value.IsNull() {
		resultNum := value.AsNumber()
		return &SingleOpOutput{
			Success:   true,
			ResultVar: fmt.Sprint(resultNum),
		}, nil
	} else {
		return &SingleOpOutput{
			Success: false,
			Message: "Got null",
		}, nil
	}
}

func singleOpWriteSlib(ctx context.Context, env types.Environment, input *SingleOpInput) (*SingleOpOutput, error) {
	store := statestore.CreateEnv(ctx, env)
	value := float64(rand.Intn(100))
	if result := store.Object("tempvar").SetNumber("value", value); result.Err != nil {
		return &SingleOpOutput{
			Success: false,
			Message: result.Err.Error(),
		}, nil
	} else {
		return &SingleOpOutput{
			Success:   true,
			ResultVar: fmt.Sprint(value),
		}, nil
	}
}

func (h *singleOpHandler) onRequest(ctx context.Context, input *SingleOpInput) (*SingleOpOutput, error) {
	switch input.OpType {
	case "read":
		return singleOpReadSlib(ctx, h.env, input)
	case "write":
		return singleOpWriteSlib(ctx, h.env, input)
	default:
		panic(fmt.Sprintf("Unknown op type: %s", input.OpType))
	}
}

func (h *singleOpHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &SingleOpInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	output, err := h.onRequest(ctx, parsedInput)
	if err != nil {
		return nil, err
	}
	return json.Marshal(output)
}
