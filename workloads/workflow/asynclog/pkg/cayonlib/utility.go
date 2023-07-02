package cayonlib

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

func CreateMainTable(lambdaId string) {
	_, _ = DBClient.CreateTable(&dynamodb.CreateTableInput{
		BillingMode: aws.String("PAY_PER_REQUEST"),
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("K"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("K"),
				KeyType:       aws.String("HASH"),
			},
		},
		TableName: aws.String(kTablePrefix + lambdaId),
	})
}

func CreateLogTable(lambdaId string) {
	panic("Not implemented")
}

func CreateCollectorTable(lambdaId string) {
	panic("Not implemented")
}

func CreateBaselineTable(lambdaId string) {
	panic("Not implemented")
}

func CreateLambdaTables(lambdaId string) {
	CreateMainTable(lambdaId)
	// CreateLogTable(lambdaId)
	// CreateCollectorTable(lambdaId)
}

func CreateTxnTables(lambdaId string) {
	CreateBaselineTable(lambdaId)
	CreateLogTable(lambdaId)
	CreateCollectorTable(lambdaId)
}

func DeleteTable(tablename string) {
	_, _ = DBClient.DeleteTable(&dynamodb.DeleteTableInput{TableName: aws.String(kTablePrefix + tablename)})
}

func DeleteLambdaTables(lambdaId string) {
	DeleteTable(lambdaId)
	// DeleteTable(fmt.Sprintf("%s-log", lambdaId))
	// DeleteTable(fmt.Sprintf("%s-collector", lambdaId))
}

func WaitUntilDeleted(tablename string) {
	for {
		res, err := DBClient.DescribeTable(&dynamodb.DescribeTableInput{TableName: aws.String(kTablePrefix + tablename)})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case dynamodb.ErrCodeResourceNotFoundException:
					return
				}
			}
		} else if *res.Table.TableStatus != "DELETING" {
			DeleteTable(tablename)
		}
		time.Sleep(3 * time.Second)
	}
}

func WaitUntilAllDeleted(tablenames []string) {
	for _, tablename := range tablenames {
		WaitUntilDeleted(tablename)
	}
}

func WaitUntilActive(tablename string) bool {
	counter := 0
	for {
		res, err := DBClient.DescribeTable(&dynamodb.DescribeTableInput{TableName: aws.String(kTablePrefix + tablename)})
		if err != nil {
			counter += 1
			fmt.Printf("%s DescribeTable error: %v\n", tablename, err)
		} else {
			if *res.Table.TableStatus == "ACTIVE" {
				return true
			}
			fmt.Printf("%s status: %s\n", tablename, *res.Table.TableStatus)
			if *res.Table.TableStatus != "CREATING" && counter > 6 {
				return false
			}
		}
		time.Sleep(3 * time.Second)
	}
}

func WaitUntilAllActive(tablenames []string) bool {
	for _, tablename := range tablenames {
		res := WaitUntilActive(tablename)
		if !res {
			return false
		}
	}
	return true
}

func WriteHead(tablename string, key string) {
	panic("Not implemented")
}

func WriteTail(tablename string, key string, row string) {
	panic("Not implemented")
}

func WriteNRows(tablename string, key string, n int) {
	panic("Not implemented")
}

func Populate(tablename string, key string, value interface{}, baseline bool) {
	LibWrite(tablename, aws.JSONValue{"K": key},
		map[expression.NameBuilder]expression.OperandBuilder{
			expression.Name("VERSION"): expression.Value(0),
			expression.Name("V"):       expression.Value(value),
		})
}

func CHECK(err error) {
	if err != nil {
		panic(err)
	}
}

// DEBUG
func ASSERT(cond bool, tip string) {
	if !cond {
		panic(fmt.Errorf("[ERROR] assertion failed: %v", tip))
	}
}

func DumpStackTrace() {
	buf := make([]byte, 10000)
	n := runtime.Stack(buf, false)
	log.Printf("[DEBUG] Stack trace : %s ", string(buf[:n]))
}

func dumpDeps(env *Env, logEntryLocalId uint64, depth int) string {
	logEntry, err := env.FaasEnv.AsyncSharedLogRead(env.FaasCtx, logEntryLocalId)
	CHECK(err)
	output := fmt.Sprintf("%v LocalId=%v SeqNum=%v %+v\n",
		strings.Repeat("  ", depth), logEntryLocalId, logEntry.SeqNum, logEntry.Identifiers)
	for _, depLogLocalId := range logEntry.Deps {
		output += dumpDeps(env, depLogLocalId, depth+1)
	}
	return output
}
func DumpDepChain(env *Env, logEntryLocalId uint64, logTip string) {
	log.Printf("[DEBUG] Dep chain for log: %v\n%v\n", logTip, dumpDeps(env, logEntryLocalId, 0))
}

func ListTables() {
	// create the input configuration instance
	input := &dynamodb.ListTablesInput{}

	log.Printf("Tables:\n")

	for {
		// Get the list of tables
		result, err := DBClient.ListTables(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case dynamodb.ErrCodeInternalServerError:
					log.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
				default:
					log.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				log.Println(err.Error())
			}
			return
		}

		for _, n := range result.TableNames {
			log.Println(*n)
		}

		// assign the last read tablename as the start for our next call to the ListTables function
		// the maximum number of table names returned in a call is 100 (default), which requires us to make
		// multiple calls to the ListTables function to retrieve all table names
		input.ExclusiveStartTableName = result.LastEvaluatedTableName

		if result.LastEvaluatedTableName == nil {
			break
		}
	}
}
