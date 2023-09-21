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
	flag.StringVar(&FLAGS_bench_case, "bench_case", "write", "write|read|read_cached|ipcbench")
	if FLAGS_bench_case != "write" && FLAGS_bench_case != "read" && FLAGS_bench_case != "read_cached" && FLAGS_bench_case != "ipcbench" {
		panic(fmt.Sprintf("unknown benchmark case: %s", FLAGS_bench_case))
	}

	rand.Seed(int64(FLAGS_rand_seed))
}

func invokeFn(client *http.Client, fnName string, input *common.FnInput) common.FnOutput {
	url := utils.BuildFunctionUrl(FLAGS_faas_gateway, fnName)
	response := common.FnOutput{}
	if err := utils.JsonPostRequest(client, url, input, &response); err != nil {
		log.Printf("[ERROR] fn: %s request failed: %v", fnName, err)
	} else if !response.Success {
		log.Printf("[ERROR] fn: %s request failed: %v", fnName, response.Message)
	}
	return response
}

func printSummary(results []common.FnOutput) {
	asyncLatencies := make([]float64, 0, 128)
	normedAsyncLatencies := make([]float64, 0, 128)

	latencies := make([]float64, 0, 128)
	normedLatencies := make([]float64, 0, 128)

	for _, result := range results {
		if result.Success {
			if result.AsyncLatency != -1 {
				asyncLatency := float64(result.AsyncLatency) / 1000.0
				asyncLatencies = append(asyncLatencies, asyncLatency)
				if result.BatchSize > 1 {
					normedAsyncLatencies = append(normedAsyncLatencies, asyncLatency/float64(result.BatchSize))
				}
			}

			latency := float64(result.Latency) / 1000.0
			latencies = append(latencies, latency)
			if result.BatchSize > 1 {
				normedLatencies = append(normedLatencies, latency/float64(result.BatchSize))
			}
		} else {
			log.Printf("[ERROR] failed append: %v", result.Message)
		}
	}

	if len(asyncLatencies) > 0 {
		median, _ := stats.Median(asyncLatencies)
		p99, _ := stats.Percentile(asyncLatencies, 99.0)
		fmt.Printf("AsyncLatency: median = %.3fms, tail (p99) = %.3fms\n", median, p99)
	}
	if len(normedAsyncLatencies) > 0 {
		median, _ := stats.Median(normedAsyncLatencies)
		p99, _ := stats.Percentile(normedAsyncLatencies, 99.0)
		fmt.Printf("Normed AsyncLatency: median = %.3fms, tail (p99) = %.3fms\n", median, p99)
	}

	if len(latencies) > 0 {
		median, _ := stats.Median(latencies)
		p99, _ := stats.Percentile(latencies, 99.0)
		fmt.Printf("Latency: median = %.3fms, tail (p99) = %.3fms\n", median, p99)
	}
	if len(normedLatencies) > 0 {
		median, _ := stats.Median(normedLatencies)
		p99, _ := stats.Percentile(normedLatencies, 99.0)
		fmt.Printf("Normed latency: median = %.3fms, tail (p99) = %.3fms\n", median, p99)
	}
}

func runBench(client *http.Client, benchCase string, benchFnName string, fnInput *common.FnInput) {
	durationNs := FLAGS_duration * 1000000000 // s -> ns

	resultsMu := sync.Mutex{}
	appendResults := make([]common.FnOutput, 0, 100000)
	resCount := 0

	start := time.Now()
	wg := sync.WaitGroup{}
	for i := 0; i < FLAGS_concurrency; i++ {
		wg.Add(1)
		go func() {
			for time.Since(start) < time.Duration(durationNs) {
				res := invokeFn(client, benchFnName, fnInput)
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
	printSummary(appendResults)
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
		input := &common.FnInput{
			PayloadSize: FLAGS_payload_size,
			BatchSize:   FLAGS_batch_size,
		}
		{
			benchFnName := "benchBokiLogAppend"
			benchCase := "BokiLogAppend"
			runBench(client, benchCase, benchFnName, input)
		}
		{
			benchFnName := "benchAsyncLogAppend"
			benchCase := "AsyncLogAppend"
			runBench(client, benchCase, benchFnName, input)
		}
	} else if FLAGS_bench_case == "read" {
		input := &common.FnInput{
			PayloadSize: FLAGS_payload_size,
			BatchSize:   FLAGS_batch_size,
			ReadCached:  false,
		}
		{
			benchFnName := "benchBokiLogRead"
			benchCase := "BokiLogRead"
			runBench(client, benchCase, benchFnName, input)
		}
		{
			benchFnName := "benchAsyncLogRead"
			benchCase := "AsyncLogRead"
			runBench(client, benchCase, benchFnName, input)
		}
	} else if FLAGS_bench_case == "read_cached" {
		input := &common.FnInput{
			PayloadSize: FLAGS_payload_size,
			BatchSize:   FLAGS_batch_size,
			ReadCached:  true,
		}
		{
			benchFnName := "benchBokiLogRead"
			benchCase := "BokiLogReadCached"
			runBench(client, benchCase, benchFnName, input)
		}
		{
			benchFnName := "benchAsyncLogRead"
			benchCase := "AsyncLogReadCached"
			runBench(client, benchCase, benchFnName, input)
		}
	} else if FLAGS_bench_case == "ipcbench" {
		input := &common.FnInput{
			BatchSize: FLAGS_batch_size,
		}
		{
			benchFnName := "ipcBench"
			benchCase := "IPCBench"
			runBench(client, benchCase, benchFnName, input)
		}
	} else {
		panic("unreachable")
	}
}
