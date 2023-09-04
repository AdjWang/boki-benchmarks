package handlers

import (
	"context"
	"fmt"
	"time"

	"cs.utexas.edu/zjia/faas/types"
	"cs.utexas.edu/zjia/microbenchmark/common"
	"cs.utexas.edu/zjia/microbenchmark/utils"
)

type AppendOutput struct {
	Success       bool     `json:"success"`
	Message       string   `json:"message"`
	Stage1Latency int      `json:"latency1"`
	Stage2Latency int      `json:"latency2"`
	BatchSize     int      `json:"batchSize"`
	SeqNums       []uint64 `json:"seqNums"`
}

type AppendInput struct {
	PayloadSize int `json:"payloadSize"`
	BatchSize   int `json:"batchSize"`
}

func BokiLogAppend(ctx context.Context, env types.Environment, input *AppendInput) (*AppendOutput, error) {
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
			return &AppendOutput{
				Success: false,
				Message: fmt.Sprintf("Log append failed: %v", err),
			}, nil
		}
		seqNums = append(seqNums, seqNum)
	}
	elapsed := time.Since(pushStart)
	return &AppendOutput{
		Success:       true,
		Stage1Latency: -1,
		Stage2Latency: int(elapsed.Microseconds()),
		BatchSize:     input.BatchSize,
		SeqNums:       seqNums,
	}, nil
}

func AsyncLogAppend(ctx context.Context, env types.Environment, input *AppendInput) (*AppendOutput, error) {
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
			return &AppendOutput{
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
			return &AppendOutput{
				Success: false,
				Message: fmt.Sprintf("AsyncLogAppend await failed: %v", err),
			}, nil
		} else {
			seqNums = append(seqNums, seqNum)
		}
	}
	// record
	elapsed := time.Since(pushStart)
	return &AppendOutput{
		Success:       true,
		Stage1Latency: int(asyncElapsed.Microseconds()),
		Stage2Latency: int(elapsed.Microseconds()),
		BatchSize:     input.BatchSize,
		SeqNums:       seqNums,
	}, nil
}
