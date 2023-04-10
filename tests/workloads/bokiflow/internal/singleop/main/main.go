package main

import (
	// "fmt"
	// "github.com/aws/aws-lambda-go/lambda"

	"log"
	"os"
	"time"

	"github.com/eniac/Beldi/pkg/cayonlib"

	"cs.utexas.edu/zjia/faas"
)

var TXN = "DISABLE"

func Handler(env *cayonlib.Env) interface{} {
	results := map[string]int64{}
	log.Println("Handler start")

	if TXN == "ENABLE" {
		panic("Not implemented")
	}
	if cayonlib.TYPE == "BELDI" {
		// DEBUG
		// log.Println("DWrite")
		// a := shortuuid.New()
		// start := time.Now()
		// cayonlib.Write(env, "singleop", "K", map[expression.NameBuilder]expression.OperandBuilder{
		// 	expression.Name("V"): expression.Value(a),
		// })
		// results["latencyDWrite"] = time.Since(start).Microseconds()
		// // fmt.Printf("DURATION DWrite %s\n", time.Since(start))

		// log.Println("CWriteT")
		// start = time.Now()
		// cayonlib.CondWrite(env, "singleop", "K", map[expression.NameBuilder]expression.OperandBuilder{
		// 	expression.Name("V2"): expression.Value(1),
		// }, expression.Name("V").Equal(expression.Value(a)))
		// results["latencyCWriteT"] = time.Since(start).Microseconds()
		// // fmt.Printf("DURATION CWriteT %s\n", time.Since(start))

		// log.Println("CWriteF")
		// start = time.Now()
		// cayonlib.CondWrite(env, "singleop", "K", map[expression.NameBuilder]expression.OperandBuilder{
		// 	expression.Name("V2"): expression.Value(a),
		// }, expression.Name("V").Equal(expression.Value(2)))
		// results["latencyCWriteF"] = time.Since(start).Microseconds()
		// // fmt.Printf("DURATION CWriteF %s\n", time.Since(start))

		// log.Println("Read")
		// start = time.Now()
		// cayonlib.Read(env, "singleop", "K")
		// results["latencyRead"] = time.Since(start).Microseconds()
		// // fmt.Printf("DURATION Read %s\n", time.Since(start))

		log.Println("Call")
		start := time.Now()
		cayonlib.SyncInvoke2(env, "nop", "")
		results["latencyCall"] = time.Since(start).Microseconds()
		// fmt.Printf("DURATION Call %s\n", time.Since(start))
	} else {
		log.Println("unused type")
		results["result"] = -1
	}
	log.Println("Handler end")
	return results
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Lshortfile)
	// lambda.Start(cayonlib.Wrapper(Handler))
	faas.Serve(cayonlib.CreateFuncHandlerFactory(Handler))
}
