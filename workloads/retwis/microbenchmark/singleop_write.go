package microbenchmark

import (
	"context"
	"encoding/json"
	"strconv"

	statestore "cs.utexas.edu/zjia/faas/slib/asyncstatestore"
	"cs.utexas.edu/zjia/faas/types"
)

type SingleOpWriteInput struct {
	InputVar string `json:"var"`
}

type SingleOpWriteOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type singleOpWriteHandler struct {
	env types.Environment
}

func NewSlibSingleOpWriteHandler(env types.Environment) types.FuncHandler {
	return &singleOpWriteHandler{
		env: env,
	}
}

func singleOpWriteSlib(ctx context.Context, env types.Environment, input *SingleOpWriteInput) (*SingleOpWriteOutput, error) {
	store := statestore.CreateEnv(ctx, env)
	value, err := strconv.ParseUint(input.InputVar, 10, 64)
	if err != nil {
		return &SingleOpWriteOutput{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	if result := store.Object("tempvar").SetNumber("value", float64(value)); result.Err != nil {
		return &SingleOpWriteOutput{
			Success: false,
			Message: result.Err.Error(),
		}, nil
	} else {
		return &SingleOpWriteOutput{
			Success: true,
		}, nil
	}
}

func (h *singleOpWriteHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &SingleOpWriteInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	output, err := singleOpWriteSlib(ctx, h.env, parsedInput)
	if err != nil {
		return nil, err
	}
	return json.Marshal(output)
}
