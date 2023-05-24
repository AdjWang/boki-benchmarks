package cayonlib

import (
	"fmt"

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
	instanceId string
	stepNumber int32
	FsmCommon[IntentLogEntry]
	stepLogs     map[int32]*IntentLogEntry
	postStepLogs map[int32]*IntentLogEntry
}

// Implement Fsm and FsmReceiver
func NewIntentFsm(instanceId string) *IntentFsm {
	this := &IntentFsm{
		instanceId:   instanceId,
		stepNumber:   0,
		FsmCommon:    NewEmptyFsmCommon[IntentLogEntry](),
		stepLogs:     make(map[int32]*IntentLogEntry),
		postStepLogs: make(map[int32]*IntentLogEntry),
	}
	this.receiver = this
	return this
}

func (fsm *IntentFsm) ApplyLog(logEntry *types.CondLogEntry) bool {
	decoded, err := snappy.Decode(nil, logEntry.Data)
	CHECK(err)
	var intentLog IntentLogEntry
	err = json.Unmarshal(decoded, &intentLog)
	CHECK(err)
	if intentLog.InstanceId == fsm.instanceId {
		// log.Printf("[INFO] Found my log: seqnum=%d, step=%d", logEntry.SeqNum, intentLog.StepNumber)
		intentLog.SeqNum = logEntry.SeqNum
		fsm.applyLog(&intentLog)
	}
	// resolve cond
	if intentLog.PostStep {
		preStepLog := fsm.GetStepLog(intentLog.StepNumber)
		// we believe a post step log must has a preceding PreStepLog
		ASSERT(preStepLog != nil,
			fmt.Sprintf("post step log %+v not has its pre step log in %+v", intentLog, fsm.stepLogs))
		return true
	} else {
		recordedIntentLog := fsm.GetStepLog(intentLog.StepNumber)
		if recordedIntentLog == nil {
			panic(fmt.Sprintf("Cannot find intent log for step %d", intentLog.StepNumber))
		}
		return logEntry.SeqNum == recordedIntentLog.SeqNum
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
			fsm.stepNumber++
			fsm.stepLogs[step] = intentLog
		}
	}
}

func (fsm *IntentFsm) GetTag() uint64 {
	return IntentStepStreamTag(fsm.instanceId)
}

func (fsm *IntentFsm) GetTailSeqNum() uint64 {
	return fsm.tail.SeqNum
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

func AsyncProposeNextStep(env *Env, data aws.JSONValue, depLocalId uint64) (types.Future[uint64], *IntentLogEntry) {
	step := env.StepNumber
	env.StepNumber += 1
	intentLog := env.FsmHub.GetInstanceStepFsm().GetStepLog(step)
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
				FsmType: FsmType_STEPSTREAM,
				TagKeys: []string{env.InstanceId},
			},
		},
		&intentLog,
		depLocalId,
	)
	return future, intentLog
}

func AsyncLogStepResult(env *Env, instanceId string, stepNumber int32, data aws.JSONValue, depLocalId uint64) types.Future[uint64] {
	return LibAsyncAppendLog(env, IntentStepStreamTag(instanceId),
		[]types.TagMeta{
			{
				FsmType: FsmType_STEPSTREAM,
				TagKeys: []string{instanceId},
			},
		},
		&IntentLogEntry{
			InstanceId: instanceId,
			StepNumber: stepNumber,
			PostStep:   true,
			Data:       data,
		},
		depLocalId,
	)
}

func FetchStepResultLog(env *Env, stepNumber int32, catch bool) *IntentLogEntry {
	intentLog := env.FsmHub.GetInstanceStepFsm().GetPostStepLog(stepNumber)
	if intentLog != nil {
		return intentLog
	}
	if catch {
		env.FsmHub.GetInstanceStepFsm().Catch(env)
	} else {
		return nil
	}
	return env.FsmHub.GetInstanceStepFsm().GetPostStepLog(stepNumber)
}

func LibSyncAppendLog(env *Env, tag uint64, tagMeta []types.TagMeta, data interface{}, depLocalId uint64) {
	future := LibAsyncAppendLog(env, tag, tagMeta, data, depLocalId)
	env.AsyncLogCtx.ChainStep(future.GetLocalId())
	// sync until receives index
	// If the async log is not propagated to a different engine, waiting for
	// the seqnum is enough to gaurantee read-your-write consistency.
	err := future.Await(gSyncTimeout)
	CHECK(err)

	// // But wait for index is fast enough so no need to do this optimization.
	// indexFuture, err := env.FaasEnv.AsyncSharedLogReadIndex(env.FaasCtx, future.GetLocalId())
	// CHECK(err)
	// err = indexFuture.Await(gSyncTimeout)
	// CHECK(err)
}

func LibAsyncAppendLog(env *Env, tag uint64, tagMeta []types.TagMeta, data interface{}, depLocalId uint64) types.Future[uint64] {
	serializedData, err := json.Marshal(data)
	CHECK(err)
	encoded := snappy.Encode(nil, serializedData)
	future, err := env.FaasEnv.AsyncSharedLogCondAppend(env.FaasCtx, []uint64{tag}, tagMeta, encoded, []uint64{depLocalId})
	CHECK(err)
	return future
}
func CheckLogDataField(intentLog *IntentLogEntry, field string, expected string) {
	if tmp := intentLog.Data[field].(string); tmp != expected {
		panic(fmt.Sprintf("Field %s mismatch: expected=%s, have=%s", field, expected, tmp))
	}
}
