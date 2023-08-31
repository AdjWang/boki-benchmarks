package microbenchmark

import (
	"context"
	"encoding/json"
	"fmt"

	statestore "cs.utexas.edu/zjia/faas/slib/asyncstatestore"
	"cs.utexas.edu/zjia/faas/types"
)

type SingleOpReadInput struct {
}

type SingleOpReadOutput struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	ResultVar string `json:"resvar"`
}

type singleOpReadHandler struct {
	env types.Environment
}

func NewSlibSingleOpReadHandler(env types.Environment) types.FuncHandler {
	return &singleOpReadHandler{
		env: env,
	}
}

func singleOpReadSlib(ctx context.Context, env types.Environment, input *SingleOpReadInput) (*SingleOpReadOutput, error) {
	store := statestore.CreateEnv(ctx, env)
	timelineObj := store.Object("tempvar")
	if value, _ := timelineObj.Get("value"); !value.IsNull() {
		resultNum := value.AsNumber()
		return &SingleOpReadOutput{
			Success:   true,
			ResultVar: fmt.Sprint(resultNum),
		}, nil
	} else {
		return &SingleOpReadOutput{
			Success: false,
			Message: "Got null",
		}, nil
	}
}

func (h *singleOpReadHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &SingleOpReadInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	output, err := singleOpReadSlib(ctx, h.env, parsedInput)
	if err != nil {
		return nil, err
	}
	return json.Marshal(output)
}
