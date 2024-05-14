package cayonlib

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
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
	lockId     string
	tailSeqNum uint64
	stepNumber int32
	tail       *LockLogEntry
	// DEBUG
	applyConseq []LockLogEntry
}

var lockFsms = map[string]LockFsm{}
var lockFsmsMutex = sync.RWMutex{}

func getOrCreateLockFsm(lockId string) LockFsm {
	lockFsmsMutex.RLock()
	fsm, exists := lockFsms[lockId]
	lockFsmsMutex.RUnlock()
	if exists {
		log.Printf("[DEBUG] get lock fsm lockId=%s", lockId)
		return fsm
	} else {
		log.Printf("[DEBUG] create lock fsm lockId=%s", lockId)
		return LockFsm{
			lockId:      lockId,
			tailSeqNum:  uint64(0),
			stepNumber:  0,
			tail:        nil,
			applyConseq: make([]LockLogEntry, 0),
		}
	}
}

func storeBackLockFsm(fsm LockFsm) {
	lockFsmsMutex.Lock()
	current, exists := lockFsms[fsm.lockId]
	if !exists || current.tailSeqNum < fsm.tailSeqNum {
		log.Printf("[DEBUG] store back lock fsm lockId=%s", fsm.lockId)
		lockFsms[fsm.lockId] = fsm
	} else {
		log.Printf("[DEBUG] skip store back lock fsm lockId=%s (exists=%v tail seqnum current=%v fsm=%v)",
			fsm.lockId, exists, current.tailSeqNum, fsm.tailSeqNum)
	}
	lockFsmsMutex.Unlock()
}

func (fsm *LockFsm) catch(env *Env) {
	tag := LockStreamTag(fsm.lockId)
	for {
		logEntry, err := env.FaasEnv.SharedLogReadNext(env.FaasCtx, tag, fsm.tailSeqNum)
		CHECK(err)
		log.Printf("[DEBUG] lock fsm catch tag=%v seqnum=%v logEntry=%+v", tag, fsm.tailSeqNum, logEntry)
		if logEntry == nil {
			break
		}
		decoded, err := snappy.Decode(nil, logEntry.Data)
		CHECK(err)
		var lockLog LockLogEntry
		err = json.Unmarshal(decoded, &lockLog)
		CHECK(err)
		mine := (lockLog.LockId == fsm.lockId && lockLog.StepNumber == fsm.stepNumber)
		log.Printf("[DEBUG] Found log: tag=%v mine=%v seqnum=%d, step=%d", tag, mine, logEntry.SeqNum, lockLog.StepNumber)
		if lockLog.LockId == fsm.lockId && lockLog.StepNumber == fsm.stepNumber {
			fsm.applyConseq = append(fsm.applyConseq, lockLog)

			lockLog.SeqNum = logEntry.SeqNum
			if lockLog.UnlockOp {
				if fsm.tail == nil || fsm.tail.UnlockOp || fsm.tail.Holder != lockLog.Holder {
					panic(fmt.Sprintf("Invalid Unlock op for lock %s and holder %s", fsm.lockId, lockLog.Holder))
				}
			} else {
				if fsm.tail != nil && !fsm.tail.UnlockOp {
					panic(fmt.Sprintf("Invalid Lock op for lock %s and holder %s", fsm.lockId, lockLog.Holder))
				}
			}
			fsm.tail = &lockLog
			fsm.stepNumber++
		}
		fsm.tailSeqNum = logEntry.SeqNum + 1
	}
}

func (fsm *LockFsm) holder() string {
	if fsm.tail == nil || fsm.tail.UnlockOp {
		return ""
	} else {
		return fsm.tail.Holder
	}
}

func (fsm *LockFsm) Lock(env *Env, holder string) bool {
	fsm.catch(env)
	currentHolder := fsm.holder()
	if currentHolder == holder {
		return true
	} else if currentHolder != "" {
		return false
	}
	lockLogEntry := LockLogEntry{
		LockId:     fsm.lockId,
		StepNumber: fsm.stepNumber,
		UnlockOp:   false,
		Holder:     holder,
	}
	seqNum := LibAppendLog(env, LockStreamTag(fsm.lockId), &lockLogEntry)
	lockLogEntry.SeqNum = seqNum
	log.Printf("[DEBUG] Lock append lock tag=%v log entry=%+v", LockStreamTag(fsm.lockId), lockLogEntry)
	fsm.catch(env)
	if fsm.holder() == holder {
		env.SeqNum = seqNum
		return true
	}
	return false
}

func (fsm *LockFsm) Unlock(env *Env, holder string) {
	fsm.catch(env)
	if fsm.holder() != holder {
		log.Printf("[WARN] %s is not the holder for lock %s", holder, fsm.lockId)
		return
	}
	lockLogEntry := LockLogEntry{
		LockId:     fsm.lockId,
		StepNumber: fsm.stepNumber,
		UnlockOp:   true,
		Holder:     holder,
	}
	seqNum := LibAppendLog(env, LockStreamTag(fsm.lockId), &lockLogEntry)
	lockLogEntry.SeqNum = seqNum
	log.Printf("[DEBUG] Lock append lock tag=%v log entry=%+v", LockStreamTag(fsm.lockId), lockLogEntry)
	env.SeqNum = seqNum
}

func Lock(env *Env, tablename string, key string) bool {
	lockId := fmt.Sprintf("%s-%s", tablename, key)
	fsm := getOrCreateLockFsm(lockId)
	success := fsm.Lock(env, env.TxnId)
	storeBackLockFsm(fsm)
	if !success {
		log.Printf("[WARN] Failed to lock %s with txn %s", lockId, env.TxnId)
		log.Printf("[DEBUG] Lock failed fsm lock log entries=%v =====", len(fsm.applyConseq))
		for _, lockLogEntry := range fsm.applyConseq {
			log.Printf("[DEBUG] Lock failed fsm lock log entry=%+v", lockLogEntry)
		}
	}
	return success
}

func Unlock(env *Env, tablename string, key string) {
	lockId := fmt.Sprintf("%s-%s", tablename, key)
	fsm := getOrCreateLockFsm(lockId)
	fsm.Unlock(env, env.TxnId)
	storeBackLockFsm(fsm)
}

func TPLRead(env *Env, tablename string, key string) (bool, interface{}) {
	if Lock(env, tablename, key) {
		return true, Read(env, tablename, key)
	} else {
		return false, nil
	}
}

type TxnLogEntry struct {
	SeqNum   uint64        `json:"-"`
	LambdaId string        `json:"lambdaId"`
	TxnId    string        `json:"txnId"`
	Callee   string        `json:"callee"`
	WriteOp  aws.JSONValue `json:"write"`
}

func TPLWrite(env *Env, tablename string, key string, value aws.JSONValue) bool {
	if Lock(env, tablename, key) {
		tag := TransactionStreamTag(env.LambdaId, env.TxnId)
		LibAppendLog(env, tag, &TxnLogEntry{
			LambdaId: env.LambdaId,
			TxnId:    env.TxnId,
			Callee:   "",
			WriteOp: aws.JSONValue{
				"tablename": tablename,
				"key":       key,
				"value":     value,
			},
		})
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
