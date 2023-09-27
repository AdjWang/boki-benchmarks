package ipcbench

import (
	"context"
	"fmt"
	"time"

	"cs.utexas.edu/zjia/faas/types"
	"cs.utexas.edu/zjia/microbenchmark/common"
)

func IPCBench(ctx context.Context, env types.Environment, input *common.FnInput) (*common.FnOutput, error) {
	startTs := time.Now()
	for i := 0; i < input.PayloadSize; i++ {
		err := env.SharedLogIPCBench(ctx, uint64(input.BatchSize))
		if err != nil {
			return &common.FnOutput{
				Success: false,
				Message: fmt.Sprint(err),
			}, nil
		}
	}
	elapsed := time.Since(startTs)
	return &common.FnOutput{
		Success:      true,
		AsyncLatency: -1,
		Latency:      int(elapsed.Microseconds()),
		BatchSize:    input.BatchSize,
	}, nil
}
