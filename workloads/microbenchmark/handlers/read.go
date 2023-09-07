package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"cs.utexas.edu/zjia/faas/protocol"
	"cs.utexas.edu/zjia/faas/types"
	"cs.utexas.edu/zjia/microbenchmark/common"
)

type ReadOutput struct {
	Success       bool     `json:"success"`
	Message       string   `json:"message"`
	Stage1Latency int      `json:"latency1"`
	Stage2Latency int      `json:"latency2"`
	BatchSize     int      `json:"batchSize"`
	SeqNums       []uint64 `json:"seqNums"`
}

type ReadInput struct {
	PayloadSize  int  `json:"payloadSize"`
	BatchSize    int  `json:"batchSize"`
	ReadAsStream bool `json:"readAsStream"`
}

func syncToForward(ctx context.Context, env types.Environment, headSeqNum uint64, tailSeqNum uint64) ([]uint64, error) {
	tag := uint64(1)
	seqNum := headSeqNum
	if tailSeqNum < seqNum {
		log.Fatalf("[FATAL] Current seqNum=%#016x, cannot sync to %#016x", seqNum, tailSeqNum)
	}
	seqNums := make([]uint64, 0, 100)
	for seqNum < tailSeqNum {
		logEntry, err := env.SharedLogReadNext(ctx, tag, seqNum)
		if err != nil {
			return nil, err
		}
		if logEntry == nil || logEntry.SeqNum >= tailSeqNum {
			break
		}
		seqNum = logEntry.SeqNum + 1
		seqNums = append(seqNums, logEntry.SeqNum)

		// bussiness logics:
		// logContent := decodeLogEntry(logEntry)
		// ...
	}
	return seqNums, nil
}

func syncToFuture(ctx context.Context, env types.Environment, headSeqNum uint64, logIndex types.LogEntryIndex) ([]uint64, error) {
	seqNums := make([]uint64, 0, 100)
	seqNum := headSeqNum
	for seqNum < logIndex.SeqNum {
		logEntry, err := env.SharedLogReadNextUntil(ctx, 1 /*tag*/, seqNum, logIndex,
			types.ReadOptions{FromCached: true, AuxTags: []uint64{1}})
		if err != nil {
			return nil, err
		}
		if logEntry == nil || logEntry.LocalId == logIndex.LocalId || logEntry.SeqNum >= logIndex.SeqNum {
			break
		}
		seqNum = logEntry.SeqNum + 1
		seqNums = append(seqNums, logEntry.SeqNum)
	}
	return seqNums, nil
}

func BokiLogRead(ctx context.Context, env types.Environment, input *ReadInput) (*ReadOutput, error) {
	output, err := BokiLogAppend(ctx, env, &AppendInput{
		PayloadSize: input.PayloadSize,
		BatchSize:   input.BatchSize,
	})
	if err != nil {
		return nil, err
	} else if !output.Success {
		return nil, fmt.Errorf("BokiLogAppend error: %+v", output)
	} else if err := common.AssertSeqNumOrder(output.SeqNums); err != nil {
		// log appends issued from a single function should be ordered
		// whether it is async or not
		return &ReadOutput{
			Success: false,
			Message: fmt.Sprintf("log order assertion failed: %v", err),
		}, nil
	}
	seqNums := output.SeqNums
	var seqNumsRead []uint64
	readStart := time.Now()
	if input.ReadAsStream {
		logIndex := types.LogEntryIndex{LocalId: protocol.InvalidLogLocalId, SeqNum: seqNums[len(seqNums)-1]}
		seqNumsRead, err = syncToFuture(ctx, env, seqNums[0], logIndex)
	} else {
		seqNumsRead, err = syncToForward(ctx, env, seqNums[0], seqNums[len(seqNums)-1])
	}
	if err != nil {
		return &ReadOutput{
			Success: false,
			Message: fmt.Sprintf("syncToForward failed: %v", err),
		}, nil
	}
	elapsed := time.Since(readStart)
	log.Printf("[DEBUG] got logEntry n_seqnums=%v", len(seqNumsRead))
	return &ReadOutput{
		Success:       true,
		Stage1Latency: -1,
		Stage2Latency: int(elapsed.Microseconds()),
		BatchSize:     input.BatchSize,
		SeqNums:       seqNums,
	}, nil
}

func asyncToForward(ctx context.Context, env types.Environment,
	headSeqNum uint64, tailSeqNum uint64) (time.Duration, time.Duration, error) {

	readStart := time.Now()
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
	asyncElapsed := time.Since(readStart)
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
	elapsed := time.Since(readStart)
	return asyncElapsed, elapsed, nil
}
func AsyncLogRead(ctx context.Context, env types.Environment, input *ReadInput) (*ReadOutput, error) {
	output, err := AsyncLogAppend(ctx, env, &AppendInput{
		PayloadSize: input.PayloadSize,
		BatchSize:   input.BatchSize,
	})
	if err != nil {
		return nil, err
	} else if !output.Success {
		return nil, fmt.Errorf("AsyncLogAppend error: %+v", output)
	} else if err := common.AssertSeqNumOrder(output.SeqNums); err != nil {
		// log appends issued from a single function should be ordered
		// whether it is async or not
		return &ReadOutput{
			Success: false,
			Message: fmt.Sprintf("log order assertion failed: %v", err),
		}, nil
	}
	seqNums := output.SeqNums
	asyncElapsed, elapsed, err := asyncToForward(ctx, env, seqNums[0], seqNums[len(seqNums)-1])
	if err != nil {
		return &ReadOutput{
			Success: false,
			Message: fmt.Sprintf("asyncToForward failed: %v", err),
		}, nil
	}
	return &ReadOutput{
		Success:       true,
		Stage1Latency: int(asyncElapsed.Microseconds()),
		Stage2Latency: int(elapsed.Microseconds()),
		BatchSize:     input.BatchSize,
		SeqNums:       seqNums,
	}, nil
}
