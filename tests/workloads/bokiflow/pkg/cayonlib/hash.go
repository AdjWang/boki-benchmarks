package cayonlib

import (
	"cs.utexas.edu/zjia/faas/types"
	"github.com/cespare/xxhash/v2"
)

const IntentLogTag uint64 = 1

func IntentLogTagMeta() []types.TagMeta {
	return []types.TagMeta{
		{
			FsmType: FsmType_INTENTLOG,
			TagKeys: []string{"1"},
		},
	}
}

const intentStepStreamLowBits uint64 = 2
const lockStreamLowBits uint64 = 3
const transactionStreamLowBits uint64 = 4

const (
	FsmType_STEPSTREAM        uint8 = 0
	FsmType_INTENTLOG         uint8 = 1
	FsmType_TRANSACTIONSTREAM uint8 = 2
	FsmType_LOCKSTREAM        uint8 = 3
)

// Factory is only used by depdency tracking, Env should use raw allocators
func GetOrCreateFsm(env *Env, fsmType uint8, tagKeys ...string) Fsm {
	switch fsmType {
	case FsmType_STEPSTREAM:
		if env.Fsm.instanceId == tagKeys[0] {
			return env.Fsm
		} else {
			return NewIntentFsm(tagKeys[0])
		}
	case FsmType_INTENTLOG:
		// TODO: implement Fsm for intentlog fsm
		panic("TODO")
	case FsmType_TRANSACTIONSTREAM:
		// TODO: implement Fsm for txn fsm
		panic("TODO")
	case FsmType_LOCKSTREAM:
		// TODO: implement Fsm for lock fsm
		// return NewLockFsm(tagKeys[0])
		panic("TODO")
	default:
		panic("unreachable")
	}
}

func IntentStepStreamTag(instanceId string) uint64 {
	h := xxhash.Sum64String(instanceId)
	tag := (h << 3) + intentStepStreamLowBits
	if tag == 0 || (^tag) == 0 {
		panic("Invalid tag")
	}
	return tag
}

func TransactionStreamTag(lambdaId string, txnId string) uint64 {
	h := xxhash.Sum64String(lambdaId) ^ xxhash.Sum64String(txnId)
	tag := (h << 3) + transactionStreamLowBits
	if tag == 0 || (^tag) == 0 {
		panic("Invalid tag")
	}
	return tag
}

func LockStreamTag(lockId string) uint64 {
	h := xxhash.Sum64String(lockId)
	tag := (h << 3) + lockStreamLowBits
	if tag == 0 || (^tag) == 0 {
		panic("Invalid tag")
	}
	return tag
}
