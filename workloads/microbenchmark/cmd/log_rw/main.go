package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"cs.utexas.edu/zjia/faas"
	"cs.utexas.edu/zjia/faas/types"
	"cs.utexas.edu/zjia/microbenchmark/common"
	"cs.utexas.edu/zjia/microbenchmark/handlers"
)

type bokiLogAppendHandler struct {
	env types.Environment
}

func (h *bokiLogAppendHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &handlers.AppendInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	output, err := handlers.BokiLogAppend(ctx, h.env, parsedInput)
	if err != nil {
		return nil, err
	}
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

type asyncLogAppendOpHandler struct {
	env types.Environment
}

func (h *asyncLogAppendOpHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &handlers.AppendInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	output, err := handlers.AsyncLogAppend(ctx, h.env, parsedInput)
	if err != nil {
		return nil, err
	}
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

type bokiLogReadHandler struct {
	env types.Environment
}

func (h *bokiLogReadHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &handlers.ReadInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	output, err := handlers.BokiLogSyncTo(ctx, h.env, parsedInput)
	if err != nil {
		return nil, err
	}
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

type asyncLogReadHandler struct {
	env types.Environment
}

func (h *asyncLogReadHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &handlers.ReadInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	output, err := handlers.AsyncLogSyncTo(ctx, h.env, parsedInput)
	if err != nil {
		return nil, err
	}
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

type funcHandlerFactory struct {
}

func (f *funcHandlerFactory) New(env types.Environment, funcName string) (types.FuncHandler, error) {
	if funcName == "benchBokiLogAppend" {
		return &bokiLogAppendHandler{env: env}, nil
	} else if funcName == "benchAsyncLogAppend" {
		return &asyncLogAppendOpHandler{env: env}, nil
	} else if funcName == "benchBokiLogRead" {
		return &bokiLogReadHandler{env: env}, nil
	} else if funcName == "benchAsyncLogRead" {
		return &asyncLogReadHandler{env: env}, nil
	} else {
		return nil, nil
	}
}

func (f *funcHandlerFactory) GrpcNew(env types.Environment, service string) (types.GrpcFuncHandler, error) {
	return nil, fmt.Errorf("not implemented")
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	faas.Serve(&funcHandlerFactory{})
}
