package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"cs.utexas.edu/zjia/faas"
	"cs.utexas.edu/zjia/faas/types"
)

type fooHandler struct {
	env types.Environment
}

type barHandler struct {
	env types.Environment
}

type funcHandlerFactory struct {
}

func (f *funcHandlerFactory) New(env types.Environment, funcName string) (types.FuncHandler, error) {
	if funcName == "Foo" {
		return &fooHandler{env: env}, nil
	} else if funcName == "Bar" {
		return &barHandler{env: env}, nil
	} else {
		return nil, nil
	}
}

func (f *funcHandlerFactory) GrpcNew(env types.Environment, service string) (types.GrpcFuncHandler, error) {
	return nil, fmt.Errorf("Not implemented")
}

var count int = 0

func (h *fooHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	// barOutput, err := h.env.InvokeFunc(ctx, "Bar", input)
	// if err != nil {
	// 	return nil, err
	// }
	// output := fmt.Sprintf("From function Bar: %s", string(barOutput))

	// output := fmt.Sprintf("[Inside Foo] %s, World, count: %d\n", string(input), count)
	count++
	// if count == 5 {
	// 	panic("manually err")
	// }

	// prof
	start := time.Now()
	defer func() {
		engineId, err := strconv.Atoi(os.Getenv("FAAS_ENGINE_ID"))
		if err != nil {
			engineId = -1
		}
		duration := time.Since(start)
		fmt.Printf("[PROF] LibAppendLog engine=%v, tag=%v, duration=%v\n", engineId, 233, duration.Seconds())
	}()

	seqNum, err := h.env.SharedLogAppend(context.Background(), []uint64{233}, []byte{byte(count)})
	output := fmt.Sprintf("shared log append count: %v, seqNum: %v, err: %v\n", byte(count), seqNum, err)

	return []byte(output), nil
}

func (h *barHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	output := fmt.Sprintf("%s, World\n", string(input))
	return []byte(output), nil
}

func main() {
	faas.Serve(&funcHandlerFactory{})
}
