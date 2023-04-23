package cayonlib

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

type AsyncLogContext interface {
	GetLastStepLogMeta() types.FutureMeta
	// used by other async logs
	ChainFuture(future types.FutureMeta) AsyncLogContext
	// only and must be used by async step logs
	ChainStep(stepFuture types.FutureMeta) AsyncLogContext
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
	GetOrCreateAbsFsm(fsmType uint8, tagKeys ...string) Fsm // Abs: Abstract
	GetInstanceStepFsm() *IntentFsm
	GetOrCreateLockFsm(lockId string) *LockFsm
	StoreBackLockFsm(fsm *LockFsm)
	GetOrCreateTxnFsm(lambdaId string, txnId string) *TxnFsm
	StoreBackTxnFsm(fsm *TxnFsm)
}
type FsmHubImpl struct {
	env *Env
	// raw fsms required by bokiflow apps
	stepFsm       *IntentFsm
	lockFsms      map[string]*LockFsm
	lockFsmsMutex sync.RWMutex
	txnFsms       map[string]*TxnFsm
	txnFsmsMutex  sync.RWMutex
	// abstracted fsms required by depdency tracking
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
	stepFsm := NewIntentFsm(env.InstanceId)
	stepFsm.Catch(env)
	return &FsmHubImpl{
		env:           env,
		stepFsm:       stepFsm,
		lockFsms:      make(map[string]*LockFsm),
		lockFsmsMutex: sync.RWMutex{},
		txnFsms:       make(map[string]*TxnFsm),
		txnFsmsMutex:  sync.RWMutex{},
		absFsms:       make(map[uint64]Fsm),
	}
}

// Fsm is only used by depdency tracking, Env should use fsm raw allocators
func (fsmhub *FsmHubImpl) GetOrCreateAbsFsm(fsmType uint8, tagKeys ...string) Fsm {
	assertTagKeysNum := func(fsmType uint8, lenTagKeys int, requiredNum int) {
		if len(tagKeys) != requiredNum {
			panic(fmt.Sprintf("invalid fsm: %v tag keys: %v, requiring %v",
				fsmType, strings.Join(tagKeys, ", "), requiredNum))
		}
	}

	switch fsmType {
	case FsmType_STEPSTREAM:
		assertTagKeysNum(fsmType, len(tagKeys), 1)
		if fsmhub.stepFsm.instanceId == tagKeys[0] {
			return fsmhub.stepFsm
		} else {
			if fsm, ok := fsmhub.absFsms[IntentStepStreamTag(tagKeys[0])]; ok {
				return fsm
			} else {
				newAbsFsm := NewIntentFsm(tagKeys[0])
				fsmhub.absFsms[IntentStepStreamTag(tagKeys[0])] = newAbsFsm
				return newAbsFsm
			}
		}
	case FsmType_INTENTLOG:
		assertTagKeysNum(fsmType, len(tagKeys), 1)
		// not necessary for now
		panic("not implemented")
	case FsmType_TRANSACTIONSTREAM:
		assertTagKeysNum(fsmType, len(tagKeys), 2)
		return fsmhub.GetOrCreateTxnFsm(tagKeys[0], tagKeys[1])
	case FsmType_LOCKSTREAM:
		assertTagKeysNum(fsmType, len(tagKeys), 1)
		return fsmhub.GetOrCreateLockFsm(tagKeys[0])
	default:
		panic("unreachable")
	}
}

func (fsmhub *FsmHubImpl) GetInstanceStepFsm() *IntentFsm {
	return fsmhub.stepFsm
}

func (fsmhub *FsmHubImpl) GetOrCreateLockFsm(lockId string) *LockFsm {
	fsmhub.lockFsmsMutex.RLock()
	fsm, exists := fsmhub.lockFsms[lockId]
	fsmhub.lockFsmsMutex.RUnlock()
	if exists {
		return fsm
	} else {
		return NewLockFsm(lockId)
	}
}

func (fsmhub *FsmHubImpl) StoreBackLockFsm(fsm *LockFsm) {
	fsmhub.lockFsmsMutex.Lock()
	current, exists := fsmhub.lockFsms[fsm.lockId]
	if !exists || current.GetTailSeqNum() < fsm.GetTailSeqNum() {
		fsmhub.lockFsms[fsm.lockId] = fsm
	}
	fsmhub.lockFsmsMutex.Unlock()
}

func (fsmhub *FsmHubImpl) GetOrCreateTxnFsm(lambdaId string, txnId string) *TxnFsm {
	fsmhub.txnFsmsMutex.RLock()
	fsm, exists := fsmhub.txnFsms[lambdaId+"-"+txnId]
	fsmhub.txnFsmsMutex.RUnlock()
	if exists {
		return fsm
	} else {
		return NewTxnFsm(lambdaId, txnId)
	}
}

func (fsmhub *FsmHubImpl) StoreBackTxnFsm(fsm *TxnFsm) {
	fsmhub.txnFsmsMutex.Lock()
	current, exists := fsmhub.lockFsms[fsm.LambdaId+"-"+fsm.TxnId]
	if !exists || current.GetTailSeqNum() < fsm.GetTailSeqNum() {
		fsmhub.txnFsms[fsm.LambdaId+"-"+fsm.TxnId] = fsm
	}
	fsmhub.txnFsmsMutex.Unlock()
}

type asyncLogContextImpl struct {
	faasEnv types.Environment // delegation
	mu      sync.Mutex

	// serializables
	AsyncLogOps     []types.FutureMeta
	LastStepLogMeta types.FutureMeta
}

func DebugNewAsyncLogContext() AsyncLogContext {
	return &asyncLogContextImpl{
		faasEnv:         nil,
		mu:              sync.Mutex{},
		AsyncLogOps:     make([]types.FutureMeta, 0, 20),
		LastStepLogMeta: types.InvalidFutureMeta,
	}
}
func NewAsyncLogContext(faasEnv types.Environment) AsyncLogContext {
	return &asyncLogContextImpl{
		faasEnv:         faasEnv,
		mu:              sync.Mutex{},
		AsyncLogOps:     make([]types.FutureMeta, 0, 20),
		LastStepLogMeta: types.InvalidFutureMeta,
	}
}

func (fc *asyncLogContextImpl) GetLastStepLogMeta() types.FutureMeta {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.LastStepLogMeta
}

func (fc *asyncLogContextImpl) ChainFuture(future types.FutureMeta) AsyncLogContext {
	fc.mu.Lock()
	fc.AsyncLogOps = append(fc.AsyncLogOps, future)
	fc.mu.Unlock()
	return fc
}

func (fc *asyncLogContextImpl) ChainStep(stepFuture types.FutureMeta) AsyncLogContext {
	fc.mu.Lock()
	fc.AsyncLogOps = append(fc.AsyncLogOps, stepFuture)
	fc.LastStepLogMeta = stepFuture
	fc.mu.Unlock()
	return fc
}

func (fc *asyncLogContextImpl) Sync(timeout time.Duration) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	wg := sync.WaitGroup{}
	errCh := make(chan error)
	for _, futureMeta := range fc.AsyncLogOps {
		wg.Add(1)
		go func(ctx context.Context, futureMeta types.FutureMeta) {
			if _, err := fc.faasEnv.AsyncSharedLogReadIndex(ctx, futureMeta); err != nil {
				errCh <- errors.Wrapf(err, "failed to read index for future: %+v", futureMeta)
			} else {
				// log.Printf("wait future=%+v done", future)
				// seqNum, err := future.GetResult()
				// log.Printf("wait futureMeta.LocalId=0x%016X state=%v seqNum=0x%016X err=%v",
				// 	futureMeta.LocalId, futureMeta.State, seqNum, err)
			}
			wg.Done()
		}(ctx, futureMeta)
	}
	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		waitCh <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		// log.Println("wait future all done timeout")
		return ctx.Err()
	case err := <-errCh:
		// log.Println("wait future all done with error:", err)
		return err
	case <-waitCh:
		// log.Println("wait future all done without error")
		// clear synchronized logs
		fc.AsyncLogOps = make([]types.FutureMeta, 0, 100)
		return nil
	}
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
	var asyncLogCtxPropagator asyncLogContextImpl
	err := json.Unmarshal(data, &asyncLogCtxPropagator)
	if err != nil {
		return nil, errors.Wrapf(err, "unresolvable data: %v", data)
	}
	return &asyncLogContextImpl{
		faasEnv:         faasEnv,
		mu:              sync.Mutex{},
		AsyncLogOps:     asyncLogCtxPropagator.AsyncLogOps,
		LastStepLogMeta: asyncLogCtxPropagator.LastStepLogMeta,
	}, nil
}

// DEBUG
func (fc *asyncLogContextImpl) String() string {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fmt.Sprintf("log chain: %+v, last log: %+v", fc.AsyncLogOps, fc.LastStepLogMeta)
}
