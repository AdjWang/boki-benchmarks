package cayonlib

import (
	"encoding/json"
	"fmt"
	"log"
	"math"

	"cs.utexas.edu/zjia/faas/types"
	"github.com/golang/snappy"
)

const (
	LogState_PENDING   uint8 = 0
	LogState_APPLIED   uint8 = 1
	LogState_DISCARDED uint8 = 2
)

type LogState struct {
	State uint8 `json:"state"`
}

type Fsm interface {
	Catch(env *Env)
	GetLogState(seqNum uint64) (LogState, bool)

	// DEBUG
	DebugGetLogReadOrder() string
}
type FsmReceiver interface {
	ApplyLog(log *types.CondLogEntry) bool
	GetTag() uint64
	GetTailSeqNum() uint64
}

// Implement Fsm
type FsmCommon[TLogEntry any] struct {
	receiver          FsmReceiver
	tail              *TLogEntry
	resolvedLogStatus map[uint64]LogState

	// DEBUG: check multi-client reading order consistency
	resolvingSeq []uint64
	applyingSeq  []uint64
	appliedSeq   []uint64
}

func NewEmptyFsmCommon[TLogEntry any]() FsmCommon[TLogEntry] {
	return FsmCommon[TLogEntry]{
		receiver:          nil,
		tail:              nil,
		resolvedLogStatus: make(map[uint64]LogState),

		// DEBUG
		resolvingSeq: make([]uint64, 0, 100),
		applyingSeq:  make([]uint64, 0, 100),
		appliedSeq:   make([]uint64, 0, 100),
	}
}

func (fsm *FsmCommon[TLogEntry]) DebugGetLogReadOrder() string {
	return fmt.Sprintf("tag=%v, resolvingSeq=%v, applyingSeq=%v, appliedSeq=%v\n",
		fsm.receiver.GetTag(), fsm.resolvingSeq, fsm.applyingSeq, fsm.appliedSeq)
}

func isResolvedOrRejected(logState LogState) bool {
	return logState.State == LogState_APPLIED ||
		logState.State == LogState_DISCARDED
}

func (fsm *FsmCommon[TLogEntry]) resolveWithPendings(env *Env, condLogEntry *types.CondLogEntry,
	handlePendings bool,
	pendings map[uint64][]*types.CondLogEntry /*ref*/) {

	// log.Printf("[DEBUG] resolveWithPendings logEntry=%+v, handlePendings=%v, pendings=%+v",
	// 	condLogEntry, handlePendings, pendings)

	currentSeqNum := condLogEntry.SeqNum
	if logState, jumpSeqNum := fsm.doResolve(env, condLogEntry); isResolvedOrRejected(logState) {
		// check if any available pending logs can be handled after a log is
		// resolved or rejected
		fsm.resolvedLogStatus[currentSeqNum] = logState
		// DEBUG
		fsm.resolvingSeq = append(fsm.resolvingSeq, currentSeqNum)

		// only handle pending logs after a new log is resolved or rejected
		if !handlePendings {
			// now handle pending logs
			if pendingLogs, ok := pendings[currentSeqNum]; ok {
				for _, log := range pendingLogs {
					fsm.resolveWithPendings(env, log, true /*handlePendings*/, pendings /*ref*/)
				}
				delete(pendings, currentSeqNum)
			}
		}
	} else {
		// DEBUG
		log.Printf("[DEBUG] pending log=%+v", condLogEntry)

		if jumpSeqNum <= currentSeqNum {
			panic("unreachable")
		}
		// put off resolving until the dependent seqnum is met
		if _, ok := pendings[jumpSeqNum]; !ok {
			pendings[jumpSeqNum] = make([]*types.CondLogEntry, 0, 10)
		}
		pendings[jumpSeqNum] = append(pendings[jumpSeqNum], condLogEntry)
	}
}

func (fsm *FsmCommon[TLogEntry]) Catch(env *Env) {
	ASSERT(fsm.receiver != nil, "set receiver to fsm object self")

	tag := fsm.receiver.GetTag()
	seqNum := uint64(0)
	if fsm.tail != nil {
		seqNum = fsm.receiver.GetTailSeqNum() + 1
	}
	pendings := make(map[uint64][]*types.CondLogEntry)
	for {
		condLogEntry, err := env.FaasEnv.AsyncSharedLogReadNext(env.FaasCtx, tag, seqNum)
		CHECK(err)
		// log.Printf("[DEBUG] fsm=%v Catch seqnum %v get log %+v", reflect.TypeOf(fsm.receiver), seqNum, condLogEntry)
		if condLogEntry == nil {
			break
		}
		fsm.resolveWithPendings(env, condLogEntry, false /*handlePendings*/, pendings /*ref*/)
		seqNum = condLogEntry.SeqNum + 1
	}
	ASSERT(len(pendings) == 0, fmt.Sprintf("all pendings are expected to be resolved, existing: %+v", pendings))
}

func (fsm *FsmCommon[TLogEntry]) GetLogState(seqNum uint64) (LogState, bool) {
	if logState, ok := fsm.resolvedLogStatus[seqNum]; ok {
		return logState, true
	} else {
		return LogState{math.MaxUint8}, false
	}
}

func inside(a uint64, b []uint64) bool {
	for _, i := range b {
		if a == i {
			return true
		}
	}
	return false
}

func setLogStateAuxData(env *Env, seqnum uint64, logState LogState) {
	serializedData, err := json.Marshal(logState)
	CHECK(err)
	encoded := snappy.Encode(nil, serializedData)
	err = env.FaasEnv.SharedLogSetAuxData(env.FaasCtx, seqnum, encoded)
	CHECK(err)
}

func getLogStateAuxData(auxData []byte) LogState {
	var logState LogState
	if len(auxData) > 0 {
		decoded, err := snappy.Decode(nil, auxData)
		CHECK(err)
		if len(decoded) > 0 {
			err = json.Unmarshal(decoded, &logState)
			CHECK(err)
		} else {
			logState = LogState{LogState_PENDING}
		}
	} else {
		logState = LogState{LogState_PENDING}
	}
	return logState
}

// Return: resolved state, put off target seqnum
// the put off target seqnum only set when LogState is PENDING
func (fsm *FsmCommon[TLogEntry]) doResolve(env *Env, condLogEntry *types.CondLogEntry) (LogState, uint64) {
	// log.Printf("fsm=%v resolve log %+v", reflect.TypeOf(fsm.receiver), condLogEntry)
	// 1. resolve deps
	for _, depLogLocalId := range condLogEntry.Deps {
		if !types.IsLocalIdValid(depLogLocalId) {
			// invalid local id is used as the initial dep now, just ignore it
			continue
		}
		depLogEntry, err := env.FaasEnv.AsyncSharedLogRead(env.FaasCtx, depLogLocalId)
		CHECK(err)
		// log.Printf("fsm=%v resolve log dep 0x%064X get deplog %+v", reflect.TypeOf(fsm.receiver), depLogLocalId, depLogEntry)

		// treat not resolved log as pending
		logState := getLogStateAuxData(depLogEntry.AuxData)
		if logState.State == LogState_PENDING {
			if inside(fsm.receiver.GetTag(), depLogEntry.Tags) {
				if depLogEntry.SeqNum > condLogEntry.SeqNum {
					// deps will come later, put off resolving
					// do nothing to fallthrough to switch
				} else if depLogEntry.SeqNum < condLogEntry.SeqNum {
					log.Printf("[DEBUG] invalid dep log: %+v", depLogEntry)
					panic("unreachable that deps should have been handled")
				} else {
					// a log cannot depend on itself
					panic("unreachable")
				}
			} else {
				// delegate resolving to the target fsm
				// A potential issue here is circular dependency, which causes infinite loop. Becareful!
				if ok := ResolveLog(env, depLogEntry.Tags, depLogEntry.TagBuildMetas, depLogEntry.SeqNum); ok {
					logState = LogState{LogState_APPLIED}
				} else {
					logState = LogState{LogState_DISCARDED}
				}
			}
		}

		switch logState.State {
		case LogState_PENDING:
			// deps will come later, put off resolving
			return LogState{LogState_PENDING}, depLogEntry.SeqNum
		case LogState_APPLIED:
			// handle next dep
			continue
		case LogState_DISCARDED:
			setLogStateAuxData(env, condLogEntry.SeqNum, LogState{LogState_DISCARDED})
			return LogState{LogState_DISCARDED}, 0
		default:
			panic("unreachable")
		}
	}
	// 2. check conds
	// from here the deps are all solved, condLogEntry.State must be PENDING
	// DEBUG
	fsm.applyingSeq = append(fsm.applyingSeq, condLogEntry.SeqNum)

	if ok := fsm.receiver.ApplyLog(condLogEntry); ok {
		// DEBUG
		fsm.appliedSeq = append(fsm.appliedSeq, condLogEntry.SeqNum)

		setLogStateAuxData(env, condLogEntry.SeqNum, LogState{LogState_APPLIED})
		return LogState{LogState_APPLIED}, 0
	} else {
		setLogStateAuxData(env, condLogEntry.SeqNum, LogState{LogState_DISCARDED})
		return LogState{LogState_DISCARDED}, 0
	}
}

// Return:
// - true: applied
// - false: discarded
func ResolveLog(env *Env, tags []uint64, tagBuildMetas []types.TagMeta, seqNum uint64) bool {
	tag := tags[0] // only 1 in bokiflow
	if tag == IntentLogTag {
		panic("unreachable that a step is designed to never depend on an intent")
	}
	tagBuildMeta := tagBuildMetas[0] // only 1 in bokiflow
	// log.Printf("ResolveLog tagMeta: %+v seqNum: %v catch begin", tagBuildMeta, seqNum)
	fsm := env.FsmHub.GetOrCreateAbsFsm(tagBuildMeta.FsmType, tagBuildMeta.TagKeys...)
	fsm.Catch(env)

	logState, ok := fsm.GetLogState(seqNum)
	ASSERT(ok, fmt.Sprintf("log %v should have been resolved", seqNum))
	// log.Printf("ResolveLog catch over, get log state: %+v", logState)

	if logState.State == LogState_APPLIED {
		return true
	} else if logState.State == LogState_DISCARDED {
		return false
	} else {
		// Catch(env) -> resolved -> cannot be PENDING now
		panic("unreachable")
	}
}
