package common

// type QueueInitInput struct {
// 	QueueNames []string `json:"queueNames"`
// }

// type ProducerFnInput struct {
// 	QueueName   string `json:"queueName"`
// 	QueueShards int    `json:"queueShards"`
// 	Duration    int    `json:"duration"`
// 	PayloadSize int    `json:"payloadSize"`
// 	IntervalMs  int    `json:"interval"`
// 	BatchSize   int    `json:"batchSize"`
// }

// type ConsumerFnInput struct {
// 	QueueName   string `json:"queueName"`
// 	QueueShards int    `json:"queueShards"`
// 	FixedShard  int    `json:"fixedShard"`
// 	Duration    int    `json:"duration"`
// 	IntervalMs  int    `json:"interval"`
// 	BatchSize   int    `json:"batchSize"`
// 	BlockingPop bool   `json:"blocking"`
// }

type FnOutput struct {
	Success      bool     `json:"success"`
	Message      string   `json:"message"`
	AsyncLatency int      `json:"alatency"`
	Latency      int      `json:"latency"`
	BatchSize    int      `json:"batchSize"`
	SeqNums      []uint64 `json:"seqNums"`
}

type FnInput struct {
	PayloadSize int `json:"payloadSize"`
	BatchSize   int `json:"batchSize"`
}
