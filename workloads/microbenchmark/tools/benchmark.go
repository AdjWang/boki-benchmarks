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
var FLAGS_append_interval int
var FLAGS_concurrency int
var FLAGS_rand_seed int

func init() {
	flag.StringVar(&FLAGS_faas_gateway, "faas_gateway", "127.0.0.1:8081", "")
	flag.IntVar(&FLAGS_duration, "duration", 10, "")
	flag.IntVar(&FLAGS_payload_size, "payload_size", 64, "")
	flag.IntVar(&FLAGS_batch_size, "batch_size", 1, "")
	flag.IntVar(&FLAGS_append_interval, "append_interval", 4, "")
	flag.IntVar(&FLAGS_concurrency, "concurrency", 100, "")
	flag.IntVar(&FLAGS_rand_seed, "rand_seed", 23333, "")

	rand.Seed(int64(FLAGS_rand_seed))
}

func invokeBokiLogAppend(client *http.Client, response *common.FnOutput, wg *sync.WaitGroup) {
	defer wg.Done()

	input := &common.BokiLogAppendInput{
		Duration:    FLAGS_duration,
		PayloadSize: FLAGS_payload_size,
		IntervalMs:  FLAGS_append_interval,
		BatchSize:   FLAGS_batch_size,
	}
	url := utils.BuildFunctionUrl(FLAGS_faas_gateway, "benchBokiLogAppend")
	if err := utils.JsonPostRequest(client, url, input, response); err != nil {
		log.Printf("[ERROR] BokiLogAppend request failed: %v", err)
	} else if !response.Success {
		log.Printf("[ERROR] BokiLogAppend request failed: %s", response.Message)
	}
}

func invokeAsyncLogAppend(client *http.Client, response *common.FnOutput, wg *sync.WaitGroup) {
	defer wg.Done()

	input := &common.AsyncLogAppendInput{
		Duration:    FLAGS_duration,
		PayloadSize: FLAGS_payload_size,
		IntervalMs:  FLAGS_append_interval,
		BatchSize:   FLAGS_batch_size,
	}
	url := utils.BuildFunctionUrl(FLAGS_faas_gateway, "benchAsyncLogAppend")
	if err := utils.JsonPostRequest(client, url, input, response); err != nil {
		log.Printf("[ERROR] AsyncLogAppend request failed: %v", err)
	} else if !response.Success {
		log.Printf("[ERROR] AsyncLogAppend request failed: %s", response.Message)
	}
}

func printSummary(title string, results []common.FnOutput) {
	latencies := make([]float64, 0, 128)
	tput := float64(0)
	normedLatencies := make([]float64, 0, 128)
	for _, result := range results {
		if result.Success {
			totalMessages := 0
			for idx, elem := range result.Latencies {
				latency := float64(elem) / 1000.0
				latencies = append(latencies, latency)
				if idx < len(result.NumMessages) {
					num := result.NumMessages[idx]
					normedLatencies = append(normedLatencies, latency/float64(num))
					totalMessages += num
				} else {
					totalMessages++
				}
			}
			tput += float64(totalMessages) / result.Duration
		}
	}
	fmt.Printf("[%s]\n", title)
	fmt.Printf("Throughput: %.1f ops per sec\n", tput)
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

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()

	concurrency := FLAGS_concurrency

	client := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost: concurrency,
			MaxIdleConns:    concurrency,
			IdleConnTimeout: 30 * time.Second,
		},
		Timeout: time.Duration(FLAGS_duration*10) * time.Second,
	}

	{
		var wg sync.WaitGroup
		appendResults := make([]common.FnOutput, concurrency)
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go invokeBokiLogAppend(client, &appendResults[i], &wg)
		}
		wg.Wait()
		printSummary("BokiLogAppend", appendResults)
	}

	{
		var wg sync.WaitGroup
		appendResults := make([]common.FnOutput, concurrency)
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go invokeAsyncLogAppend(client, &appendResults[i], &wg)
		}
		wg.Wait()
		printSummary("AsyncLogAppend", appendResults)
	}
}
