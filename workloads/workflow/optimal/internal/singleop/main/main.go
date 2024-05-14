package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/eniac/Beldi/pkg/cayonlib"

	"cs.utexas.edu/zjia/faas"
	"cs.utexas.edu/zjia/faas/types"
)

const table = "singleop"

var nKeys = 100
var value = 1

func init() {
	if nk, err := strconv.Atoi(os.Getenv("NUM_KEYS")); err == nil {
		nKeys = nk
	}
	rand.Seed(time.Now().UnixNano())
}

// DEBUG
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
	} else if !reflect.DeepEqual(logEntry.AuxData, expected.AuxData) {
		output += fmt.Sprintf("[FAIL]  %v aux data=%v, expect=%v\n", funcName, logEntry.AuxData, expected.AuxData)
		return output, false
	} else {
		output += fmt.Sprintf("[PASS] %v seqNum=0x%016X, tags=%v, data=%v, auxData=%v\n",
			funcName, logEntry.SeqNum, logEntry.Tags, logEntry.Data, logEntry.AuxData)
		return output, true
	}
}

// DEBUG
func TestSLogReadYourWrite(env *cayonlib.Env, count int) {
	output := "TestSLogReadYourWrite\n"
	// test
	var seqNumAppended uint64
	tags := []uint64{1}
	data := []byte{1, 2, 3}
	{
		seqNum, err := env.FaasEnv.SharedLogAppend(env.FaasCtx, tags, data)
		if err != nil {
			output += fmt.Sprintf("[FAIL] shared log append error: %v\n", err)
			log.Printf("TestSLogReadYourWrite failed. Output=%v", output)
			return
		} else {
			output += fmt.Sprintf("[PASS] shared log append seqNum: 0x%016X\n", seqNum)
			seqNumAppended = seqNum
		}
	}
	readTag := uint64(1)
	{
		logEntry, err := env.FaasEnv.SharedLogReadNext(env.FaasCtx, readTag, seqNumAppended)
		if err != nil {
			output += fmt.Sprintf("[FAIL] shared log read next error: %v\n", err)
			log.Printf("TestSLogReadYourWrite failed. Output=%v", output)
			return
		} else {
			res, passed := assertLogEntry("shared log read next", logEntry, &types.LogEntry{
				SeqNum:  seqNumAppended,
				Tags:    tags,
				Data:    data,
				AuxData: []byte{},
			})
			output += res
			if !passed {
				log.Printf("TestSLogReadYourWrite failed. Output=%v", output)
				return
			}
		}
	}
	log.Printf("TestSLogReadYourWrite passed count=%v", count)
}

func Handler(env *cayonlib.Env) interface{} {
	TestSLogReadYourWrite(env, 1)

	results := map[string]int64{}

	start := time.Now()
	if cayonlib.TYPE == "NONE" {
		cayonlib.LibReadSingleVersion(table, strconv.Itoa(rand.Intn(nKeys)))
	} else {
		cayonlib.Read(env, table, strconv.Itoa(rand.Intn(nKeys)))
	}
	results["Read"] = time.Since(start).Microseconds()

	start = time.Now()
	if cayonlib.TYPE == "NONE" {
		cayonlib.LibWriteMultiVersion(table, strconv.Itoa(rand.Intn(nKeys)), 0, map[expression.NameBuilder]expression.OperandBuilder{
			expression.Name("V"): expression.Value(value),
		})
	} else {
		cayonlib.Write(env, table, strconv.Itoa(rand.Intn(nKeys)), map[expression.NameBuilder]expression.OperandBuilder{
			expression.Name("V"): expression.Value(value),
		}, false)
	}
	results["Write"] = time.Since(start).Microseconds()

	start = time.Now()
	if cayonlib.TYPE == "NONE" {
		env.FaasEnv.InvokeFunc(env.FaasCtx, "nop", []byte{})
	} else {
		cayonlib.SyncInvoke(env, "nop", "")
	}
	results["Invoke"] = time.Since(start).Microseconds()

	return results
}

func main() {
	faas.Serve(cayonlib.CreateFuncHandlerFactory(Handler))
}
