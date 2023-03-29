package main

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"

	"cs.utexas.edu/zjia/faas"
	"cs.utexas.edu/zjia/faas/types"
)

type basicLogOpHandler struct {
	env types.Environment
}

type benchHandler struct {
	env types.Environment
}

type funcHandlerFactory struct {
}

func (f *funcHandlerFactory) New(env types.Environment, funcName string) (types.FuncHandler, error) {
	if funcName == "BasicLogOp" {
		return &basicLogOpHandler{env: env}, nil
	} else if funcName == "Bench" {
		return &benchHandler{env: env}, nil
	} else {
		return nil, nil
	}
}

func (f *funcHandlerFactory) GrpcNew(env types.Environment, service string) (types.GrpcFuncHandler, error) {
	return nil, fmt.Errorf("not implemented")
}

func assertLogEntry(funcName string, logEntry *types.LogEntry, expected *types.LogEntry) (string, bool) {
	output := ""
	if logEntry == nil && expected != nil {
		output += fmt.Sprintf("[FAIL] %v logEntry=%v, expect=%v\n", funcName, logEntry, expected)
		return output, false
	} else if logEntry == nil && expected == nil {
		output += fmt.Sprintf("[PASS] %v logEntry==nil assert true\n", funcName)
		return output, true
	} else if logEntry.SeqNum != expected.SeqNum {
		output += fmt.Sprintf("[FAIL] %v seqNum=0x%016X, expect=%v\n", funcName, logEntry.SeqNum, expected.SeqNum)
		return output, false
	} else if !reflect.DeepEqual(logEntry.Tags, expected.Tags) {
		output += fmt.Sprintf("[FAIL] %v tags=%v, expect=%v\n", funcName, logEntry.Tags, expected.Tags)
		return output, false
	} else if !reflect.DeepEqual(logEntry.Data, expected.Data) {
		output += fmt.Sprintf("[FAIL] %v data=%v, expect=%v\n", funcName, logEntry.Data, expected.Data)
		return output, false
	} else if !reflect.DeepEqual(logEntry.AuxData, expected.AuxData) {
		output += fmt.Sprintf("[FAIL]  %v aux data=%v, expect=%v\n", funcName, logEntry.AuxData, expected.AuxData)
		return output, false
	} else {
		output += fmt.Sprintf("[PASS] %v seqNum=0x%016X, tags=%v, data=%v, auxData=%v\n",
			funcName, logEntry.SeqNum, logEntry.Tags, logEntry.Data, logEntry.AuxData)
		return output, true
	}
}

func (h *basicLogOpHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	output := ""
	// list env
	output += fmt.Sprintf("env.FAAS_ENGINE_ID=%v\n", os.Getenv("FAAS_ENGINE_ID"))
	output += fmt.Sprintf("env.FAAS_CLIENT_ID=%v\n", os.Getenv("FAAS_CLIENT_ID"))
	// test
	var seqNumAppended uint64
	tags := []uint64{1}
	data := []byte{1, 2, 3}
	{
		seqNum, err := h.env.SharedLogAppend(ctx, tags, data)
		if err != nil {
			output += fmt.Sprintf("[FAIL] shared log append error: %v\n", err)
			return []byte(output), nil
		} else {
			output += fmt.Sprintf("[PASS] shared log append seqNum: %v\n", seqNum)
			seqNumAppended = seqNum
		}
	}
	readTag := uint64(1)
	{
		logEntry, err := h.env.SharedLogReadNext(ctx, readTag, 0)
		if err != nil {
			output += fmt.Sprintf("[FAIL] shared log read next error: %v\n", err)
			return []byte(output), nil
		} else {
			res, passed := assertLogEntry("shared log read next", logEntry, &types.LogEntry{
				SeqNum:  seqNumAppended,
				Tags:    tags,
				Data:    data,
				AuxData: []byte{},
			})
			output += res
			if !passed {
				return []byte(output), nil
			}
		}
	}
	{
		logEntry, err := h.env.SharedLogReadNextBlock(ctx, readTag, 0)
		if err != nil {
			output += fmt.Sprintf("[FAIL] shared log read next error: %v\n", err)
			return []byte(output), nil
		} else {
			res, passed := assertLogEntry("shared log read next block", logEntry, &types.LogEntry{
				SeqNum:  seqNumAppended,
				Tags:    tags,
				Data:    data,
				AuxData: []byte{},
			})
			output += res
			if !passed {
				return []byte(output), nil
			}
		}
	}
	{
		logEntry, err := h.env.SharedLogReadPrev(ctx, readTag, seqNumAppended)
		if err != nil {
			output += fmt.Sprintf("[FAIL] shared log read next error: %v\n", err)
			return []byte(output), nil
		} else {
			res, passed := assertLogEntry("shared log read prev", logEntry, &types.LogEntry{
				SeqNum:  seqNumAppended,
				Tags:    tags,
				Data:    data,
				AuxData: []byte{},
			})
			output += res
			if !passed {
				return []byte(output), nil
			}
		}
	}
	auxData := []byte{7, 8, 9}
	{
		if err := h.env.SharedLogSetAuxData(ctx, seqNumAppended, auxData); err != nil {
			output += fmt.Sprintf("[FAIL] shared log set aux data error: %v\n", err)
			return []byte(output), nil
		} else {
			output += fmt.Sprintf("[PASS] shared log set aux data=%v", auxData)
		}
	}
	{
		logEntry, err := h.env.SharedLogCheckTail(ctx, readTag)
		if err != nil {
			output += fmt.Sprintf("[FAIL] shared log check tail error: %v\n", err)
			return []byte(output), nil
		} else {
			res, passed := assertLogEntry("shared log check tail", logEntry, &types.LogEntry{
				SeqNum:  seqNumAppended,
				Tags:    tags,
				Data:    data,
				AuxData: []byte{7, 8, 9},
			})
			output += res
			if !passed {
				return []byte(output), nil
			}
		}
	}

	return []byte(output), nil
}

func (h *benchHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	// prof
	engineId, err := strconv.Atoi(os.Getenv("FAAS_ENGINE_ID"))
	if err != nil {
		engineId = -1
	}

	tags := []uint64{1}
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i)
	}

	start := time.Now()

	seqNum, err := h.env.SharedLogAppend(ctx, tags, data)
	// test := ""
	// { // consistency check
	// 	logEntry, err := h.env.SharedLogReadNext(ctx, 1 /*tag*/, seqNum)
	// 	test += fmt.Sprintln("[TEST]", logEntry, err)
	// }
	// { // consistency check
	// 	logEntry, err := h.env.SharedLogReadNext(ctx, 1 /*tag*/, seqNum)
	// 	test += fmt.Sprintln("[TEST]", logEntry, err)
	// }
	// { // consistency check
	// 	logEntry, err := h.env.SharedLogReadNextBlock(ctx, 1 /*tag*/, seqNum)
	// 	test += fmt.Sprintln("[TEST]", logEntry, err)
	// }

	duration := time.Since(start)
	prof := fmt.Sprintf("[PROF] LibAppendLog 1k engine=%v, tag=%v, duration=%v\n", engineId, 1, duration.Seconds())

	// { // consistency check
	// 	logEntry, err := h.env.SharedLogReadNext(ctx, 1 /*tag*/, seqNum)
	// 	test += fmt.Sprintln("[TEST]", logEntry, err)
	// }

	if err != nil {
		return []byte(fmt.Sprintf("[FAIL] shared log append 1k error: %v\n", err)), nil
	} else {
		// return []byte(test + prof + fmt.Sprintf("[PASS] shared log append 1k seqNum=0x%016X\n", seqNum)), nil
		return []byte(prof + fmt.Sprintf("[PASS] shared log append 1k seqNum=0x%016X\n", seqNum)), nil
	}
}

func main() {
	faas.Serve(&funcHandlerFactory{})
}
