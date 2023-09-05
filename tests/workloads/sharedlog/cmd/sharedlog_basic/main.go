package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"time"

	"cs.utexas.edu/zjia/faas"
	"cs.utexas.edu/zjia/faas/protocol"
	"cs.utexas.edu/zjia/faas/types"
	"github.com/eniac/Beldi/pkg/cayonlib"
	"github.com/pkg/errors"
)

const timeout = time.Second * 60

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

type shardedAuxDataHandler struct {
	env types.Environment
}

type syncToHandler struct {
	env types.Environment
}

type syncToFutureHandler struct {
	env types.Environment
}

type funcHandlerFactory struct {
}

func (f *funcHandlerFactory) New(env types.Environment, funcName string) (types.FuncHandler, error) {
	if funcName == "BasicLogOp" {
		return &basicLogOpHandler{env: env}, nil
	} else if funcName == "AsyncLogOp" {
		return &asyncLogOpHandler{env: env}, nil
	} else if funcName == "AsyncLogOpChild" {
		return &asyncLogOpChildHandler{env: env}, nil
	} else if funcName == "ShardedAuxData" {
		return &shardedAuxDataHandler{env: env}, nil
	} else if funcName == "SyncToOp" {
		return &syncToHandler{env: env}, nil
	} else if funcName == "SyncToFutureOp" {
		return &syncToFutureHandler{env: env}, nil
	} else {
		return nil, nil
	}
}

func (f *funcHandlerFactory) GrpcNew(env types.Environment, service string) (types.GrpcFuncHandler, error) {
	return nil, fmt.Errorf("not implemented")
}

// AuxData format:
type AuxData map[ /*tag*/ uint64] /*value*/ string

func NewAuxData() AuxData {
	return make(AuxData)
}

func DeserializeAuxData(rawData []byte) AuxData {
	if len(rawData) == 0 {
		return nil
	}
	result := NewAuxData()
	if err := json.Unmarshal(rawData, &result); err != nil {
		rawDataStr := "["
		for _, i := range rawData {
			rawDataStr += fmt.Sprintf("%02X ", i)
		}
		rawDataStr += "]"
		panic(errors.Wrap(err, rawDataStr))
	}
	return result
}

func (ab AuxData) Update(tag uint64, value string) AuxData {
	ab[tag] = value
	return ab
}

func (ab AuxData) Serialize() []byte {
	res, err := json.Marshal(ab)
	if err != nil {
		panic(err)
	}
	return res
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
	} else if !reflect.DeepEqual(DeserializeAuxData(logEntry.AuxData), DeserializeAuxData(expected.AuxData)) {
		output += fmt.Sprintf("[FAIL]  %v aux data=%v:%v, expect=%v:%v\n",
			funcName, logEntry.AuxData, string(logEntry.AuxData), expected.AuxData, string(expected.AuxData))
		return output, false
	} else {
		output += fmt.Sprintf("[PASS] %v seqNum=0x%016X, tags=%v, data=%v, auxData=%v\n",
			funcName, logEntry.SeqNum, logEntry.Tags, logEntry.Data, string(logEntry.AuxData))
		return output, true
	}
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
				SeqNum: seqNumAppended,
				Tags:   tags,
				Data:   data,
				// AuxData: []byte{7, 8, 9},
				AuxData: NewAuxData().Update(0 /*tag*/, string([]byte{7, 8, 9})).Serialize(),
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
	tags := []types.Tag{
		{
			StreamType: 1,
			StreamId:   1,
		},
	}
	data := []byte{1, 2, 3}
	var lastFuture types.Future[uint64]
	{
		future, err := h.env.AsyncSharedLogAppend(ctx, tags, data)
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
			return output
		} else {
			output += fmt.Sprintf("[PASS] async shared log append localid: 0x%016X\n", future.GetLocalId())
			if err := future.Await(time.Second); err != nil {
				output += fmt.Sprintf("[FAIL] async shared log verify error: %v\n", err)
				return output
			} else if seqNum, err := future.GetResult(timeout); err == nil {
				output += fmt.Sprintf("[PASS] async shared log append seqNum: 0x%016X\n", seqNum)
				seqNumAppended = seqNum
			} else {
				output += fmt.Sprintf("[FAIL] async shared log get result error: %v\n", err)
				return output
			}
		}
		lastFuture = future
	}
	log.Printf("[INFO] async append with local id: 0x%016X", lastFuture.GetLocalId())
	{
		condLogEntry, err := h.env.AsyncSharedLogRead(ctx, lastFuture.GetLocalId())
		log.Printf("[INFO] async read with local id: 0x%016X response: %+v, %v", lastFuture.GetLocalId(), condLogEntry, err)
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log read error: %v\n", err)
			return output
		} else {
			res, passed := assertLogEntry("async shared log read", &condLogEntry.LogEntry, &types.LogEntry{
				SeqNum:  seqNumAppended,
				Tags:    []uint64{1}, // the same as which in []tags
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
	tags := []types.Tag{
		{
			StreamType: 1,
			StreamId:   1,
		},
	}
	data := []byte{4, 5, 6}
	var lastFuture types.Future[uint64]
	{
		preFuture, err := h.env.AsyncSharedLogAppend(ctx, tags, []byte{1, 2, 3})
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
			return output
		}
		future, err := h.env.AsyncSharedLogAppendWithDeps(ctx, tags, data, []uint64{preFuture.GetLocalId()})
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
			return output
		} else {
			output += fmt.Sprintf("[PASS] async shared log append localid: 0x%016X\n", future.GetLocalId())
			if err := future.Await(time.Second); err != nil {
				output += fmt.Sprintf("[FAIL] async shared log verify error: %v\n", err)
				return output
			} else if seqNum, err := future.GetResult(timeout); err == nil {
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
		condLogEntry, err := h.env.AsyncSharedLogRead(ctx, lastFuture.GetLocalId())
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log read error: %v\n", err)
			return output
		} else {
			res, passed := assertLogEntry("async shared log read", &condLogEntry.LogEntry, &types.LogEntry{
				SeqNum:  seqNumAppended,
				Tags:    []uint64{1}, // the same as which in []tags
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
	tags := []types.Tag{
		{
			StreamType: 1,
			StreamId:   1,
		},
	}
	for i := 0; i < 10; i++ {
		data := []byte{byte(i)}
		future, err := h.env.AsyncSharedLogAppend(ctx, tags, data)
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
			return output
		}
		asyncLogCtx.ChainFuture(future.GetLocalId())
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
	tags := []types.Tag{
		{
			StreamType: 1,
			StreamId:   1,
		},
	}
	data := []byte{2}
	future, err := h.env.AsyncSharedLogAppend(ctx, tags, data)
	if err != nil {
		output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
		return []byte(output), nil
	}
	asyncLogCtx.ChainStep(future.GetLocalId())

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

func (h *shardedAuxDataHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	output := "worker.shardedAuxDataHandler.Call\n"

	// use tag 2 to avoid data unmarshal error by previous set sync log aux data
	tags := []types.Tag{{StreamType: 0, StreamId: 2}}
	data := []byte{1, 2, 3}
	seqNum1, seqNum2, seqNum3 := uint64(0), uint64(0), uint64(0)
	{
		future, err := h.env.AsyncSharedLogAppend(ctx, tags, data)
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
			return []byte(output), nil
		}
		seqNum, err := future.GetResult(60 * time.Second)
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log append get result error: %v\n", err)
			return []byte(output), nil
		}
		seqNum1 = seqNum
	}
	{
		future, err := h.env.AsyncSharedLogAppend(ctx, tags, data)
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
			return []byte(output), nil
		}
		seqNum, err := future.GetResult(60 * time.Second)
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log append get result error: %v\n", err)
			return []byte(output), nil
		}
		seqNum2 = seqNum
	}
	{
		future, err := h.env.AsyncSharedLogAppend(ctx, tags, data)
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
			return []byte(output), nil
		}
		seqNum, err := future.GetResult(60 * time.Second)
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log append get result error: %v\n", err)
			return []byte(output), nil
		}
		seqNum3 = seqNum
	}

	{
		logEntry, err := h.env.AsyncSharedLogCheckTail(ctx, tags[0].StreamId)
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log check tail error: %v\n", err)
			return []byte(output), nil
		} else {
			res, passed := assertLogEntry("async shared log check tail", &logEntry.LogEntry, &types.LogEntry{
				SeqNum:  seqNum3,
				Tags:    []uint64{2},
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
		logEntry, err := h.env.AsyncSharedLogReadPrevWithAux(ctx, tags[0].StreamId, protocol.MaxLogSeqnum)
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log check tail with aux error: %v\n", err)
			return []byte(output), nil
		} else if logEntry == nil {
			output += "[PASS] async shared log check tail with aux return nil because no aux data\n"
		} else {
			output += fmt.Sprintf("[FAIL] async shared log check tail with aux should return nil: %+v\n", logEntry)
			return []byte(output), nil
		}
	}

	{
		auxData := []byte{10, 11, 12}
		if err := h.env.SharedLogSetAuxDataWithShards(ctx, types.LogEntryIndex{
			SeqNum:  seqNum2,
			LocalId: protocol.InvalidLogLocalId,
		}, tags[0].StreamId, auxData); err != nil {
			output += fmt.Sprintf("[FAIL] async shared log set aux data error: %v\n", err)
			return []byte(output), nil
		} else {
			output += fmt.Sprintf("[PASS] async shared log set aux data=%v\n", auxData)
		}
	}
	{
		auxData := []byte{7, 8, 9}
		if err := h.env.SharedLogSetAuxDataWithShards(ctx, types.LogEntryIndex{
			SeqNum:  seqNum1,
			LocalId: protocol.InvalidLogLocalId,
		}, tags[0].StreamId, auxData); err != nil {
			output += fmt.Sprintf("[FAIL] async shared log set aux data error: %v\n", err)
			return []byte(output), nil
		} else {
			output += fmt.Sprintf("[PASS] async shared log set aux data=%v\n", auxData)
		}
	}

	{
		logEntry, err := h.env.AsyncSharedLogReadPrevWithAux(ctx, tags[0].StreamId, protocol.MaxLogSeqnum)
		// logEntry, err := h.env.AsyncSharedLogReadPrev(ctx, tags[0].StreamId, protocol.MaxLogSeqnum)
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log check tail with aux error: %v\n", err)
			return []byte(output), nil
		} else if logEntry == nil {
			output += "[FAIL] async shared log check tail with aux not found\n"
			return []byte(output), nil
		} else {
			res, passed := assertLogEntry("async shared log check tail with aux", &logEntry.LogEntry, &types.LogEntry{
				SeqNum: seqNum2,
				Tags:   []uint64{2},
				Data:   data,
				// AuxData: []byte{10, 11, 12},
				AuxData: NewAuxData().Update(tags[0].StreamId, string([]byte{10, 11, 12})).Serialize(),
			})
			output += res
			if !passed {
				return []byte(output), nil
			}
		}
	}
	{
		logEntry, err := h.env.AsyncSharedLogReadPrevWithAux(ctx, tags[0].StreamId, seqNum1)
		// logEntry, err := h.env.AsyncSharedLogReadPrev(ctx, tags[0].StreamId, protocol.MaxLogSeqnum)
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log check tail with aux error: %v\n", err)
			return []byte(output), nil
		} else if logEntry == nil {
			output += "[FAIL] async shared log check tail with aux not found\n"
			return []byte(output), nil
		} else {
			res, passed := assertLogEntry("async shared log check tail with aux", &logEntry.LogEntry, &types.LogEntry{
				SeqNum: seqNum1,
				Tags:   []uint64{2},
				Data:   data,
				// AuxData: []byte{7, 8, 9},
				AuxData: NewAuxData().Update(tags[0].StreamId, string([]byte{7, 8, 9})).Serialize(),
			})
			output += res
			if !passed {
				return []byte(output), nil
			}
		}
	}
	{
		logEntry, err := h.env.AsyncSharedLogReadPrevWithAux(ctx, 999, protocol.MaxLogSeqnum)
		if err != nil {
			output += fmt.Sprintf("[FAIL] async shared log check tail with aux error: %v\n", err)
			return []byte(output), nil
		} else if logEntry == nil {
			output += "[PASS] async shared log check tail with aux return nil because no aux data\n"
		} else {
			output += fmt.Sprintf("[FAIL] async shared log check tail with aux should return nil: %+v\n", logEntry)
			return []byte(output), nil
		}
	}

	return []byte(output), nil
}

func (h *syncToHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	output := "worker.syncToHandler.Call\n"

	seqNums := make([]uint64, 6)
	logEntries := make([]*types.LogEntry, 0)
	tags := []uint64{3}
	data := []byte{1, 2, 3}
	var err error
	{
		// sync to new position, should return logs appended without the tail one
		for i := 0; i < 3; i++ {
			seqNums[i], err = h.env.SharedLogAppend(ctx, tags, data)
			if err != nil {
				output += fmt.Sprintf("[FAIL] shared log append error: %v\n", err)
				return []byte(output), nil
			}
		}
		count := 0
		seqNum := uint64(0)
		for seqNum < seqNums[2] {
			logEntry, err := h.env.SharedLogReadNextUntil(ctx, tags[0], seqNum, types.LogEntryIndex{
				LocalId: protocol.InvalidLogLocalId,
				SeqNum:  seqNums[2]}, types.ReadOptions{FromCached: false, AuxTags: []uint64{tags[0]}})
			if err != nil {
				output += fmt.Sprintf("[FAIL] log stream entry type assertion failed: %v\n", err)
				return []byte(output), nil
			}
			if logEntry == nil || logEntry.SeqNum >= seqNums[2] {
				break
			}
			seqNum = logEntry.SeqNum + 1
			count++
			logEntries = append(logEntries, logEntry)
		}
		resultSeqNums := make([]uint64, 0)
		for _, logEntry := range logEntries {
			resultSeqNums = append(resultSeqNums, logEntry.SeqNum)
		}
		output += fmt.Sprintf("[INFO] sync to seqnums: %v\n", resultSeqNums)
		if count != 2 {
			output += fmt.Sprintf("[FAIL] incorrect sync to num: %v, need: 2\n", count)
			return []byte(output), nil
		}
		for i := 0; i < 2; i++ {
			if logEntries[i].SeqNum != seqNums[i] {
				output += fmt.Sprintf("[FAIL] incorrect log entry seqnum=%016X need seqnum=%016X\n",
					logEntries[i].SeqNum, seqNums[i])
				return []byte(output), nil
			}
		}
	}
	if err := h.env.SharedLogSetAuxDataWithShards(ctx, types.LogEntryIndex{
		SeqNum:  seqNums[1],
		LocalId: protocol.InvalidLogLocalId,
	}, tags[0], []byte{3, 4, 5}); err != nil {
		output += fmt.Sprintf("[FAIL] set aux data failed: %v\n", err)
		return []byte(output), nil
	}
	{
		// sync to new position, should return prevAux+succeedingLogs without the tail one
		for i := 0; i < 3; i++ {
			seqNums[i+3], err = h.env.SharedLogAppend(ctx, tags, data)
			if err != nil {
				output += fmt.Sprintf("[FAIL] shared log append error: %v\n", err)
				return []byte(output), nil
			}
		}
		count := 0
		seqNum := uint64(0)
		for seqNum < seqNums[5] {
			logEntry, err := h.env.SharedLogReadNextUntil(ctx, tags[0], seqNum, types.LogEntryIndex{
				LocalId: protocol.InvalidLogLocalId,
				SeqNum:  seqNums[5]}, types.ReadOptions{FromCached: true, AuxTags: []uint64{tags[0]}})
			if err != nil {
				output += fmt.Sprintf("[FAIL] log stream entry type assertion failed: %v\n", err)
				return []byte(output), nil
			}
			if logEntry == nil || logEntry.SeqNum >= seqNums[5] {
				break
			}
			seqNum = logEntry.SeqNum + 1
			count++
			logEntries = append(logEntries, logEntry)
		}
		resultSeqNums := make([]uint64, 0)
		for _, logEntry := range logEntries {
			resultSeqNums = append(resultSeqNums, logEntry.SeqNum)
		}
		output += fmt.Sprintf("[INFO] sync to seqnums: %v\n", resultSeqNums)
		if count != 4 {
			output += fmt.Sprintf("[FAIL] incorrect sync to num : %v, need: 4\n", count)
			return []byte(output), nil
		}
		if logEntries[2].AuxData == nil {
			inspect := "[\n"
			for _, logEntry := range logEntries {
				inspect += fmt.Sprintf("\t%+v\n", logEntry)
			}
			output += fmt.Sprintf("[FAIL] incorrect sync to start point, got: %v]\n", inspect)
			return []byte(output), nil
		}
		for i := 3; i < 6; i++ {
			if logEntries[i].SeqNum != seqNums[i-1] {
				output += fmt.Sprintf("[FAIL] incorrect log entry seqnum=%016X need seqnum=%016X\n",
					logEntries[i].SeqNum, seqNums[i-1])
			}
		}
	}
	{
		// sync to an existing view, should return one synced entry
		count := 0
		seqNum := seqNums[0]
		for seqNum < seqNums[2] {
			logEntry, err := h.env.SharedLogReadNextUntil(ctx, tags[0], seqNum, types.LogEntryIndex{
				LocalId: protocol.InvalidLogLocalId,
				SeqNum:  seqNums[2]}, types.ReadOptions{FromCached: true, AuxTags: []uint64{tags[0]}})
			if err != nil {
				output += fmt.Sprintf("[FAIL] log stream entry decode failed: %v\n", err)
				return []byte(output), nil
			}
			if logEntry == nil || logEntry.SeqNum >= seqNums[2] {
				break
			}
			seqNum = logEntry.SeqNum + 1
			count++
			logEntries = append(logEntries, logEntry)
		}
		resultSeqNums := make([]uint64, 0)
		for _, logEntry := range logEntries {
			resultSeqNums = append(resultSeqNums, logEntry.SeqNum)
		}
		output += fmt.Sprintf("[INFO] sync to seqnums: %v\n", resultSeqNums)
		if count != 1 {
			output += fmt.Sprintf("[FAIL] incorrect sync to num : %v, need: 1\n", count)
			return []byte(output), nil
		}
	}
	{
		// sync to an existing view, should return one synced entry
		count := 0
		seqNum := seqNums[0]
		for seqNum < seqNums[1] {
			logEntry, err := h.env.SharedLogReadNextUntil(ctx, tags[0], seqNum, types.LogEntryIndex{
				LocalId: protocol.InvalidLogLocalId,
				SeqNum:  seqNums[1]}, types.ReadOptions{FromCached: true, AuxTags: []uint64{tags[0]}})
			if err != nil {
				output += fmt.Sprintf("[FAIL] log stream entry decode failed: %v\n", err)
				return []byte(output), nil
			}
			if logEntry == nil || logEntry.SeqNum >= seqNums[1] {
				break
			}
			seqNum = logEntry.SeqNum + 1
			count++
			logEntries = append(logEntries, logEntry)
		}
		resultSeqNums := make([]uint64, 0)
		for _, logEntry := range logEntries {
			resultSeqNums = append(resultSeqNums, logEntry.SeqNum)
		}
		output += fmt.Sprintf("[INFO] sync to seqnums: %v\n", resultSeqNums)
		if count != 1 {
			// just return the synced view entry
			output += fmt.Sprintf("[FAIL] incorrect sync to num : %v, need: 1\n", count)
			return []byte(output), nil
		}
	}
	return []byte(output), nil
}

func (h *syncToFutureHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	output := "worker.syncToFutureHandler.Call\n"

	logEntries := make([]*types.LogEntryWithMeta, 0)
	tags := []types.Tag{{StreamType: 0, StreamId: 4}}
	data := []byte{1, 2, 3}
	var err error

	future1, err := h.env.AsyncSharedLogAppend(ctx, tags, data)
	if err != nil {
		output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
		return []byte(output), nil
	}
	future2, err := h.env.AsyncSharedLogAppend(ctx, tags, data)
	if err != nil {
		output += fmt.Sprintf("[FAIL] async shared log append error: %v\n", err)
		return []byte(output), nil
	}

	count := 0
	seqNum := uint64(0)
	for {
		logEntry, err := h.env.AsyncSharedLogReadNextUntil(ctx, tags[0].StreamId, seqNum, types.LogEntryIndex{
			LocalId: future2.GetLocalId(),
			SeqNum:  protocol.InvalidLogSeqNum}, types.ReadOptions{FromCached: true, AuxTags: []uint64{tags[0].StreamId}})
		if err != nil {
			output += fmt.Sprintf("[FAIL] log stream entry decode failed: %v\n", err)
			return []byte(output), nil
		}
		if logEntry == nil || logEntry.LocalId == future2.GetLocalId() {
			break
		}
		seqNum = logEntry.SeqNum + 1
		count++
		logEntries = append(logEntries, logEntry)

		// if future2.IsResolved() {
		// 	seqNum2, err := future2.GetResult(time.Second)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	if seqNum >= seqNum2 {
		// 		break
		// 	}
		// }
	}
	resultSeqNums := make([]uint64, 0)
	for _, logEntry := range logEntries {
		resultSeqNums = append(resultSeqNums, logEntry.SeqNum)
	}
	output += fmt.Sprintf("[INFO] sync to seqnums: %v\n", resultSeqNums)
	if count != 1 {
		output += fmt.Sprintf("[FAIL] incorrect sync to num: %v, need: 1\n", count)
		return []byte(output), nil
	}
	seqNum1, err := future1.GetResult(60 * time.Second)
	if err != nil {
		output += fmt.Sprintf("[FAIL] future1 GetResult failed: %v\n", err)
		return []byte(output), nil
	}
	if logEntries[0].SeqNum != seqNum1 {
		output += fmt.Sprintf("[FAIL] incorrect log entry seqnum=%016X need seqnum=%016X\n",
			logEntries[0].SeqNum, seqNum1)
		return []byte(output), nil
	}
	return []byte(output), nil
}

func main() {
	log.SetFlags(log.Lshortfile)
	faas.Serve(&funcHandlerFactory{})
}
