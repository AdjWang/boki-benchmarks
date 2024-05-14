package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"time"

	"cs.utexas.edu/zjia/faas"
	"cs.utexas.edu/zjia/faas/types"
	"github.com/eniac/Beldi/pkg/cayonlib"
)

type fooHandler struct {
	env types.Environment
}

type barHandler struct {
	env types.Environment
}

type basicLogOpHandler struct {
	env types.Environment
}

type asyncLogOpHandler struct {
	env types.Environment
}

// child function as async log receiver
type asyncLogOpChildHandler struct {
	env types.Environment
}

type benchHandler struct {
	env types.Environment
}

type funcHandlerFactory struct {
}

func (f *funcHandlerFactory) New(env types.Environment, funcName string) (types.FuncHandler, error) {
	if funcName == "Foo" {
		return &fooHandler{env: env}, nil
	} else if funcName == "Bar" {
		return &barHandler{env: env}, nil
	} else if funcName == "BasicLogOp" {
		return &basicLogOpHandler{env: env}, nil
	} else if funcName == "AsyncLogOp" {
		return &asyncLogOpHandler{env: env}, nil
	} else if funcName == "AsyncLogOpChild" {
		return &asyncLogOpChildHandler{env: env}, nil
	} else if funcName == "Bench" {
		return &benchHandler{env: env}, nil
	} else {
		log.Panicf("[FATAL] unknown func name=%v", funcName)
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
		output += fmt.Sprintf("[FAIL] %v seqNum=0x%016X, expect=0x%016X\n", funcName, logEntry.SeqNum, expected.SeqNum)
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

func (h *fooHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	barOutput, err := h.env.InvokeFunc(ctx, "Bar", input)
	if err != nil {
		return nil, err
	}
	output := fmt.Sprintf("foo invokes bar, output=%v", string(barOutput))
	return []byte(output), nil
}

func (h *barHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	output := fmt.Sprintf("bar invoked with arg=%v\n", string(input))
	return []byte(output), nil
}

func (h *basicLogOpHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	output := "worker.basicLogOpHandler.Call\n"
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
			output += fmt.Sprintf("[PASS] shared log append seqNum: 0x%016X\n", seqNum)
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
			output += fmt.Sprintf("[PASS] shared log set aux data=%v\n", auxData)
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

func asyncLogTestAppendRead(ctx context.Context, h *asyncLogOpHandler, output string) string {
	output += "test async log append read\n"
	var seqNumAppended uint64
	tags := []uint64{1}
	tagsMeta := []types.TagMeta{
		{
			FsmType: 1,
			TagKeys: []string{""},
		},
	}
	data := []byte{1, 2, 3}
	var lastFuture types.Future[uint64]
	{
		future, err := h.env.AsyncSharedLogAppend(ctx, tags, tagsMeta, data)
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
			return output
		} else {
			output += fmt.Sprintf("[PASS] async shared log append localid: 0x%016X\n", future.GetMeta().LocalId)
			if err := future.Await(time.Second); err != nil {
				output += fmt.Sprintf("[FAIL] async shared log verify error: %v\n", err)
				return output
			} else if seqNum, err := future.GetResult(); err == nil {
				output += fmt.Sprintf("[PASS] async shared log append seqNum: 0x%016X\n", seqNum)
				seqNumAppended = seqNum
			} else {
				output += fmt.Sprintf("[FAIL] async shared log get result error: %v\n", err)
				return output
			}
		}
		lastFuture = future
	}
	{
		condLogEntry, err := h.env.AsyncSharedLogRead(ctx, lastFuture.GetMeta())
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log read error: %v\n", err)
			return output
		} else {
			res, passed := assertLogEntry("async shared log read", &condLogEntry.LogEntry, &types.LogEntry{
				SeqNum:  seqNumAppended,
				Tags:    tags,
				Data:    data,
				AuxData: []byte{},
			})
			output += res
			// output += fmt.Sprintf("condLogEntry=%+v\n", condLogEntry)
			if !passed {
				return output
			}
		}
	}
	return output
}

func asyncLogTestCondAppendRead(ctx context.Context, h *asyncLogOpHandler, output string) string {
	output += "test async log cond append read\n"
	var seqNumAppended uint64
	tags := []uint64{1}
	tagsMeta := []types.TagMeta{
		{
			FsmType: 1,
			TagKeys: []string{""},
		},
	}
	data := []byte{4, 5, 6}
	var lastFuture types.Future[uint64]
	{
		preFuture, err := h.env.AsyncSharedLogAppend(ctx, tags, tagsMeta, []byte{1, 2, 3})
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
			return output
		}
		CondResolver_IsTheFirstStep := uint8(0)
		future, err := h.env.AsyncSharedLogCondAppend(ctx, tags, tagsMeta, data, func(cond types.CondHandle) {
			cond.AddDep(preFuture.GetMeta())
			cond.AddCond(CondResolver_IsTheFirstStep)
		})
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
			return output
		} else {
			output += fmt.Sprintf("[PASS] async shared log append localid: 0x%016X\n", future.GetMeta().LocalId)
			if err := future.Await(time.Second); err != nil {
				output += fmt.Sprintf("[FAIL] async shared log verify error: %v\n", err)
				return output
			} else if seqNum, err := future.GetResult(); err == nil {
				output += fmt.Sprintf("[PASS] async shared log append seqNum: 0x%016X\n", seqNum)
				seqNumAppended = seqNum
			} else {
				output += fmt.Sprintf("[FAIL] async shared log get result error: %v\n", err)
				return output
			}
		}
		lastFuture = future
	}
	{
		condLogEntry, err := h.env.AsyncSharedLogRead(ctx, lastFuture.GetMeta())
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log read error: %v\n", err)
			return output
		} else {
			res, passed := assertLogEntry("async shared log read", &condLogEntry.LogEntry, &types.LogEntry{
				SeqNum:  seqNumAppended,
				Tags:    tags,
				Data:    data,
				AuxData: []byte{},
			})
			output += res
			// output += fmt.Sprintf("condLogEntry=%+v\n", condLogEntry)
			if !passed {
				return output
			}
		}
	}
	return output
}

func asyncLogTestSync(ctx context.Context, h *asyncLogOpHandler, output string) string {
	output += "test async log sync\n"
	asyncLogCtx := cayonlib.NewAsyncLogContext(h.env)
	tags := []uint64{1}
	tagsMeta := []types.TagMeta{
		{
			FsmType: 1,
			TagKeys: []string{""},
		},
	}
	for i := 0; i < 10; i++ {
		data := []byte{byte(i)}
		future, err := h.env.AsyncSharedLogAppend(ctx, tags, tagsMeta, data)
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
			return output
		}
		asyncLogCtx.ChainFuture(future.GetMeta())
	}
	err := asyncLogCtx.Sync(time.Second)
	if err != nil {
		output += fmt.Sprintf("[FAIL] async shared log sync error: %v\n", err)
		return output
	} else {
		output += fmt.Sprintln("[PASS] async shared log sync succeed")
	}
	return output
}

func (h *asyncLogOpHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	output := "worker.asyncLogOpHandler.Call\n"
	// list env
	output += fmt.Sprintf("env.FAAS_ENGINE_ID=%v\n", os.Getenv("FAAS_ENGINE_ID"))
	output += fmt.Sprintf("env.FAAS_CLIENT_ID=%v\n", os.Getenv("FAAS_CLIENT_ID"))

	output += asyncLogTestAppendRead(ctx, h, output)
	output += asyncLogTestCondAppendRead(ctx, h, output)
	output += asyncLogTestSync(ctx, h, output)

	output += "test async log ctx propagate\n"
	asyncLogCtx := cayonlib.NewAsyncLogContext(h.env)
	tags := []uint64{1}
	tagsMeta := []types.TagMeta{
		{
			FsmType: 1,
			TagKeys: []string{""},
		},
	}
	data := []byte{2}
	future, err := h.env.AsyncSharedLogAppend(ctx, tags, tagsMeta, data)
	if err != nil {
		output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
		return []byte(output), nil
	}
	asyncLogCtx.ChainStep(future.GetMeta())

	asyncLogCtxData, err := asyncLogCtx.Serialize()
	if err != nil {
		output += fmt.Sprintf("[FAIL] async shared log propagate serialize error: %v\n", err)
		return []byte(output), nil
	}
	res, err := h.env.InvokeFunc(ctx, "AsyncLogOpChild", asyncLogCtxData)
	return bytes.Join([][]byte{[]byte(output), res}, nil), err
}

func (h *asyncLogOpChildHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	output := "worker.asyncLogOpChildHandler.Call\n"
	// list env
	output += fmt.Sprintf("env.FAAS_ENGINE_ID=%v\n", os.Getenv("FAAS_ENGINE_ID"))
	output += fmt.Sprintf("env.FAAS_CLIENT_ID=%v\n", os.Getenv("FAAS_CLIENT_ID"))

	asyncLogCtx, err := cayonlib.DeserializeAsyncLogContext(h.env, input)
	if err != nil {
		output += fmt.Sprintf("[FAIL] async shared log ctx propagate restore error: %v\n", err)
		return []byte(output), nil
	}
	// DEBUG: print
	output += fmt.Sprintf("async log ctx: %v\n", asyncLogCtx)

	err = asyncLogCtx.Sync(time.Second)
	if err != nil {
		output += fmt.Sprintf("[FAIL] async shared log remote sync error: %v\n", err)
		return []byte(output), nil
	} else {
		output += fmt.Sprintln("[PASS] async shared log remote sync succeed")
	}

	return []byte(output), nil
}

func (h *benchHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	// output := "worker.benchHandler.Call\n"

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

	// test := ""
	seqNum, err := h.env.SharedLogAppend(ctx, tags, data)
	if err != nil {
		panic(err)
	}
	// { // consistency check
	// 	logEntry, err := h.env.SharedLogReadNext(ctx, 1 /*tag*/, 0)
	// 	test += fmt.Sprintln("[TEST]", logEntry, err)
	// }
	// { // consistency check
	// 	logEntry, err := h.env.SharedLogReadNext(ctx, 1 /*tag*/, seqNum)
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
