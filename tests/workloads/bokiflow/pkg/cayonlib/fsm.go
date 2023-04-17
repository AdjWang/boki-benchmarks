package cayonlib

import (
	"encoding/json"
	"log"

	"cs.utexas.edu/zjia/faas/types"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
)

const (
	LogEntryState_PENDING   uint8 = 0
	LogEntryState_APPLIED   uint8 = 1
	LogEntryState_DISCARDED uint8 = 2
)

type LogState struct {
	State uint8 `json:"state"`
}

type Fsm interface {
	Catch(env *Env)
}
type FsmReceiver interface {
	ApplyLog(log *types.CondLogEntry) bool
	GetTag() uint64
	GetTailSeqNum() uint64
}

// Implement Fsm
type FsmCommon[TLogEntry any] struct {
	reciever FsmReceiver
	tail     *TLogEntry
}

func (fsm *FsmCommon[TLogEntry]) Catch(env *Env) {
	tag := fsm.reciever.GetTag()
	seqNum := uint64(0)
	if fsm.tail != nil {
		seqNum = fsm.reciever.GetTailSeqNum() + 1
	}
	pendings := make(map[uint64][]*types.CondLogEntry)
	for {
		condLogEntry, err := env.FaasEnv.AsyncSharedLogReadNext(env.FaasCtx, tag, seqNum)
		CHECK(err)
		// log.Printf("fsm=%v Catch seqnum %v get log %v", fsm, seqNum, condLogEntry)
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
			pendings[jumpSeqNum] = append(pendings[jumpSeqNum], condLogEntry)
		}
		seqNum = condLogEntry.SeqNum + 1
	}
}

// return:
//
//	resolved/rejected: true
//	pending: false
func (fsm *FsmCommon[TLogEntry]) resolve(env *Env, condLogEntry *types.CondLogEntry) (bool, uint64) {
	log.Printf("fsm=%+v resolve log %+v", fsm, condLogEntry)
	// resolve deps
	for _, dep := range condLogEntry.Deps {
		depCondLogEntry, err := env.FaasEnv.AsyncSharedLogRead(env.FaasCtx, dep)
		CHECK(err)
		log.Printf("fsm=%+v resolve log dep %+v get deplog %+v", fsm, dep, depCondLogEntry)

		var logState LogState
		if len(depCondLogEntry.AuxData) > 0 {
			decoded, err := snappy.Decode(nil, depCondLogEntry.AuxData)
			CHECK(err)
			if len(decoded) > 0 {
				err = json.Unmarshal(decoded, &logState)
				CHECK(err)
			} else {
				logState = LogState{LogEntryState_PENDING}
			}
		} else {
			logState = LogState{LogEntryState_PENDING}
		}

	recheck:
		switch logState.State {
		case LogEntryState_PENDING:
			if inside(fsm.reciever.GetTag(), depCondLogEntry.Tags) && depCondLogEntry.SeqNum > condLogEntry.SeqNum {
				// to pending
				return false, depCondLogEntry.SeqNum
			} else if inside(fsm.reciever.GetTag(), depCondLogEntry.Tags) && depCondLogEntry.SeqNum < condLogEntry.SeqNum {
				log.Println(dep.State)
				panic("impossible here that deps.state is pending")
			} else { // if !inside(IntentStepStreamTag(env.InstanceId), depCondLogEntry.Tags) {
				logState = ResolveLog(env, depCondLogEntry)
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
	if ok := fsm.reciever.ApplyLog(condLogEntry); ok {
		// log.Printf("ApplyLog %v, set aux applied", condLogEntry)
		serializedData, err := json.Marshal(LogState{LogEntryState_APPLIED})
		CHECK(err)
		encoded := snappy.Encode(nil, serializedData)
		err = env.FaasEnv.SharedLogSetAuxData(env.FaasCtx, condLogEntry.SeqNum, encoded)
		CHECK(err)
	} else {
		// log.Printf("ApplyLog %v, set aux discarded", condLogEntry)
		serializedData, err := json.Marshal(LogState{LogEntryState_DISCARDED})
		CHECK(err)
		encoded := snappy.Encode(nil, serializedData)
		err = env.FaasEnv.SharedLogSetAuxData(env.FaasCtx, condLogEntry.SeqNum, encoded)
		CHECK(err)
	}
	// resolved/rejected
	return true, 0
}

func inside(a uint64, b []uint64) bool {
	for _, i := range b {
		if a == i {
			return true
		}
	}
	return false
}

func ResolveLog(env *Env, logEntry *types.CondLogEntry) LogState {
	tag := logEntry.Tags[0]
	if tag == IntentLogTag {
		panic("impossible that a step depend to a intent")
	}
	// log.Printf("ResolveLog %v catch begin", logEntry)
	tagBuildMeta := logEntry.TagBuildMeta[0] // only 1 in bokiflow
	fsm := env.FsmHub.GetOrCreateAbsFsm(tagBuildMeta.FsmType, tagBuildMeta.TagKeys...)
	fsm.Catch(env)
	// log.Printf("ResolveLog %v catch over", logEntry)

	condLogEntry, err := env.FaasEnv.AsyncSharedLogReadNext(env.FaasCtx, tag, logEntry.SeqNum)
	CHECK(err)
	// log.Printf("Catched. get log %v of seqnum %v", condLogEntry, logEntry.SeqNum)
	if condLogEntry == nil {
		panic("impossible")
	}
	var logState LogState
	decoded, err := snappy.Decode(nil, condLogEntry.AuxData)
	CHECK(errors.Wrapf(err, "invalid aux data for log: %+v", condLogEntry))
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
