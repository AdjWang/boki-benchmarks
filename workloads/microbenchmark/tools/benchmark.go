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

func init() {
	flag.StringVar(&FLAGS_faas_gateway, "faas_gateway", "127.0.0.1:8081", "")
	flag.IntVar(&FLAGS_duration, "duration", 10, "")
	flag.IntVar(&FLAGS_payload_size, "payload_size", 64, "")
	flag.IntVar(&FLAGS_batch_size, "batch_size", 1, "")
	flag.IntVar(&FLAGS_concurrency, "concurrency", 100, "")
	flag.IntVar(&FLAGS_rand_seed, "rand_seed", 23333, "")

	rand.Seed(int64(FLAGS_rand_seed))
}

func invokeBokiLogAppend(client *http.Client) common.FnOutput {
	input := &common.BokiLogAppendInput{
		PayloadSize: FLAGS_payload_size,
		BatchSize:   FLAGS_batch_size,
	}
	url := utils.BuildFunctionUrl(FLAGS_faas_gateway, "benchBokiLogAppend")
	response := common.FnOutput{}
	if err := utils.JsonPostRequest(client, url, input, &response); err != nil {
		log.Printf("[ERROR] BokiLogAppend request failed: %v", err)
	} else if !response.Success {
		log.Printf("[ERROR] BokiLogAppend request failed: %s", response.Message)
	}
	return response
}

func invokeAsyncLogAppend(client *http.Client) common.FnOutput {
	input := &common.AsyncLogAppendInput{
		PayloadSize: FLAGS_payload_size,
		BatchSize:   FLAGS_batch_size,
	}
	url := utils.BuildFunctionUrl(FLAGS_faas_gateway, "benchAsyncLogAppend")
	response := common.FnOutput{}
	if err := utils.JsonPostRequest(client, url, input, &response); err != nil {
		log.Printf("[ERROR] AsyncLogAppend request failed: %v", err)
	} else if !response.Success {
		log.Printf("[ERROR] AsyncLogAppend request failed: %s", response.Message)
	}
	return response
}

func printSummary(results []common.FnOutput) {
	latencies := make([]float64, 0, 128)
	normedLatencies := make([]float64, 0, 128)
	for _, result := range results {
		if result.Success {
			latency := float64(result.Latency) / 1000.0
			latencies = append(latencies, latency)
			if result.BatchSize > 1 {
				normedLatencies = append(normedLatencies, latency/float64(result.BatchSize))
			}
		} else {
			log.Printf("[ERROR] failed append: %v", result.Message)
		}
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

func runBench(client *http.Client, benchFunc func(*http.Client) common.FnOutput, benchCase string) {
	throughput := FLAGS_concurrency * FLAGS_batch_size
	durationNs := FLAGS_duration * 1000000000

	resultsMu := sync.Mutex{}
	appendResults := make([]common.FnOutput, 0, 100000)

	start := time.Now()
	for time.Since(start) < time.Duration(durationNs) {
		for i := 0; i < FLAGS_concurrency; i++ {
			go func() {
				res := benchFunc(client)
				resultsMu.Lock()
				appendResults = append(appendResults, res)
				resultsMu.Unlock()
			}()
		}
		time.Sleep(time.Second)
	}
	actualDuration := time.Since(start).Seconds()

	resultsMu.Lock()
	fmt.Printf("[%s]\n", benchCase)
	fmt.Printf("Target Throughput: %d, Actual Throughput: %.1f ops per sec\n",
		throughput, float64(len(appendResults))*float64(FLAGS_batch_size)/actualDuration)
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
		Timeout: time.Duration(FLAGS_duration*10) * time.Second,
	}

	{
		benchFunc := invokeBokiLogAppend
		benchCase := "BokiLogAppend"
		runBench(client, benchFunc, benchCase)
	}
	{
		benchFunc := invokeAsyncLogAppend
		benchCase := "AsyncLogAppend"
		runBench(client, benchFunc, benchCase)
	}
}
