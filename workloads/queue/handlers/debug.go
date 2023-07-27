package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cs.utexas.edu/zjia/faas-queue/common"
	"cs.utexas.edu/zjia/faas-queue/utils"

	"cs.utexas.edu/zjia/faas/types"
)

type debugProducerHandler struct {
	env      types.Environment
	seqNumCh chan string
}

type debugConsumerHandler struct {
	env types.Environment
}

func NewDebugProducerHandler(env types.Environment) types.FuncHandler {
	return &debugProducerHandler{
		env:      env,
		seqNumCh: utils.SeqNumGenerator(),
	}
}

func NewDebugConsumerHandler(env types.Environment) types.FuncHandler {
	return &debugConsumerHandler{env: env}
}

func (h *debugProducerHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.ProducerFnInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	output, err := producerDebug(ctx, h.env, parsedInput, h.seqNumCh)
	if err != nil {
		return nil, err
	}
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (h *debugConsumerHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.ConsumerFnInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	output, err := consumerDebug(ctx, h.env, parsedInput)
	if err != nil {
		return nil, err
	}
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

// only do once to debug
func producerDebug(ctx context.Context, env types.Environment, input *common.ProducerFnInput, seqNumCh chan string) (*common.FnOutput, error) {
	q, err := createQueue(ctx, env, input.QueueName, input.QueueShards)
	if err != nil {
		return &common.FnOutput{
			Success: false,
			Message: fmt.Sprintf("NewQueue failed: %v", err),
		}, nil
	}
	seqNumStr := <-seqNumCh
	payload := utils.RandomString(input.PayloadSize - utils.TimestampStrLen - utils.SeqNumStrLen)
	payload = seqNumStr + payload
	payload = utils.FormatTime(time.Now()) + payload
	err = q.Push(payload)
	if err != nil {
		return &common.FnOutput{
			Success: false,
			Message: fmt.Sprintf("QueuePush failed: %v", err),
		}, nil
	}
	return &common.FnOutput{
		Success: true,
	}, nil
}

// only do once to debug
func consumerDebug(ctx context.Context, env types.Environment, input *common.ConsumerFnInput) (*common.FnOutput, error) {
	q, err := createQueue(ctx, env, input.QueueName, input.QueueShards)
	if err != nil {
		return &common.FnOutput{
			Success: false,
			Message: fmt.Sprintf("NewQueue failed: %v", err),
		}, nil
	}
	payload, err := q.Pop()
	if err != nil {
		return &common.FnOutput{
			Success: false,
			Message: fmt.Sprintf("QueuePop failed: %v", err),
		}, nil
	}
	// PROF
	// log.Printf("prof=[%v]\n", q.GetProfInfo())

	return &common.FnOutput{
		Success: true,
		Message: fmt.Sprint(utils.ParseSeqNum(payload)),
	}, nil
}
