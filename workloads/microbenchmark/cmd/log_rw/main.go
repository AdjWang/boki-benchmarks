package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cs.utexas.edu/zjia/faas"
	"cs.utexas.edu/zjia/faas/types"
	"cs.utexas.edu/zjia/microbenchmark/common"
	"cs.utexas.edu/zjia/microbenchmark/utils"
)

type bokiLogAppendHandler struct {
	env types.Environment
}

type asyncLogAppendOpHandler struct {
	env types.Environment
}

type funcHandlerFactory struct {
}

func (f *funcHandlerFactory) New(env types.Environment, funcName string) (types.FuncHandler, error) {
	if funcName == "benchBokiLogAppend" {
		return &bokiLogAppendHandler{env: env}, nil
	} else if funcName == "benchAsyncLogAppend" {
		return &asyncLogAppendOpHandler{env: env}, nil
	} else {
		return nil, nil
	}
}

func (f *funcHandlerFactory) GrpcNew(env types.Environment, service string) (types.GrpcFuncHandler, error) {
	return nil, fmt.Errorf("not implemented")
}

func (h *bokiLogAppendHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.BokiLogAppendInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	output, err := bokiLogAppend(ctx, h.env, parsedInput)
	if err != nil {
		return nil, err
	}
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func bokiLogAppend(ctx context.Context, env types.Environment, input *common.BokiLogAppendInput) (*common.FnOutput, error) {
	duration := time.Duration(input.Duration) * time.Second
	interval := time.Duration(input.IntervalMs) * time.Millisecond

	latencies := make([]int, 0, 128) // record push duration
	startTime := time.Now()
	numMessages := make([]int, 0, 128)
	for time.Since(startTime) < duration {
		// prepare payload
		payloads := make([]string, 0, input.BatchSize)
		for i := 0; i < input.BatchSize; i++ {
			payload := utils.RandomString(input.PayloadSize - utils.TimestampStrLen)
			payloads = append(payloads, payload)
		}
		pushStart := time.Now()
		// bench test case
		tags := []uint64{1}
		for _, payload := range payloads {
			_, err := env.SharedLogAppend(ctx, tags, []byte(payload))
			if err != nil {
				return &common.FnOutput{
					Success:  false,
					Message:  fmt.Sprintf("Log append failed: %v", err),
					Duration: time.Since(startTime).Seconds(),
				}, nil
			}
		}
		// record
		elapsed := time.Since(pushStart)
		// record push duration
		latencies = append(latencies, int(elapsed.Microseconds()))
		// record push num
		numMessages = append(numMessages, len(payloads))
		// sleep for `interval`
		time.Sleep(time.Until(pushStart.Add(interval)))
	}
	return &common.FnOutput{
		Success:     true,
		Duration:    time.Since(startTime).Seconds(),
		Latencies:   latencies,
		NumMessages: numMessages,
	}, nil
}

func (h *asyncLogAppendOpHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.AsyncLogAppendInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	output, err := asyncLogAppend(ctx, h.env, parsedInput)
	if err != nil {
		return nil, err
	}
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func asyncLogAppend(ctx context.Context, env types.Environment, input *common.AsyncLogAppendInput) (*common.FnOutput, error) {
	duration := time.Duration(input.Duration) * time.Second
	interval := time.Duration(input.IntervalMs) * time.Millisecond

	latencies := make([]int, 0, 128) // record push duration
	startTime := time.Now()
	numMessages := make([]int, 0, 128)
	for time.Since(startTime) < duration {
		// prepare payload
		payloads := make([]string, 0, input.BatchSize)
		for i := 0; i < input.BatchSize; i++ {
			payload := utils.RandomString(input.PayloadSize - utils.TimestampStrLen)
			payloads = append(payloads, payload)
		}
		pushStart := time.Now()
		// bench test case
		tags := []uint64{1}
		tagsMeta := []types.TagMeta{{FsmType: 1, TagKeys: []string{""}}}
		futures := make([]types.Future[uint64], 0, len(payloads))
		deps := []uint64{}
		for _, payload := range payloads {
			future, err := env.AsyncSharedLogCondAppend(ctx, tags, tagsMeta, []byte(payload), deps)
			if err != nil {
				return &common.FnOutput{
					Success:  false,
					Message:  fmt.Sprintf("AsyncLogAppend failed: %v", err),
					Duration: time.Since(startTime).Seconds(),
				}, nil
			}
			futures = append(futures, future)
			deps = []uint64{future.GetLocalId()}
		}
		for _, future := range futures {
			if err := future.Await(60 * time.Second); err != nil {
				return &common.FnOutput{
					Success:  false,
					Message:  fmt.Sprintf("AsyncLogAppend await failed: %v", err),
					Duration: time.Since(startTime).Seconds(),
				}, nil
			}
		}
		// record
		elapsed := time.Since(pushStart)
		// record push duration
		latencies = append(latencies, int(elapsed.Microseconds()))
		// record push num
		numMessages = append(numMessages, len(payloads))
		// sleep for `interval`
		time.Sleep(time.Until(pushStart.Add(interval)))
	}
	return &common.FnOutput{
		Success:     true,
		Duration:    time.Since(startTime).Seconds(),
		Latencies:   latencies,
		NumMessages: numMessages,
	}, nil
}

// func asyncLogTestSync(ctx context.Context, h *asyncLogOpHandler, output string) string {
// 	output += "test async log sync\n"
// 	asyncLogCtx := cayonlib.NewAsyncLogContext(h.env)
// 	tags := []uint64{1}
// 	tagsMeta := []types.TagMeta{
// 		{
// 			FsmType: 1,
// 			TagKeys: []string{""},
// 		},
// 	}
// 	for i := 0; i < 10; i++ {
// 		data := []byte{byte(i)}
// 		future, err := h.env.AsyncSharedLogAppend(ctx, tags, tagsMeta, data)
// 		if err != nil {
// 			output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
// 			return output
// 		}
// 		asyncLogCtx.ChainFuture(future.GetLocalId())
// 	}
// 	err := asyncLogCtx.Sync(time.Second)
// 	if err != nil {
// 		output += fmt.Sprintf("[FAIL] async shared log sync error: %v\n", err)
// 		return output
// 	} else {
// 		output += fmt.Sprintln("[PASS] async shared log sync succeed")
// 	}
// 	return output
// }

// func (h *asyncLogOpHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
// 	output := "test async log ctx propagate\n"
// 	asyncLogCtx := cayonlib.NewAsyncLogContext(h.env)
// 	tags := []uint64{1}
// 	tagsMeta := []types.TagMeta{
// 		{
// 			FsmType: 1,
// 			TagKeys: []string{""},
// 		},
// 	}
// 	data := []byte{2}
// 	future, err := h.env.AsyncSharedLogAppend(ctx, tags, tagsMeta, data)
// 	if err != nil {
// 		output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
// 		return []byte(output), nil
// 	}
// 	asyncLogCtx.ChainStep(future.GetLocalId())

// 	asyncLogCtxData, err := asyncLogCtx.Serialize()
// 	if err != nil {
// 		output += fmt.Sprintf("[FAIL] async shared log propagate serialize error: %v\n", err)
// 		return []byte(output), nil
// 	}
// 	res, err := h.env.InvokeFunc(ctx, "AsyncLogOpChild", asyncLogCtxData)
// 	return bytes.Join([][]byte{[]byte(output), res}, nil), err
// }

// func (h *asyncLogOpChildHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
// 	output := "worker.asyncLogOpChildHandler.Call\n"
// 	// list env
// 	output += fmt.Sprintf("env.FAAS_ENGINE_ID=%v\n", os.Getenv("FAAS_ENGINE_ID"))
// 	output += fmt.Sprintf("env.FAAS_CLIENT_ID=%v\n", os.Getenv("FAAS_CLIENT_ID"))

// 	asyncLogCtx, err := cayonlib.DeserializeAsyncLogContext(h.env, input)
// 	if err != nil {
// 		output += fmt.Sprintf("[FAIL] async shared log ctx propagate restore error: %v\n", err)
// 		return []byte(output), nil
// 	}
// 	// DEBUG: print
// 	output += fmt.Sprintf("async log ctx: %v\n", asyncLogCtx)

// 	err = asyncLogCtx.Sync(time.Second)
// 	if err != nil {
// 		output += fmt.Sprintf("[FAIL] async shared log remote sync error: %v\n", err)
// 		return []byte(output), nil
// 	} else {
// 		output += fmt.Sprintln("[PASS] async shared log remote sync succeed")
// 	}

// 	return []byte(output), nil
// }

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	faas.Serve(&funcHandlerFactory{})
}
