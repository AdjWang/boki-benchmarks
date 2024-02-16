package cayonlib

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"cs.utexas.edu/zjia/faas/types"
	"github.com/pkg/errors"
)

type LogEntry struct {
	SeqNum uint64
	Data   map[string]interface{}
}

type Env struct {
	LambdaId    string
	InstanceId  string
	StepNumber  int32
	Input       interface{}
	TxnId       string
	Instruction string
	// Baseline    bool		// unused
	FaasCtx context.Context
	FaasEnv types.Environment
	FsmHub  FsmHub
	// Step flow
	AsyncLogCtx AsyncLogContext
}

// All operations should be thread safe
type AsyncLogContext interface {
	GetLastStepLocalId() uint64
	// used by other async logs
	ChainFuture(futureLocalId uint64) AsyncLogContext
	// only and must be used by async step logs
	ChainStep(stepFutureLocalId uint64) AsyncLogContext
	Sync(timeout time.Duration) error
	Serialize() ([]byte, error)

	// DEBUG
	String() string
}

const (
	FsmType_STEPSTREAM        uint8 = 0
	FsmType_INTENTLOG         uint8 = 1
	FsmType_TRANSACTIONSTREAM uint8 = 2
	FsmType_LOCKSTREAM        uint8 = 3
)

type FsmHub interface {
	GetOrCreateAbsFsm(tag types.Tag) Fsm // Abs: Abstract
	GetInstanceStepFsm() *IntentFsm
	GetOrCreateLockFsm(bokiTag uint64) *LockFsm
	StoreBackLockFsm(fsm *LockFsm)
	GetOrCreateTxnFsm(bokiTag uint64) *TxnFsm
	StoreBackTxnFsm(fsm *TxnFsm)
}
type FsmHubImpl struct {
	env *Env
	// raw fsms required by bokiflow apps
	stepFsm       *IntentFsm
	lockFsms      map[uint64]*LockFsm
	lockFsmsMutex sync.RWMutex
	txnFsms       map[uint64]*TxnFsm
	txnFsmsMutex  sync.RWMutex
	// abstracted fsms required by dependency tracking
	absFsms map[uint64]Fsm
}

// DEBUG
func (fsmhub *FsmHubImpl) String() string {
	clientStr := fmt.Sprintf("LambdaId=%v, InstanceId=%v, StepNumber=%v\n",
		fsmhub.env.LambdaId, fsmhub.env.InstanceId, fsmhub.env.StepNumber)
	stepFsmStr := fmt.Sprintf("stepFsm: %v", fsmhub.stepFsm.DebugGetLogReadOrder())
	lockFsmStr := ""
	for lockId, lockFsm := range fsmhub.lockFsms {
		lockFsmStr += fmt.Sprintf("lockFsm_%v: %v", lockId, lockFsm.DebugGetLogReadOrder())
	}
	txnFsmStr := ""
	for txnId, txnFsm := range fsmhub.txnFsms {
		txnFsmStr += fmt.Sprintf("txnFsm_%v: %v", txnId, txnFsm.DebugGetLogReadOrder())
	}
	absFsmStr := ""
	for absId, absFsm := range fsmhub.absFsms {
		absFsmStr += fmt.Sprintf("absFsm_%v: %v", absId, absFsm.DebugGetLogReadOrder())
	}
	return clientStr + stepFsmStr + lockFsmStr + txnFsmStr + absFsmStr
}

func NewFsmHub(env *Env) FsmHub {
	stepFsm := NewIntentFsm(IntentStepStreamTag(env.InstanceId))
	stepFsm.Catch(env)
	return &FsmHubImpl{
		env:           env,
		stepFsm:       stepFsm,
		lockFsms:      make(map[uint64]*LockFsm),
		lockFsmsMutex: sync.RWMutex{},
		txnFsms:       make(map[uint64]*TxnFsm),
		txnFsmsMutex:  sync.RWMutex{},
		absFsms:       make(map[uint64]Fsm),
	}
}

// Fsm is only used by depdency tracking, Env should use fsm raw allocators
func (fsmhub *FsmHubImpl) GetOrCreateAbsFsm(tag types.Tag) Fsm {
	fsmType, bokiTag := tag.StreamType, tag.StreamId
	switch fsmType {
	case FsmType_STEPSTREAM:
		if fsmhub.stepFsm.bokiTag == tag.StreamId {
			return fsmhub.stepFsm
		} else {
			if fsm, ok := fsmhub.absFsms[bokiTag]; ok {
				return fsm
			} else {
				newAbsFsm := NewIntentFsm(bokiTag)
				fsmhub.absFsms[bokiTag] = newAbsFsm
				return newAbsFsm
			}
		}
	case FsmType_INTENTLOG:
		// not necessary for now
		panic("not implemented")
	case FsmType_TRANSACTIONSTREAM:
		return fsmhub.GetOrCreateTxnFsm(bokiTag)
	case FsmType_LOCKSTREAM:
		return fsmhub.GetOrCreateLockFsm(bokiTag)
	default:
		panic("unreachable")
	}
}

func (fsmhub *FsmHubImpl) GetInstanceStepFsm() *IntentFsm {
	return fsmhub.stepFsm
}

func (fsmhub *FsmHubImpl) GetOrCreateLockFsm(bokiTag uint64) *LockFsm {
	fsmhub.lockFsmsMutex.RLock()
	fsm, exists := fsmhub.lockFsms[bokiTag]
	fsmhub.lockFsmsMutex.RUnlock()
	if exists {
		return fsm
	} else {
		return NewLockFsm(bokiTag)
	}
}

func (fsmhub *FsmHubImpl) StoreBackLockFsm(fsm *LockFsm) {
	fsmhub.lockFsmsMutex.Lock()
	current, exists := fsmhub.lockFsms[fsm.bokiTag]
	if !exists || current.GetTailSeqNum() < fsm.GetTailSeqNum() {
		fsmhub.lockFsms[fsm.bokiTag] = fsm
	}
	fsmhub.lockFsmsMutex.Unlock()
}

func (fsmhub *FsmHubImpl) GetOrCreateTxnFsm(bokiTag uint64) *TxnFsm {
	fsmhub.txnFsmsMutex.RLock()
	fsm, exists := fsmhub.txnFsms[bokiTag]
	fsmhub.txnFsmsMutex.RUnlock()
	if exists {
		return fsm
	} else {
		return NewTxnFsm(bokiTag)
	}
}

func (fsmhub *FsmHubImpl) StoreBackTxnFsm(fsm *TxnFsm) {
	fsmhub.txnFsmsMutex.Lock()
	current, exists := fsmhub.lockFsms[fsm.bokiTag]
	if !exists || current.GetTailSeqNum() < fsm.GetTailSeqNum() {
		fsmhub.txnFsms[fsm.bokiTag] = fsm
	}
	fsmhub.txnFsmsMutex.Unlock()
}

// Implement AsyncLogContext
type asyncLogContextImpl struct {
	faasEnv types.Environment // delegation
	mu      sync.Mutex

	// serializables
	AsyncLogOps     []uint64 // local ids
	LastStepLocalId uint64   // local id
}

func DebugNewAsyncLogContext() AsyncLogContext {
	return &asyncLogContextImpl{
		faasEnv:         nil,
		mu:              sync.Mutex{},
		AsyncLogOps:     make([]uint64, 0, 20),
		LastStepLocalId: types.InvalidLocalId,
	}
}
func NewAsyncLogContext(faasEnv types.Environment) AsyncLogContext {
	return &asyncLogContextImpl{
		faasEnv:         faasEnv,
		mu:              sync.Mutex{},
		AsyncLogOps:     make([]uint64, 0, 20),
		LastStepLocalId: types.InvalidLocalId,
	}
}

func (fc *asyncLogContextImpl) GetLastStepLocalId() uint64 {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.LastStepLocalId
}

func (fc *asyncLogContextImpl) ChainFuture(futureLocalId uint64) AsyncLogContext {
	fc.mu.Lock()
	fc.AsyncLogOps = append(fc.AsyncLogOps, futureLocalId)
	fc.mu.Unlock()
	return fc
}

func (fc *asyncLogContextImpl) ChainStep(stepFutureLocalId uint64) AsyncLogContext {
	fc.mu.Lock()
	fc.AsyncLogOps = append(fc.AsyncLogOps, stepFutureLocalId)
	fc.LastStepLocalId = stepFutureLocalId
	fc.mu.Unlock()
	return fc
}

// Sync relies on the blocking read index to ensure durability.
// Note that indices are propagated from the storage node.
func (fc *asyncLogContextImpl) Sync(timeout time.Duration) error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	fc.mu.Lock()
	groupDeps := types.GroupLocalIdByEngine(fc.AsyncLogOps)
	fc.mu.Unlock()
	// log.Printf("[DEBUG] n=%d nGroup=%d deps=%v", len(deps), len(groupDeps), deps)
	// for engineId, subDeps := range groupDeps {
	for _, subDeps := range groupDeps {
		maxId := uint64(0)
		for _, id := range subDeps {
			if maxId < id {
				maxId = id
			}
		}
		// log.Printf("[DEBUG] engineId=%016X, max=%016X", engineId, maxId)
		// DEBUG: print dep graph edges
		// log.Printf("[DEBUG] sync log id=%016X", maxId)

		if _, err := fc.faasEnv.AsyncSharedLogReadIndex(ctx, maxId); err != nil {
			return err
		}
	}
	// DEBUG: print dep graph edges
	// log.Printf("[DEBUG] sync log ids=%s", types.IdsToString(fc.AsyncLogOps))

	fc.mu.Lock()
	fc.AsyncLogOps = make([]uint64, 0, 100)
	fc.mu.Unlock()
	return nil
}

func (fc *asyncLogContextImpl) Serialize() ([]byte, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	asyncLogCtxData, err := json.Marshal(fc)
	if err != nil {
		return nil, err
	}
	// DEBUG
	if len(asyncLogCtxData) == 0 {
		panic(fmt.Sprintf("empty asyncLogCtxData: %v", asyncLogCtxData))
	}
	return asyncLogCtxData, nil
}

func DeserializeAsyncLogContext(faasEnv types.Environment, data []byte) (AsyncLogContext, error) {
	asyncLogOps, lastStep, err := DeserializeRawAsyncLogContext(data)
	if err != nil {
		return nil, err
	}
	return &asyncLogContextImpl{
		faasEnv:         faasEnv,
		mu:              sync.Mutex{},
		AsyncLogOps:     asyncLogOps,
		LastStepLocalId: lastStep,
	}, nil
}

func DeserializeRawAsyncLogContext(data []byte) ([]uint64, uint64, error) {
	var asyncLogCtxPropagator asyncLogContextImpl
	err := json.Unmarshal(data, &asyncLogCtxPropagator)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "unresolvable data: %v", data)
	}
	return asyncLogCtxPropagator.AsyncLogOps, asyncLogCtxPropagator.LastStepLocalId, nil
}

// DEBUG
func (fc *asyncLogContextImpl) String() string {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fmt.Sprintf("log chain: %s, last log: %016X", types.IdsToString(fc.AsyncLogOps), fc.LastStepLocalId)
}
