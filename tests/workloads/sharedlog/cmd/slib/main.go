package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"cs.utexas.edu/zjia/faas"
	statestore "cs.utexas.edu/zjia/faas/slib/asyncstatestore"

	// "cs.utexas.edu/zjia/faas/slib/statestore"
	"cs.utexas.edu/zjia/faas/types"
	"github.com/pkg/errors"
)

type statestoreTxnExecHandler struct {
	env types.Environment
}
type statestoreTxnCheckHandler struct {
	env types.Environment
}

type funcHandlerFactory struct {
}

func (f *funcHandlerFactory) New(env types.Environment, funcName string) (types.FuncHandler, error) {
	if funcName == "StatestoreTxnExec" {
		return &statestoreTxnExecHandler{env: env}, nil
	} else if funcName == "StatestoreTxnCheck" {
		return &statestoreTxnCheckHandler{env: env}, nil
	} else {
		return nil, nil
	}
}

func (f *funcHandlerFactory) GrpcNew(env types.Environment, service string) (types.GrpcFuncHandler, error) {
	return nil, fmt.Errorf("not implemented")
}

type TxnExecInput struct {
}

type TxnExecOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func (h *statestoreTxnExecHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &TxnExecInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, errors.Wrapf(err, "json unmarshal failed on %v", string(input))
	}
	output, err := txnExec(ctx, h.env, parsedInput)
	if err != nil {
		return nil, err
	}
	return json.Marshal(output)
}
func txnExec(ctx context.Context, env types.Environment, input *TxnExecInput) (*TxnExecOutput, error) {
	// store := statestore.CreateEnv(ctx, env)
	// nextUserIdObj := store.Object("next_user_id")
	// result := nextUserIdObj.NumberFetchAdd("value", 1)
	// if result.Err != nil {
	// 	return nil, result.Err
	// }
	// userIdValue := uint32(result.Value.AsNumber())
	// return &TxnExecOutput{
	// 	Success: true,
	// 	Message: fmt.Sprintf("userId faa=%d", userIdValue),
	// }, nil

	txn, err := statestore.CreateTxnEnv(ctx, env)
	if err != nil {
		return nil, err
	}
	counterObj := txn.Object("atomic_counter")
	if value, _ := counterObj.Get("count"); !value.IsNull() {
		num, err := strconv.ParseInt(value.AsString(), 16, 64)
		if err != nil {
			txn.TxnAbort()
			return &TxnExecOutput{
				Success: false,
				Message: fmt.Sprintf("Failed to parse fetch-and-add counter due to %v", err),
			}, nil
		}

		num += 1
		counterObj.SetString("count", fmt.Sprintf("%016x", num))

		if committed, err := txn.TxnCommit(); err != nil {
			return nil, err
		} else if committed {
			return &TxnExecOutput{
				Success: true,
				Message: fmt.Sprintf("Success count=%v", num),
			}, nil
		} else {
			return &TxnExecOutput{
				Success: false,
				Message: fmt.Sprintf("Failed to commit fetch-and-add due to conflicts, remaining %v", num-1),
			}, nil
		}
	} else {
		counterObj.SetString("count", fmt.Sprintf("%016x", 1))

		if committed, err := txn.TxnCommit(); err != nil {
			return nil, err
		} else if committed {
			return &TxnExecOutput{
				Success: true,
			}, nil
		} else {
			return &TxnExecOutput{
				Success: false,
				Message: "Failed to commit initial transaction due to conflicts",
			}, nil
		}
	}
}

type TxnCheckInput struct {
	Count int64 `json:"count"`
}

type TxnCheckOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func (h *statestoreTxnCheckHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &TxnCheckInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, errors.Wrapf(err, "json unmarshal failed on %v", string(input))
	}
	output, err := txnCheck(ctx, h.env, parsedInput)
	if err != nil {
		return nil, err
	}
	return json.Marshal(output)
}
func txnCheck(ctx context.Context, env types.Environment, input *TxnCheckInput) (*TxnCheckOutput, error) {
	txn, err := statestore.CreateReadOnlyTxnEnv(ctx, env)
	if err != nil {
		return nil, err
	}
	counterObj := txn.Object("atomic_counter")
	if value, _ := counterObj.Get("count"); value.IsNull() {
		return &TxnCheckOutput{
			Success: false,
			Message: "Failed to get atomic counter: not initialized",
		}, nil
	} else {
		num, err := strconv.ParseInt(value.AsString(), 16, 64)
		if err != nil {
			return &TxnCheckOutput{
				Success: false,
				Message: fmt.Sprintf("Failed to parse fetch-and-add counter due to %v", err),
			}, nil
		}

		if num == input.Count {
			return &TxnCheckOutput{
				Success: true,
				Message: fmt.Sprintf("Count check success count=%v", num),
			}, nil
		} else {
			return &TxnCheckOutput{
				Success: false,
				Message: fmt.Sprintf("Count check failed count=%v, target=%v", num, input.Count),
			}, nil
		}
	}
}

func main() {
	log.SetFlags(log.Lshortfile)
	faas.Serve(&funcHandlerFactory{})
}
