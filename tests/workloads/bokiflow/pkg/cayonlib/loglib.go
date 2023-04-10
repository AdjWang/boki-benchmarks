package cayonlib

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	// "log"
	"encoding/json"
	// "cs.utexas.edu/zjia/faas/types"

	"cs.utexas.edu/zjia/faas/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/golang/snappy"
	// "context"
)

type IntentLogEntry struct {
	SeqNum     uint64        `json:"-"`
	InstanceId string        `json:"instanceId"`
	StepNumber int32         `json:"step"`
	PostStep   bool          `json:"postStep"`
	Data       aws.JSONValue `json:"data"`
}

type IntentFsm struct {
	instanceId   string
	stepNumber   int32
	tail         *IntentLogEntry
	stepLogs     map[int32]*IntentLogEntry
	postStepLogs map[int32]*IntentLogEntry
}

func NewIntentFsm(instanceId string) *IntentFsm {
	return &IntentFsm{
		instanceId:   instanceId,
		stepNumber:   0,
		tail:         nil,
		stepLogs:     make(map[int32]*IntentLogEntry),
		postStepLogs: make(map[int32]*IntentLogEntry),
	}
}

func (fsm *IntentFsm) applyLog(intentLog *IntentLogEntry) {
	fsm.tail = intentLog
	step := intentLog.StepNumber
	if intentLog.PostStep {
		if _, exists := fsm.postStepLogs[step]; !exists {
			fsm.postStepLogs[step] = intentLog
		}
	} else {
		if _, exists := fsm.stepLogs[step]; !exists {
			if step != fsm.stepNumber {
				panic(fmt.Sprintf("StepNumber is not monotonic: expected=%d, seen=%d", fsm.stepNumber, step))
			}
			fsm.stepNumber += 1
			fsm.stepLogs[step] = intentLog
		}
	}
}

// func (fsm *IntentFsm) Catch(env *Env) {
// 	tag := IntentStepStreamTag(fsm.instanceId)
// 	seqNum := uint64(0)
// 	if fsm.tail != nil {
// 		seqNum = fsm.tail.SeqNum + 1
// 	}
// 	for {
// 		logEntry, err := env.FaasEnv.SharedLogReadNext(env.FaasCtx, tag, seqNum)
// 		CHECK(err)
// 		if logEntry == nil {
// 			break
// 		}
// 		decoded, err := snappy.Decode(nil, logEntry.Data)
// 		CHECK(err)
// 		var intentLog IntentLogEntry
// 		err = json.Unmarshal(decoded, &intentLog)
// 		CHECK(err)
// 		if intentLog.InstanceId == fsm.instanceId {
// 			// log.Printf("[INFO] Found my log: seqnum=%d, step=%d", logEntry.SeqNum, intentLog.StepNumber)
// 			intentLog.SeqNum = logEntry.SeqNum
// 			fsm.applyLog(&intentLog)
// 		}
// 		seqNum = logEntry.SeqNum + 1
// 	}
// }

const (
	LogEntryState_PENDING   uint8 = 0
	LogEntryState_APPLIED   uint8 = 1
	LogEntryState_DISCARDED uint8 = 2
)

type LogState struct {
	State uint8 `json:"state"`
}

// return:
//
//	resolved/rejected: true
//	pending: false
func (fsm *IntentFsm) resolve(env *Env, condLogEntry *types.CondLogEntry) (bool, uint64) {
	// resolve deps
	for _, dep := range condLogEntry.Deps {
		depCondLogEntry, err := env.FaasEnv.AsyncSharedLogRead(env.FaasCtx, dep)
		CHECK(err)

		var logState LogState
		decoded, err := snappy.Decode(nil, depCondLogEntry.AuxData)
		CHECK(err)
		if len(decoded) > 0 {
			err = json.Unmarshal(decoded, &logState)
			CHECK(err)
		} else {
			logState = LogState{LogEntryState_PENDING}
		}

	recheck:
		switch logState.State {
		case LogEntryState_PENDING:
			if inside(IntentStepStreamTag(env.InstanceId), depCondLogEntry.Tags) && depCondLogEntry.SeqNum > condLogEntry.SeqNum {
				// to pending
				return false, depCondLogEntry.SeqNum
			} else if inside(IntentStepStreamTag(env.InstanceId), depCondLogEntry.Tags) && depCondLogEntry.SeqNum < condLogEntry.SeqNum {
				log.Println(dep.State)
				panic("impossible here that deps.state is pending")
			} else { // if !inside(IntentStepStreamTag(env.InstanceId), depCondLogEntry.Tags) {
				logState = resolveByTag(env, depCondLogEntry)
				if logState.State == LogEntryState_PENDING {
					panic("impossible")
				}
				goto recheck
			}
		case LogEntryState_APPLIED:
			continue
		case LogEntryState_DISCARDED:
			serializedData, err := json.Marshal(LogState{LogEntryState_DISCARDED})
			CHECK(err)
			encoded := snappy.Encode(nil, serializedData)
			err = env.FaasEnv.SharedLogSetAuxData(env.FaasCtx, condLogEntry.SeqNum, encoded)
			CHECK(err)
			// rejected
			return true, 0
		default:
			panic("unreachable")
		}
	}

	// from here the deps are all solved, condLogEntry.State must be PENDING
	if ok := types.CheckOps(condLogEntry.Cond); ok {
		decoded, err := snappy.Decode(nil, condLogEntry.Data)
		CHECK(err)
		var intentLog IntentLogEntry
		err = json.Unmarshal(decoded, &intentLog)
		CHECK(err)
		if intentLog.InstanceId == fsm.instanceId {
			// log.Printf("[INFO] Found my log: seqnum=%d, step=%d", logEntry.SeqNum, intentLog.StepNumber)
			intentLog.SeqNum = condLogEntry.SeqNum
			fsm.applyLog(&intentLog)
		}

		serializedData, err := json.Marshal(LogState{LogEntryState_APPLIED})
		CHECK(err)
		encoded := snappy.Encode(nil, serializedData)
		err = env.FaasEnv.SharedLogSetAuxData(env.FaasCtx, condLogEntry.SeqNum, encoded)
		CHECK(err)
	} else {
		serializedData, err := json.Marshal(LogState{LogEntryState_DISCARDED})
		CHECK(err)
		encoded := snappy.Encode(nil, serializedData)
		err = env.FaasEnv.SharedLogSetAuxData(env.FaasCtx, condLogEntry.SeqNum, encoded)
		CHECK(err)
	}
	// resolved/rejected
	return true, 0
}

func (fsm *IntentFsm) Catch(env *Env) {
	tag := IntentStepStreamTag(fsm.instanceId)
	seqNum := uint64(0)
	if fsm.tail != nil {
		seqNum = fsm.tail.SeqNum + 1
	}
	pendings := make(map[uint64][]*types.CondLogEntry)
	for {
		condLogEntry, err := env.FaasEnv.AsyncSharedLogReadNext(env.FaasCtx, tag, seqNum)
		CHECK(err)
		if condLogEntry == nil {
			break
		}
		if notPending, jumpSeqNum := fsm.resolve(env, condLogEntry); notPending {
			resolvedSeqNum := condLogEntry.SeqNum
			if pendingLogs, ok := pendings[resolvedSeqNum]; ok {
				for _, log := range pendingLogs {
					if notPending_, jumpSeqNum_ := fsm.resolve(env, log); notPending_ {
						continue
					} else {
						if jumpSeqNum_ <= resolvedSeqNum {
							panic("impossible")
						}
						pendings[jumpSeqNum_] = append(pendings[jumpSeqNum_], log)
					}
				}
				delete(pendings, resolvedSeqNum)
			}
		} else {
			if _, ok := pendings[jumpSeqNum]; !ok {
				pendings[jumpSeqNum] = make([]*types.CondLogEntry, 0, 10)
			}
			entries := pendings[jumpSeqNum]
			entries = append(entries, condLogEntry)
		}
		seqNum = condLogEntry.SeqNum + 1
	}
}

func resolveByTag(env *Env, log *types.CondLogEntry) LogState {
	tag := log.Tags[0]
	if tag == IntentLogTag {
		panic("impossible that a step depend to a intent")
	}
	tagBuildMeta := log.TagBuildMeta[0]
	switch tagBuildMeta.FsmType {
	case FsmType_INTENT:
		fsm := NewIntentFsm(tagBuildMeta.TagKeys[0])
		fsm.Catch(env)
		break
	default:
		panic("unreachable")
	}
	condLogEntry, err := env.FaasEnv.AsyncSharedLogReadNext(env.FaasCtx, tag, log.SeqNum)
	CHECK(err)
	if condLogEntry == nil {
		panic("impossible")
	}
	var logState LogState
	decoded, err := snappy.Decode(nil, log.AuxData)
	CHECK(err)
	if len(decoded) > 0 {
		err = json.Unmarshal(decoded, &logState)
		CHECK(err)
	} else {
		panic("impossible")
	}
	if logState.State == LogEntryState_PENDING {
		panic("impossible")
	}
	return logState
}

func inside(a uint64, b []uint64) bool {
	for _, i := range b {
		if a == i {
			return true
		}
	}
	return false
}

func (fsm *IntentFsm) GetStepLog(stepNumber int32) *IntentLogEntry {
	if log, exists := fsm.stepLogs[stepNumber]; exists {
		return log
	} else {
		return nil
	}
}

func (fsm *IntentFsm) GetPostStepLog(stepNumber int32) *IntentLogEntry {
	if log, exists := fsm.postStepLogs[stepNumber]; exists {
		return log
	} else {
		return nil
	}
}

func ProposeNextStep(env *Env, data aws.JSONValue) (bool, *IntentLogEntry) {
	step := env.StepNumber
	env.StepNumber += 1
	intentLog := env.Fsm.GetStepLog(step)
	if intentLog != nil {
		return false, intentLog
	}
	intentLog = &IntentLogEntry{
		InstanceId: env.InstanceId,
		StepNumber: step,
		PostStep:   false,
		Data:       data,
	}
	seqNum := LibAppendLog(env, IntentStepStreamTag(env.InstanceId), &intentLog)
	env.Fsm.Catch(env)
	intentLog = env.Fsm.GetStepLog(step)
	if intentLog == nil {
		panic(fmt.Sprintf("Cannot find intent log for step %d", step))
	}
	return seqNum == intentLog.SeqNum, intentLog
}

func LogStepResult(env *Env, instanceId string, stepNumber int32, data aws.JSONValue) {
	LibAppendLog(env, IntentStepStreamTag(instanceId), &IntentLogEntry{
		InstanceId: instanceId,
		StepNumber: stepNumber,
		PostStep:   true,
		Data:       data,
	})
}

func AsyncProposeNextStep(env *Env, data aws.JSONValue, deps []types.FutureMeta) (types.Future[uint64], *IntentLogEntry) {
	step := env.StepNumber
	env.StepNumber += 1
	intentLog := env.Fsm.GetStepLog(step)
	if intentLog != nil {
		return nil, intentLog
	}
	intentLog = &IntentLogEntry{
		InstanceId: env.InstanceId,
		StepNumber: step,
		PostStep:   false,
		Data:       data,
	}
	future := LibAsyncAppendLog(env, IntentStepStreamTag(env.InstanceId),
		[]types.TagMeta{
			{
				FsmType: FsmType_INTENT,
				TagKeys: []string{env.InstanceId},
			},
		},
		&intentLog,
		func(cond types.CondHandle) {
			for _, dep := range deps {
				cond.AddDep(dep)
			}
			// TODO: read and compare
		},
	)
	// env.Fsm.Catch(env)
	// intentLog = env.Fsm.GetStepLog(step)
	// if intentLog == nil {
	// 	panic(fmt.Sprintf("Cannot find intent log for step %d", step))
	// }
	// return seqNum == intentLog.SeqNum, intentLog
	return future, intentLog
}

func AsyncLogStepResult(env *Env, instanceId string, stepNumber int32, data aws.JSONValue, cond func(types.CondHandle)) types.Future[uint64] {
	return LibAsyncAppendLog(env, IntentStepStreamTag(instanceId),
		[]types.TagMeta{
			{
				FsmType: FsmType_INTENT,
				TagKeys: []string{instanceId},
			},
		},
		&IntentLogEntry{
			InstanceId: instanceId,
			StepNumber: stepNumber,
			PostStep:   true,
			Data:       data,
		},
		cond,
	)
}

func FetchStepResultLog(env *Env, stepNumber int32, catch bool) *IntentLogEntry {
	intentLog := env.Fsm.GetPostStepLog(stepNumber)
	if intentLog != nil {
		return intentLog
	}
	if catch {
		env.Fsm.Catch(env)
	} else {
		return nil
	}
	return env.Fsm.GetPostStepLog(stepNumber)
}

func LibAppendLog(env *Env, tag uint64, data interface{}) uint64 {
	encoded := []byte{}
	start := time.Now()
	defer func() {
		engineId, err := strconv.Atoi(os.Getenv("FAAS_ENGINE_ID"))
		if err != nil {
			engineId = -1
		}
		duration := time.Since(start)
		fmt.Printf("[PROF] LibAppendLog engine=%v, tag=%v, duration=%v, datalen=%v\n", engineId, tag, duration.Seconds(), len(encoded))
	}()
	serializedData, err := json.Marshal(data)
	CHECK(err)
	encoded = snappy.Encode(nil, serializedData)
	seqNum, err := env.FaasEnv.SharedLogAppend(env.FaasCtx, []uint64{tag}, encoded)
	CHECK(err)
	return seqNum
}

func LibAsyncAppendLog(env *Env, tag uint64, tagMeta []types.TagMeta, data interface{}, cond func(types.CondHandle)) types.Future[uint64] {
	serializedData, err := json.Marshal(data)
	CHECK(err)
	encoded := snappy.Encode(nil, serializedData)
	future, err := env.FaasEnv.AsyncSharedLogCondAppend(env.FaasCtx, []uint64{tag}, tagMeta, encoded, cond)
	CHECK(err)
	return future
}
func CheckLogDataField(intentLog *IntentLogEntry, field string, expected string) {
	if tmp := intentLog.Data[field].(string); tmp != expected {
		panic(fmt.Sprintf("Field %s mismatch: expected=%s, have=%s", field, expected, tmp))
	}
}
