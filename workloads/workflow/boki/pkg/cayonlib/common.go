package cayonlib

import (
	"log"
	"os"
	"sync"
	"time"

	"cs.utexas.edu/zjia/faas/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var sess = session.Must(session.NewSessionWithOptions(session.Options{
	SharedConfigState: session.SharedConfigEnable,
}))

// var DBClient = dynamodb.New(sess, NewDBConfig(DBENV))
var DBClient = NewDynamodbSession(sess, NewDBConfig((DBENV)))

type IDBClientDecorator interface {
	AddTrace(tag string, sampleUs int64)

	Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
	GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error)
	Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error)
	DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error)
	TransactWriteItems(input *dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error)
	CreateTable(input *dynamodb.CreateTableInput) (*dynamodb.CreateTableOutput, error)
	DeleteTable(input *dynamodb.DeleteTableInput) (*dynamodb.DeleteTableOutput, error)
	DescribeTable(input *dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error)
	ListTables(input *dynamodb.ListTablesInput) (*dynamodb.ListTablesOutput, error)
}

type DBClientDecorator struct {
	traceMu  sync.Mutex
	counters map[string]*utils.CounterCollector
	stats    map[string]*utils.StatisticsCollector
	client   *dynamodb.DynamoDB
}

func NewDynamodbSession(p client.ConfigProvider, cfgs ...*aws.Config) IDBClientDecorator {
	dbClient := dynamodb.New(p, cfgs...)
	client := &DBClientDecorator{
		traceMu:  sync.Mutex{},
		counters: nil,
		stats:    nil,
		client:   dbClient,
	}
	if EnableDBTrace && client.counters == nil {
		client.counters = make(map[string]*utils.CounterCollector)
		client.stats = make(map[string]*utils.StatisticsCollector)
	}

	return client
}
func (c *DBClientDecorator) AddTrace(tag string, sampleUs int64) {
	c.traceMu.Lock()
	defer c.traceMu.Unlock()

	if _, found := c.counters[tag]; !found {
		c.counters[tag] = utils.NewCounterCollector(tag, 10*time.Second)
	}
	if _, found := c.stats[tag]; !found {
		c.stats[tag] = utils.NewStatisticsCollector(tag, 1000 /*reportSamples*/, 10*time.Second)
	}
	c.counters[tag].Tick(1)
	c.stats[tag].AddSample(float64(sampleUs))
}
func (c *DBClientDecorator) Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	if EnableDBTrace {
		apiTs := time.Now()
		defer func() {
			latency := time.Since(apiTs).Microseconds()
			c.AddTrace("Dynamodb-Query", latency)
		}()
	}
	return c.client.Query(input)
}
func (c *DBClientDecorator) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	if EnableDBTrace {
		apiTs := time.Now()
		defer func() {
			latency := time.Since(apiTs).Microseconds()
			c.AddTrace("Dynamodb-GetItem", latency)
		}()
	}
	return c.client.GetItem(input)
}
func (c *DBClientDecorator) UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error) {
	if EnableDBTrace {
		apiTs := time.Now()
		defer func() {
			latency := time.Since(apiTs).Microseconds()
			c.AddTrace("Dynamodb-UpdateItem", latency)
		}()
	}
	return c.client.UpdateItem(input)
}
func (c *DBClientDecorator) Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error) {
	if EnableDBTrace {
		apiTs := time.Now()
		defer func() {
			latency := time.Since(apiTs).Microseconds()
			c.AddTrace("Dynamodb-Scan", latency)
		}()
	}
	return c.client.Scan(input)
}
func (c *DBClientDecorator) DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
	if EnableDBTrace {
		apiTs := time.Now()
		defer func() {
			latency := time.Since(apiTs).Microseconds()
			c.AddTrace("Dynamodb-DeleteItem", latency)
		}()
	}
	return c.client.DeleteItem(input)
}
func (c *DBClientDecorator) TransactWriteItems(input *dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error) {
	if EnableDBTrace {
		apiTs := time.Now()
		defer func() {
			latency := time.Since(apiTs).Microseconds()
			c.AddTrace("Dynamodb-TransactWriteItems", latency)
		}()
	}
	return c.client.TransactWriteItems(input)
}
func (c *DBClientDecorator) CreateTable(input *dynamodb.CreateTableInput) (*dynamodb.CreateTableOutput, error) {
	if EnableDBTrace {
		apiTs := time.Now()
		defer func() {
			latency := time.Since(apiTs).Microseconds()
			c.AddTrace("Dynamodb-CreateTable", latency)
		}()
	}
	return c.client.CreateTable(input)
}
func (c *DBClientDecorator) DeleteTable(input *dynamodb.DeleteTableInput) (*dynamodb.DeleteTableOutput, error) {
	if EnableDBTrace {
		apiTs := time.Now()
		defer func() {
			latency := time.Since(apiTs).Microseconds()
			c.AddTrace("Dynamodb-DeleteTable", latency)
		}()
	}
	return c.client.DeleteTable(input)
}
func (c *DBClientDecorator) DescribeTable(input *dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error) {
	if EnableDBTrace {
		apiTs := time.Now()
		defer func() {
			latency := time.Since(apiTs).Microseconds()
			c.AddTrace("Dynamodb-DescribeTable", latency)
		}()
	}
	return c.client.DescribeTable(input)
}
func (c *DBClientDecorator) ListTables(input *dynamodb.ListTablesInput) (*dynamodb.ListTablesOutput, error) {
	return c.client.ListTables(input)
}

func NewDBConfig(dbenv string) *aws.Config {
	if dbenv == "LOCAL" {
		// local dynamodb for debugging
		log.Println("[INFO] Init local dynamodb configuration")
		return &aws.Config{
			Endpoint: aws.String("http://dynamodb:8000"),
			Region:   aws.String("us-east-2"),
			// Credentials:                   credentials.NewStaticCredentials("AKID", "SECRET_KEY", "TOKEN"),
			Credentials:                   credentials.NewStaticCredentials("2333", "abcd", "TOKEN"),
			CredentialsChainVerboseErrors: aws.Bool(true),
		}
	} else {
		log.Println("[INFO] Init remote dynamodb configuration")
		return &aws.Config{
			Region: aws.String("us-east-2"),
		}
		// DEBUG
		// data, err := os.ReadFile("/tmp/boki/dbendpoint")
		// if err != nil {
		// 	panic(err)
		// }
		// DBENDPOINT := strings.Trim(string(data), "\n")
		// log.Printf("[INFO] Init remote dynamodb configuration at %s", DBENDPOINT)
		// return &aws.Config{
		// 	Endpoint: aws.String(DBENDPOINT),
		// 	Region:   aws.String("us-east-2"),
		// 	// Credentials:                   credentials.NewStaticCredentials("AKID", "SECRET_KEY", "TOKEN"),
		// 	Credentials:                   credentials.NewStaticCredentials("2333", "abcd", "TOKEN"),
		// 	CredentialsChainVerboseErrors: aws.Bool(true),
		// 	// LogLevel:                      aws.LogLevel(aws.LogDebugWithRequestRetries),
		// }
	}
}

var T = int64(60)

var TYPE = "BELDI"             // or "BASELINE"
var DBENV = os.Getenv("DBENV") // "REMOTE" or "LOCAL" or unset

var kTablePrefix = os.Getenv("TABLE_PREFIX")
var gSyncTimeout = time.Duration(60 * time.Second)

func CHECK(err error) {
	if err != nil {
		panic(err)
	}
}

var EnableDBTrace = false
var EnableLogAppendTrace = true

// Boki lock readonly txn forever, fix this needs additional log appends.
// BokiFlow is not affected because it has no readonly txn, but the
// txnbench does.
// Put a switch here to control benchmark performance.
var FixBokiReadUnlock = false
