package cayonlib

import (
	"encoding/json"
	"fmt"
	"log"

	"cs.utexas.edu/zjia/faas/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/golang/snappy"
	// "github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

type LockLogEntry struct {
	SeqNum     uint64 `json:"-"`
	LockId     string `json:"lockId"`
	StepNumber int32  `json:"step"`
	UnlockOp   bool   `json:"unlockOp"`
	Holder     string `json:"holder"`
}

type LockFsm struct {
	stepNumber int32
	FsmCommon[LockLogEntry]
}

func NewLockFsm(bokiTag uint64) *LockFsm {
	this := &LockFsm{
		stepNumber: 0,
		FsmCommon:  NewEmptyFsmCommon[LockLogEntry](bokiTag),
	}
	this.receiver = this
	return this
}

func (fsm *LockFsm) ApplyLog(logEntry *types.LogEntryWithMeta) bool {
	decoded, err := snappy.Decode(nil, logEntry.Data)
	CHECK(err)
	var lockLog LockLogEntry
	err = json.Unmarshal(decoded, &lockLog)
	CHECK(err)
	if LockStreamTag(lockLog.LockId) == fsm.bokiTag && lockLog.StepNumber == fsm.stepNumber {
		// log.Printf("[INFO] Found my log: seqnum=%d, step=%d", logEntry.SeqNum, lockLog.StepNumber)
		lockLog.SeqNum = logEntry.SeqNum
		if lockLog.UnlockOp {
			if fsm.tail == nil || fsm.tail.UnlockOp || fsm.tail.Holder != lockLog.Holder {
				panic(fmt.Sprintf("Invalid Unlock op for lock %s and holder %s", lockLog.LockId, lockLog.Holder))
			}
		} else {
			if fsm.tail != nil && !fsm.tail.UnlockOp {
				panic(fmt.Sprintf("Invalid Lock op for lock %s and holder %s", lockLog.LockId, lockLog.Holder))
			}
		}
		fsm.tail = &lockLog
		fsm.stepNumber++
	}
	// Lock log entries has no conds, so always return true
	return true
}

func (fsm *LockFsm) GetTailSeqNum() uint64 {
	return fsm.tail.SeqNum
}

func (fsm *LockFsm) holder() string {
	if fsm.tail == nil || fsm.tail.UnlockOp {
		return ""
	} else {
		return fsm.tail.Holder
	}
}

func (fsm *LockFsm) Lock(env *Env, holder string) bool {
	fsm.Catch(env)
	currentHolder := fsm.holder()
	if currentHolder == holder {
		return true
	} else if currentHolder != "" {
		return false
	}
	LibSyncAppendLog(
		env,
		types.Tag{StreamType: FsmType_LOCKSTREAM, StreamId: fsm.bokiTag},
		&LockLogEntry{
			StepNumber: fsm.stepNumber,
			UnlockOp:   false,
			Holder:     holder,
		}, env.AsyncLogCtx.GetLastStepLocalId())

	fsm.Catch(env)
	return fsm.holder() == holder
}

func (fsm *LockFsm) Unlock(env *Env, holder string) {
	fsm.Catch(env)
	if fsm.holder() != holder {
		log.Printf("[WARN] %s is not the holder for lock tag %s", holder, fsm.bokiTag)
		return
	}
	LibSyncAppendLog(
		env,
		types.Tag{StreamType: FsmType_LOCKSTREAM, StreamId: fsm.bokiTag},
		&LockLogEntry{
			StepNumber: fsm.stepNumber,
			UnlockOp:   true,
			Holder:     holder,
		}, env.AsyncLogCtx.GetLastStepLocalId())
}

func Lock(env *Env, tablename string, key string) bool {
	lockId := fmt.Sprintf("%s-%s", tablename, key)
	fsm := env.FsmHub.GetOrCreateLockFsm(LockStreamTag(lockId))
	success := fsm.Lock(env, env.TxnId)
	env.FsmHub.StoreBackLockFsm(fsm)
	if !success {
		log.Printf("[WARN] Failed to lock %s with txn %s", lockId, env.TxnId)
	}
	return success
}

func Unlock(env *Env, tablename string, key string) {
	lockId := fmt.Sprintf("%s-%s", tablename, key)
	fsm := env.FsmHub.GetOrCreateLockFsm(LockStreamTag(lockId))
	fsm.Unlock(env, env.TxnId)
	env.FsmHub.StoreBackLockFsm(fsm)
}

type TxnLogEntry struct {
	SeqNum   uint64        `json:"-"`
	LambdaId string        `json:"lambdaId"`
	TxnId    string        `json:"txnId"`
	Callee   string        `json:"callee"`
	WriteOp  aws.JSONValue `json:"write"`
}

type TxnFsm struct {
	FsmCommon[TxnLogEntry]
	txnLogs map[uint64]*TxnLogEntry
}

func NewTxnFsm(bokiTag uint64) *TxnFsm {
	this := &TxnFsm{
		FsmCommon: NewEmptyFsmCommon[TxnLogEntry](bokiTag),
		txnLogs:   make(map[uint64]*TxnLogEntry),
	}
	this.receiver = this
	return this
}

func (fsm *TxnFsm) ApplyLog(logEntry *types.LogEntryWithMeta) bool {
	decoded, err := snappy.Decode(nil, logEntry.Data)
	CHECK(err)
	var txnLog TxnLogEntry
	err = json.Unmarshal(decoded, &txnLog)
	CHECK(err)
	if TransactionStreamTag(txnLog.LambdaId, txnLog.TxnId) == fsm.bokiTag {
		txnLog.SeqNum = logEntry.SeqNum
		fsm.txnLogs[txnLog.SeqNum] = &txnLog
	}
	// Txn log entries has no conds, so always return true
	return true
}

func (fsm *TxnFsm) GetTailSeqNum() uint64 {
	return fsm.tail.SeqNum
}

func (fsm *TxnFsm) GetAllTxnLogs() []*TxnLogEntry {
	result := make([]*TxnLogEntry, 0, len(fsm.txnLogs))
	for _, logEntry := range fsm.txnLogs {
		result = append(result, logEntry)
	}
	return result
}

func TPLRead(env *Env, tablename string, key string) (bool, interface{}) {
	if Lock(env, tablename, key) {
		return true, Read(env, tablename, key)
	} else {
		return false, nil
	}
}

func TPLWrite(env *Env, tablename string, key string, value aws.JSONValue) bool {
	if Lock(env, tablename, key) {
		env.AsyncLogCtx.ChainStep(
			LibAsyncAppendLog(
				env,
				types.Tag{
					StreamType: FsmType_TRANSACTIONSTREAM,
					StreamId:   TransactionStreamTag(env.LambdaId, env.TxnId),
				},
				&TxnLogEntry{
					LambdaId: env.LambdaId,
					TxnId:    env.TxnId,
					Callee:   "",
					WriteOp: aws.JSONValue{
						"tablename": tablename,
						"key":       key,
						"value":     value,
					},
				},
				env.AsyncLogCtx.GetLastStepLocalId(),
			).GetLocalId())
		return true
	} else {
		return false
	}
}

func BeginTxn(env *Env) {
	env.TxnId = env.InstanceId
	env.Instruction = "EXECUTE"
}

func CommitTxn(env *Env) {
	log.Printf("[INFO] Commit transaction %s", env.TxnId)
	env.Instruction = "COMMIT"
	TPLCommit(env)
	env.TxnId = ""
	env.Instruction = ""
}

func AbortTxn(env *Env) {
	log.Printf("[WARN] Abort transaction %s", env.TxnId)
	env.Instruction = "ABORT"
	TPLAbort(env)
	env.TxnId = ""
	env.Instruction = ""
}

func getAllTxnLogs(env *Env) []*TxnLogEntry {
	txnFsm := env.FsmHub.GetOrCreateTxnFsm(TransactionStreamTag(env.LambdaId, env.TxnId))
	txnFsm.Catch(env)
	return txnFsm.GetAllTxnLogs()
}

func TPLCommit(env *Env) {
	txnLogs := getAllTxnLogs(env)
	for _, txnLog := range txnLogs {
		if txnLog.Callee != "" {
			continue
		}
		tablename := txnLog.WriteOp["tablename"].(string)
		key := txnLog.WriteOp["key"].(string)
		update := map[expression.NameBuilder]expression.OperandBuilder{}
		for kk, vv := range txnLog.WriteOp["value"].(map[string]interface{}) {
			update[expression.Name(kk)] = expression.Value(vv)
		}
		Write(env, tablename, key, update)
		Unlock(env, tablename, key)
	}
	for _, txnLog := range txnLogs {
		if txnLog.Callee != "" {
			log.Printf("[INFO] Commit transaction %s for callee %s", env.TxnId, txnLog.Callee)
			SyncInvoke(env, txnLog.Callee, aws.JSONValue{})
		}
	}
}

func TPLAbort(env *Env) {
	txnLogs := getAllTxnLogs(env)
	for _, txnLog := range txnLogs {
		if txnLog.Callee != "" {
			continue
		}
		tablename := txnLog.WriteOp["tablename"].(string)
		key := txnLog.WriteOp["key"].(string)
		Unlock(env, tablename, key)
	}
	for _, txnLog := range txnLogs {
		if txnLog.Callee != "" {
			log.Printf("[INFO] Abort transaction %s for callee %s", env.TxnId, txnLog.Callee)
			SyncInvoke(env, txnLog.Callee, aws.JSONValue{})
		}
	}
}
