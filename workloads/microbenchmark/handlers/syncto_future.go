package handlers

import (
	"context"
	"fmt"
	"time"

	"cs.utexas.edu/zjia/faas/types"
	"cs.utexas.edu/zjia/microbenchmark/common"
	"cs.utexas.edu/zjia/microbenchmark/utils"
)

func BokiLogSyncTo(ctx context.Context, env types.Environment, input *ReadInput) (*ReadOutput, error) {
	if input.BatchSize == 0 {
		return nil, fmt.Errorf("Invalid zero batchsize")
	}
	tags := []uint64{1}
	payload := utils.RandomString(input.PayloadSize - utils.TimestampStrLen)
	seqNumStart, err := env.SharedLogAppend(ctx, tags, []byte(payload))
	if err != nil {
		return nil, err
	}
	synctoStart := time.Now()
	seqNums := make([]uint64, 0, input.BatchSize)
	for i := 0; i < input.BatchSize; i++ {
		payload := utils.RandomString(input.PayloadSize - utils.TimestampStrLen)
		seqNum, err := env.SharedLogAppend(ctx, tags, []byte(payload))
		if err != nil {
			return nil, err
		}
		seqNums = append(seqNums, seqNum)
	}
	appendElapsed := time.Since(synctoStart)
	seqNumsRead, err := syncToForward(ctx, env, seqNumStart, seqNums[len(seqNums)-1])
	if err != nil {
		return &ReadOutput{
			Success: false,
			Message: fmt.Sprintf("syncToForward failed: %v", err),
		}, nil
	}
	synctoElapsed := time.Since(synctoStart)
	return &ReadOutput{
		Success:       true,
		Stage1Latency: int(appendElapsed.Microseconds()),
		Stage2Latency: int(synctoElapsed.Microseconds()),
		BatchSize:     input.BatchSize,
		SeqNums:       seqNumsRead,
	}, nil
}

func AsyncLogSyncTo(ctx context.Context, env types.Environment, input *ReadInput) (*ReadOutput, error) {
	if input.BatchSize == 0 {
		return nil, fmt.Errorf("Invalid zero batchsize")
	}
	tags := []types.Tag{{StreamType: 1, StreamId: uint64(1)}}
	payload := utils.RandomString(input.PayloadSize - utils.TimestampStrLen)
	future, err := env.AsyncSharedLogAppend(ctx, tags, []byte(payload))
	if err != nil {
		return nil, err
	}
	seqNumStart, err := future.GetResult(common.Timeout)
	if err != nil {
		return nil, err
	}
	synctoStart := time.Now()
	// bench test case
	futures := make([]types.Future[uint64], 0, input.BatchSize)
	deps := []uint64{}
	for i := 0; i < input.BatchSize; i++ {
		payload := utils.RandomString(input.PayloadSize - utils.TimestampStrLen)
		future, err := env.AsyncSharedLogAppendWithDeps(ctx, tags, []byte(payload), deps)
		if err != nil {
			return nil, err
		}
		futures = append(futures, future)
		deps = []uint64{future.GetLocalId()}
	}
	appendElapsed := time.Since(synctoStart)
	seqNumsRead, err := syncToFuture(ctx, env, seqNumStart, futures[len(futures)-1].GetLogEntryIndex())
	if err != nil {
		return &ReadOutput{
			Success: false,
			Message: fmt.Sprintf("asyncToForward failed: %v", err),
		}, nil
	}
	synctoElapsed := time.Since(synctoStart)
	return &ReadOutput{
		Success:       true,
		Stage1Latency: int(appendElapsed.Microseconds()),
		Stage2Latency: int(synctoElapsed.Microseconds()),
		BatchSize:     input.BatchSize,
		SeqNums:       seqNumsRead,
	}, nil
}
