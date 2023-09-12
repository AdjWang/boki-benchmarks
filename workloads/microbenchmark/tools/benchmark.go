package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"cs.utexas.edu/zjia/microbenchmark/common"
	"cs.utexas.edu/zjia/microbenchmark/handlers"
	"cs.utexas.edu/zjia/microbenchmark/utils"

	"github.com/montanaflynn/stats"
)

var FLAGS_faas_gateway string
var FLAGS_duration int
var FLAGS_payload_size int
var FLAGS_batch_size int
var FLAGS_concurrency int
var FLAGS_rand_seed int
var FLAGS_bench_case string

func init() {
	flag.StringVar(&FLAGS_faas_gateway, "faas_gateway", "127.0.0.1:8081", "")
	flag.IntVar(&FLAGS_duration, "duration", 10, "")
	flag.IntVar(&FLAGS_payload_size, "payload_size", 64, "")
	flag.IntVar(&FLAGS_batch_size, "batch_size", 1, "")
	flag.IntVar(&FLAGS_concurrency, "concurrency", 100, "")
	flag.IntVar(&FLAGS_rand_seed, "rand_seed", 23333, "")
	flag.StringVar(&FLAGS_bench_case, "bench_case", "write", "write|read|read_stream")
	if FLAGS_bench_case != "write" && FLAGS_bench_case != "read" && FLAGS_bench_case != "read_stream" {
		panic(fmt.Sprintf("unknown benchmark case: %s", FLAGS_bench_case))
	}

	rand.Seed(int64(FLAGS_rand_seed))
}

func invokeReadFn(client *http.Client, fnName string, input *handlers.ReadInput) handlers.ReadOutput {
	url := utils.BuildFunctionUrl(FLAGS_faas_gateway, fnName)
	response := handlers.ReadOutput{}
	if err := utils.JsonPostRequest(client, url, input, &response); err != nil {
		log.Printf("[ERROR] fn: %s request failed: %v", fnName, err)
	} else if !response.Success {
		log.Printf("[ERROR] fn: %s request failed: %v", fnName, response.Message)
	}
	return response
}
func invokeAppendFn(client *http.Client, fnName string, input *handlers.AppendInput) handlers.AppendOutput {
	url := utils.BuildFunctionUrl(FLAGS_faas_gateway, fnName)
	response := handlers.AppendOutput{}
	if err := utils.JsonPostRequest(client, url, input, &response); err != nil {
		log.Printf("[ERROR] fn: %s request failed: %v", fnName, err)
	} else if !response.Success {
		log.Printf("[ERROR] fn: %s request failed: %v", fnName, response.Message)
	}
	return response
}

func printSummary(title string, datas []float64) {
	if len(datas) == 0 {
		return
	}
	median, _ := stats.Median(datas)
	p99, _ := stats.Percentile(datas, 99.0)
	fmt.Printf("%v: median = %.3fms, tail (p99) = %.3fms\n", title, median, p99)
}

func printReadSummary(results []handlers.ReadOutput) {
	appendLatencies := make([]float64, 0, 128)
	normedAppendLatencies := make([]float64, 0, 128)

	latencies := make([]float64, 0, 128)
	normedLatencies := make([]float64, 0, 128)

	for _, result := range results {
		if result.Success {
			if result.Stage1Latency != -1 {
				asyncLatency := float64(result.Stage1Latency) / 1000.0
				appendLatencies = append(appendLatencies, asyncLatency)
				if result.BatchSize > 1 {
					normedAppendLatencies = append(normedAppendLatencies, asyncLatency/float64(result.BatchSize))
				}
			}

			latency := float64(result.Stage2Latency) / 1000.0
			latencies = append(latencies, latency)
			if result.BatchSize > 1 {
				normedLatencies = append(normedLatencies, latency/float64(result.BatchSize))
			}
		} else {
			log.Printf("[ERROR] failed append: %v", result.Message)
		}
	}

	printSummary("Append Latency", appendLatencies)
	printSummary("Normed Append Latency", normedAppendLatencies)

	synctoLatencies := common.ListSub(latencies, appendLatencies)
	normedSynctoLatencies := common.ListSub(normedLatencies, normedAppendLatencies)
	printSummary("Syncto Latency", synctoLatencies)
	printSummary("Normed Syncto Latency", normedSynctoLatencies)

	printSummary("Total Latency", latencies)
	printSummary("Normed Total Latency", normedLatencies)
}

func runReadBench(client *http.Client, benchCase string, benchFnName string, fnInput *handlers.ReadInput) {
	durationNs := FLAGS_duration * 1000000000 // s -> ns

	resultsMu := sync.Mutex{}
	appendResults := make([]handlers.ReadOutput, 0, 100000)
	resCount := 0

	start := time.Now()
	wg := sync.WaitGroup{}
	for i := 0; i < FLAGS_concurrency; i++ {
		wg.Add(1)
		go func() {
			for time.Since(start) < time.Duration(durationNs) {
				res := invokeReadFn(client, benchFnName, fnInput)
				resultsMu.Lock()
				appendResults = append(appendResults, res)
				resCount += len(res.SeqNums)
				resultsMu.Unlock()
			}
			wg.Done()
		}()
	}
	wg.Wait()
	actualDuration := time.Since(start).Seconds()

	resultsMu.Lock()
	fmt.Printf("[%s]\n", benchCase)
	fmt.Printf("Duration: %vs Throughput: %.1f ops per sec\n",
		actualDuration, float64(resCount)/actualDuration)
	printReadSummary(appendResults)
	resultsMu.Unlock()
}

func printAppendSummary(results []handlers.AppendOutput) {
	asyncLatencies := make([]float64, 0, 128)
	normedAsyncLatencies := make([]float64, 0, 128)

	latencies := make([]float64, 0, 128)
	normedLatencies := make([]float64, 0, 128)

	for _, result := range results {
		if result.Success {
			if result.Stage1Latency != -1 {
				asyncLatency := float64(result.Stage1Latency) / 1000.0
				asyncLatencies = append(asyncLatencies, asyncLatency)
				if result.BatchSize > 1 {
					normedAsyncLatencies = append(normedAsyncLatencies, asyncLatency/float64(result.BatchSize))
				}
			}

			latency := float64(result.Stage2Latency) / 1000.0
			latencies = append(latencies, latency)
			if result.BatchSize > 1 {
				normedLatencies = append(normedLatencies, latency/float64(result.BatchSize))
			}
		} else {
			log.Printf("[ERROR] failed append: %v", result.Message)
		}
	}

	printSummary("AsyncLatency", asyncLatencies)
	printSummary("Normed AsyncLatency", normedAsyncLatencies)

	if len(asyncLatencies) > 0 {
		awaitLatencies := common.ListSub(latencies, asyncLatencies)
		normedAwaitLatencies := common.ListSub(normedLatencies, normedAsyncLatencies)
		printSummary("Await Latency", awaitLatencies)
		printSummary("Normed Await Latency", normedAwaitLatencies)
	}

	printSummary("Total Latency", latencies)
	printSummary("Normed Total Latency", normedLatencies)
}
func runAppendBench(client *http.Client, benchCase string, benchFnName string, fnInput *handlers.AppendInput) {
	durationNs := FLAGS_duration * 1000000000 // s -> ns

	resultsMu := sync.Mutex{}
	appendResults := make([]handlers.AppendOutput, 0, 100000)
	resCount := 0

	start := time.Now()
	wg := sync.WaitGroup{}
	for i := 0; i < FLAGS_concurrency; i++ {
		wg.Add(1)
		go func() {
			for time.Since(start) < time.Duration(durationNs) {
				res := invokeAppendFn(client, benchFnName, fnInput)
				resultsMu.Lock()
				appendResults = append(appendResults, res)
				resCount += len(res.SeqNums)
				resultsMu.Unlock()
			}
			wg.Done()
		}()
	}
	wg.Wait()
	actualDuration := time.Since(start).Seconds()

	resultsMu.Lock()
	fmt.Printf("[%s]\n", benchCase)
	fmt.Printf("Duration: %vs Throughput: %.1f ops per sec\n",
		actualDuration, float64(resCount)/actualDuration)
	printAppendSummary(appendResults)
	resultsMu.Unlock()
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()

	client := &http.Client{
		Transport: &http.Transport{
			// MaxConnsPerHost: FLAGS_concurrency,
			// MaxIdleConns:    FLAGS_concurrency,
			IdleConnTimeout: 30 * time.Second,
		},
		Timeout: time.Duration(FLAGS_duration*2) * time.Second,
	}

	if FLAGS_bench_case == "write" {
		input := &handlers.AppendInput{
			PayloadSize: FLAGS_payload_size,
			BatchSize:   FLAGS_batch_size,
		}
		{
			benchFnName := "benchBokiLogAppend"
			benchCase := "BokiLogAppend"
			runAppendBench(client, benchCase, benchFnName, input)
		}
		{
			benchFnName := "benchAsyncLogAppend"
			benchCase := "AsyncLogAppend"
			runAppendBench(client, benchCase, benchFnName, input)
		}
	} else if FLAGS_bench_case == "read" {
		input := &handlers.ReadInput{
			PayloadSize:  FLAGS_payload_size,
			BatchSize:    FLAGS_batch_size,
			ReadAsStream: false,
		}
		{
			benchFnName := "benchBokiLogRead"
			benchCase := "BokiLogRead"
			runReadBench(client, benchCase, benchFnName, input)
		}
		input.ReadAsStream = true
		{
			benchFnName := "benchBokiLogRead"
			benchCase := "BokiLogRead"
			runReadBench(client, benchCase, benchFnName, input)
		}
	} else if FLAGS_bench_case == "read_stream" {
		input := &handlers.ReadInput{
			PayloadSize: FLAGS_payload_size,
			BatchSize:   FLAGS_batch_size,
		}
		{
			benchFnName := "benchBokiLogRead"
			benchCase := "BokiLogReadCached"
			runReadBench(client, benchCase, benchFnName, input)
		}
		{
			benchFnName := "benchAsyncLogRead"
			benchCase := "AsyncLogReadCached"
			runReadBench(client, benchCase, benchFnName, input)
		}
	} else {
		panic("unreachable")
	}
}
