package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cs.utexas.edu/zjia/faas"
	"cs.utexas.edu/zjia/faas/protocol"
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

type bokiLogReadHandler struct {
	env types.Environment
}

type asyncLogReadHandler struct {
	env types.Environment
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

// faas functions

func (h *bokiLogAppendHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.FnInput{}
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

func bokiLogAppend(ctx context.Context, env types.Environment, input *common.FnInput) (*common.FnOutput, error) {
	// prepare payload
	payloads := make([]string, 0, input.BatchSize)
	for i := 0; i < input.BatchSize; i++ {
		payload := utils.RandomString(input.PayloadSize - utils.TimestampStrLen)
		payloads = append(payloads, payload)
	}
	seqNums := make([]uint64, 0, input.BatchSize)
	pushStart := time.Now()
	// bench test case
	tags := []uint64{1}
	for _, payload := range payloads {
		seqNum, err := env.SharedLogAppend(ctx, tags, []byte(payload))
		if err != nil {
			return &common.FnOutput{
				Success: false,
				Message: fmt.Sprintf("Log append failed: %v", err),
			}, nil
		}
		seqNums = append(seqNums, seqNum)
	}
	elapsed := time.Since(pushStart)
	return &common.FnOutput{
		Success:      true,
		AsyncLatency: -1,
		Latency:      int(elapsed.Microseconds()),
		BatchSize:    input.BatchSize,
		SeqNums:      seqNums,
	}, nil
}

func (h *asyncLogAppendOpHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.FnInput{}
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

func asyncLogAppend(ctx context.Context, env types.Environment, input *common.FnInput) (*common.FnOutput, error) {
	// prepare payload
	payloads := make([]string, 0, input.BatchSize)
	for i := 0; i < input.BatchSize; i++ {
		payload := utils.RandomString(input.PayloadSize - utils.TimestampStrLen)
		payloads = append(payloads, payload)
	}
	seqNums := make([]uint64, 0, input.BatchSize)
	pushStart := time.Now()
	// bench test case
	tags := []types.Tag{{StreamType: 1, StreamId: uint64(1)}}
	futures := make([]types.Future[uint64], 0, len(payloads))
	deps := []uint64{}
	for _, payload := range payloads {
		future, err := env.AsyncSharedLogAppendWithDeps(ctx, tags, []byte(payload), deps)
		if err != nil {
			return &common.FnOutput{
				Success: false,
				Message: fmt.Sprintf("AsyncLogAppend failed: %v", err),
			}, nil
		}
		futures = append(futures, future)
		deps = []uint64{future.GetLocalId()}
	}
	asyncElapsed := time.Since(pushStart)
	for _, future := range futures {
		if seqNum, err := future.GetResult(common.Timeout); err != nil {
			return &common.FnOutput{
				Success: false,
				Message: fmt.Sprintf("AsyncLogAppend await failed: %v", err),
			}, nil
		} else {
			seqNums = append(seqNums, seqNum)
		}
	}
	// record
	elapsed := time.Since(pushStart)
	return &common.FnOutput{
		Success:      true,
		AsyncLatency: int(asyncElapsed.Microseconds()),
		Latency:      int(elapsed.Microseconds()),
		BatchSize:    input.BatchSize,
		SeqNums:      seqNums,
	}, nil
}

func (h *bokiLogReadHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.FnInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	output, err := bokiLogRead(ctx, h.env, parsedInput)
	if err != nil {
		return nil, err
	}
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func syncToForward(ctx context.Context, env types.Environment, headSeqNum uint64, tailSeqNum uint64) (time.Duration, error) {
	popStart := time.Now()
	tag := uint64(1)
	seqNum := headSeqNum
	if tailSeqNum < seqNum {
		log.Fatalf("[FATAL] Current seqNum=%#016x, cannot sync to %#016x", seqNum, tailSeqNum)
	}
	for seqNum <= tailSeqNum {
		logEntry, err := env.SharedLogReadNext(ctx, tag, seqNum)
		if err != nil {
			return -1, err
		}
		if logEntry == nil || logEntry.SeqNum >= tailSeqNum {
			break
		}
		seqNum = logEntry.SeqNum + 1

		// bussiness logics:
		// logContent := decodeLogEntry(logEntry)
		// ...
	}
	elapsed := time.Since(popStart)
	return elapsed, nil
}
func syncToForwardU(ctx context.Context, env types.Environment, headSeqNum uint64, tailSeqNum uint64) (time.Duration, error) {
	popStart := time.Now()
	logStream := env.SharedLogReadNextUntil(ctx, 1 /*tag*/, types.LogEntryIndex{LocalId: protocol.InvalidLogLocalId, SeqNum: tailSeqNum})
	doneCh := make(chan struct{})
	errCh := make(chan error)
	go func(ctx context.Context) {
		for {
			var logEntry *types.LogEntry
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
				logStreamEntry, err := logStream.DequeueOrWaitForNextElement()
				if err != nil {
					errCh <- ctx.Err()
					return
				}
				logEntry = logStreamEntry.(types.LogStreamEntry[types.LogEntry]).LogEntry
				err = logStreamEntry.(types.LogStreamEntry[types.LogEntry]).Err
				if err != nil {
					errCh <- ctx.Err()
					return
				}
			}
			if logEntry == nil {
				doneCh <- struct{}{}
				break
			}
			log.Printf("[DEBUG] got logEntry seqnum=%016X", logEntry.SeqNum)
		}
	}(ctx)
	select {
	case <-doneCh:
		elapsed := time.Since(popStart)
		return elapsed, nil
	case err := <-errCh:
		elapsed := time.Since(popStart)
		return elapsed, err
	}
}
func bokiLogRead(ctx context.Context, env types.Environment, input *common.FnInput) (*common.FnOutput, error) {
	output, err := bokiLogAppend(ctx, env, input)
	if err != nil {
		return nil, err
	} else if !output.Success {
		return output, nil
	} else if err := common.AssertSeqNumOrder(output); err != nil {
		// log appends issued from a single function should be ordered
		// whether it is async or not
		return &common.FnOutput{
			Success: false,
			Message: fmt.Sprintf("log order assertion failed: %v", err),
		}, nil
	}
	seqNums := output.SeqNums
	if input.ReadCached {
		// fill cache
		// _, err := syncToForward(ctx, env, seqNums[0], seqNums[len(seqNums)-1])
		// if err != nil {
		// 	return &common.FnOutput{
		// 		Success: false,
		// 		Message: fmt.Sprintf("syncToForward failed: %v", err),
		// 	}, nil
		// }
	}
	elapsed, err := syncToForwardU(ctx, env, seqNums[0], seqNums[len(seqNums)-1])
	if err != nil {
		return &common.FnOutput{
			Success: false,
			Message: fmt.Sprintf("syncToForward failed: %v", err),
		}, nil
	}
	return &common.FnOutput{
		Success:      true,
		AsyncLatency: -1,
		Latency:      int(elapsed.Microseconds()),
		BatchSize:    input.BatchSize,
		SeqNums:      seqNums,
	}, nil
}

func (h *asyncLogReadHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.FnInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	output, err := asyncLogRead(ctx, h.env, parsedInput)
	if err != nil {
		return nil, err
	}
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func asyncToForward(ctx context.Context, env types.Environment,
	headSeqNum uint64, tailSeqNum uint64) (time.Duration, time.Duration, error) {

	popStart := time.Now()
	futures := make([]types.Future[*types.LogEntryWithMeta], 0, 100)
	tag := uint64(1)
	seqNum := headSeqNum
	if tailSeqNum < seqNum {
		log.Fatalf("[FATAL] Current seqNum=%#016x, cannot sync to %#016x", seqNum, tailSeqNum)
	}
	for seqNum <= tailSeqNum {
		logEntryFuture, err := env.AsyncSharedLogReadNext2(ctx, tag, seqNum)
		if err != nil {
			return -1, -1, err
		}
		if logEntryFuture == nil || logEntryFuture.GetSeqNum() >= tailSeqNum {
			break
		}
		seqNum = logEntryFuture.GetSeqNum() + 1
		futures = append(futures, logEntryFuture)
	}
	asyncElapsed := time.Since(popStart)
	for _, logEntryFuture := range futures {
		// logEntry, err := logEntryFuture.GetResult()
		_, err := logEntryFuture.GetResult(common.Timeout)
		if err != nil {
			return -1, -1, err
		}
		// bussiness logics:
		// logContent := decodeLogEntry(logEntry)
		// ...
	}
	elapsed := time.Since(popStart)
	return asyncElapsed, elapsed, nil
}
func asyncLogRead(ctx context.Context, env types.Environment, input *common.FnInput) (*common.FnOutput, error) {
	output, err := asyncLogAppend(ctx, env, input)
	if err != nil {
		return nil, err
	} else if !output.Success {
		return output, nil
	} else if err := common.AssertSeqNumOrder(output); err != nil {
		// log appends issued from a single function should be ordered
		// whether it is async or not
		return &common.FnOutput{
			Success: false,
			Message: fmt.Sprintf("log order assertion failed: %v", err),
		}, nil
	}
	seqNums := output.SeqNums
	if input.ReadCached {
		// FIXME: cache miss may only happens on cross engine append/read
		// fill cache
		// _, _, err := asyncToForward(ctx, env, seqNums[0], seqNums[len(seqNums)-1])
		// if err != nil {
		// 	return &common.FnOutput{
		// 		Success: false,
		// 		Message: fmt.Sprintf("asyncToForward failed: %v", err),
		// 	}, nil
		// }
	}
	asyncElapsed, elapsed, err := asyncToForward(ctx, env, seqNums[0], seqNums[len(seqNums)-1])
	if err != nil {
		return &common.FnOutput{
			Success: false,
			Message: fmt.Sprintf("asyncToForward failed: %v", err),
		}, nil
	}
	return &common.FnOutput{
		Success:      true,
		AsyncLatency: int(asyncElapsed.Microseconds()),
		Latency:      int(elapsed.Microseconds()),
		BatchSize:    input.BatchSize,
		SeqNums:      seqNums,
	}, nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	faas.Serve(&funcHandlerFactory{})
}
