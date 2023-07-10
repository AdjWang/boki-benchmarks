package main

import (
	"flag"
	"log"
	"math/rand"

	"cs.utexas.edu/zjia/nightcore-example/utils"
)

var FLAGS_faas_gateway string
var FLAGS_fn_prefix string
var FLAGS_count_target int
var FLAGS_concurrency int
var FLAGS_rand_seed int

func init() {
	flag.StringVar(&FLAGS_faas_gateway, "faas_gateway", "127.0.0.1:8081", "")
	flag.StringVar(&FLAGS_fn_prefix, "fn_prefix", "", "")
	flag.IntVar(&FLAGS_count_target, "count_target", 10, "")
	flag.IntVar(&FLAGS_concurrency, "concurrency", 1, "")
	flag.IntVar(&FLAGS_rand_seed, "rand_seed", 23333, "")

	rand.Seed(int64(FLAGS_rand_seed))
}

func counterFetchAndAdd() int {
	log.Println("[INFO] Test counter fetch and add")

	client := utils.NewFaasClient(FLAGS_faas_gateway, FLAGS_concurrency)
	for i := 0; i < FLAGS_count_target; i++ {
		client.AddJsonFnCall(FLAGS_fn_prefix+"StatestoreTxnExec", utils.JSONValue{})
	}
	results := client.WaitForResults()

	numSuccess := 0
	for _, result := range results {
		if result.Result.Success {
			numSuccess++
		}
		log.Println(result.Result.Message)
	}
	if numSuccess < FLAGS_count_target {
		log.Printf("[ERROR] %d Counter fetch and add requests failed", FLAGS_count_target-numSuccess)
	}
	return numSuccess
}

func counterCheckResult(target int) {
	log.Printf("[INFO] Test counter result as %d", target)

	client := utils.NewFaasClient(FLAGS_faas_gateway, FLAGS_concurrency)
	client.AddJsonFnCall(FLAGS_fn_prefix+"StatestoreTxnCheck", utils.JSONValue{
		"count": int64(target),
	})
	results := client.WaitForResults()

	for _, result := range results {
		if !result.Result.Success {
			log.Printf("[ERROR] check failed %+v", result.Result)
		}
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.Parse()
	target := counterFetchAndAdd()
	counterCheckResult(target)
}
