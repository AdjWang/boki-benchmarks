package cayonlib

import (
	"os"
	"time"

	// "github.com/aws/aws-sdk-go/aws"
	// "github.com/aws/aws-sdk-go/aws/credentials"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var sess = session.Must(session.NewSessionWithOptions(session.Options{
	SharedConfigState: session.SharedConfigEnable,
}))

// var DBClient = dynamodb.New(sess)

// DEBUG
// local dynamodb
var DBClient = dynamodb.New(sess,
	&aws.Config{
		Endpoint: aws.String("http://10.0.2.15:8000"),
		Region:   aws.String("us-east-2"),
		// Credentials:                   credentials.NewStaticCredentials("AKID", "SECRET_KEY", "TOKEN"),
		Credentials:                   credentials.NewStaticCredentials("2333", "abcd", "TOKEN"),
		CredentialsChainVerboseErrors: aws.Bool(true),
	})

var T = int64(60)

var TYPE = "BELDI"

var kTablePrefix = os.Getenv("TABLE_PREFIX")
var gSyncTimeout = time.Duration(60 * time.Second)
