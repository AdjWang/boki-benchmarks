package cayonlib

import (
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var sess = session.Must(session.NewSessionWithOptions(session.Options{
	SharedConfigState: session.SharedConfigEnable,
}))

var DBClient = dynamodb.New(sess, NewDBConfig(DBENV))

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
		return &aws.Config{}
	}
}

var T = int64(60)

var TYPE = "BELDI"             // or "BASELINE"
var DBENV = os.Getenv("DBENV") // "REMOTE" or "LOCAL" or unset

var kTablePrefix = os.Getenv("TABLE_PREFIX")
var gSyncTimeout = time.Duration(60 * time.Second)

const EnableLogSyncOptimization = true
const UseSerialLogSync = true

var EnableFuncCallTrace = false
var EnableLogAppendTrace = true
var EnableLogSyncTrace = true
