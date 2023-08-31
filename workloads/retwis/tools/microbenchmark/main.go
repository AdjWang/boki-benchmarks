package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"cs.utexas.edu/zjia/faas-retwis/utils"

	"github.com/montanaflynn/stats"
)

var FLAGS_faas_gateway string
var FLAGS_fn_prefix string
var FLAGS_concurrency int
var FLAGS_duration int
var FLAGS_percentages string
var FLAGS_rand_seed int

func init() {
	flag.StringVar(&FLAGS_faas_gateway, "faas_gateway", "127.0.0.1:8081", "")
	flag.StringVar(&FLAGS_fn_prefix, "fn_prefix", "", "")
	flag.IntVar(&FLAGS_concurrency, "concurrency", 1, "")
	flag.IntVar(&FLAGS_duration, "duration", 10, "")
	flag.StringVar(&FLAGS_percentages, "percentages", "50,50,0,0", "read,write,txnread,txnwr")
	flag.IntVar(&FLAGS_rand_seed, "rand_seed", 23333, "")

	rand.Seed(int64(FLAGS_rand_seed))
}

func parsePercentages(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return nil, fmt.Errorf("Need exactly four parts splitted by comma")
	}
	results := make([]int, 4)
	for i, part := range parts {
		if parsed, err := strconv.Atoi(part); err != nil {
			return nil, fmt.Errorf("Failed to parse %d-th part", i)
		} else {
			results[i] = parsed
		}
	}
	for i := 1; i < len(results); i++ {
		results[i] += results[i-1]
	}
	if results[len(results)-1] != 100 {
		return nil, fmt.Errorf("Sum of all parts is not 100")
	}
	return results, nil
}

func buildSingleOpReadRequest() utils.JSONValue {
	return utils.JSONValue{}
}

func buildSingleOpWriteRequest() utils.JSONValue {
	value := int64(rand.Intn(100))
	return utils.JSONValue{
		"var": strconv.FormatInt(value, 10),
	}
}

const kTxnConflitMsg = "Failed to commit transaction due to conflicts"

func printFnResult(fnName string, duration time.Duration, results []*utils.FaasCall) {
	total := 0
	succeeded := 0
	txnConflit := 0
	latencies := make([]float64, 0, 128)
	for _, result := range results {
		if result.FnName == FLAGS_fn_prefix+fnName {
			total++
			if result.Result.Success {
				succeeded++
			} else if result.Result.Message == kTxnConflitMsg {
				txnConflit++
			}
			if result.Result.StatusCode == 200 {
				d := result.Result.Duration
				latencies = append(latencies, float64(d.Microseconds()))
			}
		}
	}
	if total == 0 {
		fmt.Println("No results")
		return
	}
	failed := total - succeeded - txnConflit
	fmt.Printf("[%s]\n", fnName)
	fmt.Printf("Throughput: %.1f requests per sec\n", float64(total)/duration.Seconds())
	if txnConflit > 0 {
		ratio := float64(txnConflit) / float64(txnConflit+succeeded)
		fmt.Printf("Transaction conflits: %d (%.2f%%)\n", txnConflit, ratio*100.0)
	}
	if failed > 0 {
		ratio := float64(failed) / float64(total)
		fmt.Printf("Transaction conflits: %d (%.2f%%)\n", failed, ratio*100.0)
	}
	if len(latencies) > 0 {
		median, _ := stats.Median(latencies)
		p99, _ := stats.Percentile(latencies, 99.0)
		fmt.Printf("Latency: median = %.3fms, tail (p99) = %.3fms\n", median/1000.0, p99/1000.0)
	}
}

func main() {
	flag.Parse()

	percentages, err := parsePercentages(FLAGS_percentages)
	if err != nil {
		log.Fatalf("[FATAL] Invalid \"percentages\" flag: %v", err)
	}

	log.Printf("[INFO] Start running for %d seconds with concurrency of %d", FLAGS_duration, FLAGS_concurrency)

	client := utils.NewFaasClient(FLAGS_faas_gateway, FLAGS_concurrency)
	startTime := time.Now()
	for {
		if time.Since(startTime) > time.Duration(FLAGS_duration)*time.Second {
			break
		}
		k := rand.Intn(100)
		if k < percentages[0] {
			client.AddJsonFnCall(FLAGS_fn_prefix+"MicrobenchSingleOpRead", buildSingleOpReadRequest())
		} else if k < percentages[1] {
			client.AddJsonFnCall(FLAGS_fn_prefix+"MicrobenchSingleOpWrite", buildSingleOpWriteRequest())
		} else {
			panic("not implemented") // for txn
		}
	}
	results := client.WaitForResults()
	elapsed := time.Since(startTime)
	fmt.Printf("Benchmark runs for %v, %.1f request per sec\n", elapsed, float64(len(results))/elapsed.Seconds())

	printFnResult("MicrobenchSingleOpRead", elapsed, results)
	printFnResult("MicrobenchSingleOpWrite", elapsed, results)
}
