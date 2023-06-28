package main

import (
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/eniac/Beldi/pkg/cayonlib"
	// "time"
)

func ClearAll() {
	cayonlib.DeleteLambdaTables("singleop")
	cayonlib.DeleteLambdaTables("nop")
	// beldilib.DeleteTable("bsingleop")
	// beldilib.DeleteTable("bnop")
	// beldilib.DeleteLambdaTables("tsingleop")
	// beldilib.DeleteLambdaTables("tnop")
}

func health_check() {
	tablename := "singleop"
	key := "K"
	item := cayonlib.LibRead(tablename, aws.JSONValue{"K": key}, []string{"V"})
	log.Printf("[INFO] Read data from DB: %v", item)
	if len(item) == 0 {
		panic("read data from DB failed")
	}
}

func main() {
	log.SetFlags(log.Lshortfile)

	option := os.Args[1]
	if option == "health_check" {
		health_check()
		return
	} else if option == "clean" {
		log.Println("clear all")
		ClearAll()
		return
	}

	log.Println("clear all")
	ClearAll()

	log.Println("wait until all tables are deleted")
	cayonlib.WaitUntilAllDeleted([]string{"singleop", "nop"})

	log.Println("create lambda table: singleop")
	cayonlib.CreateLambdaTables("singleop")
	log.Println("create lambda table: nop")
	cayonlib.CreateLambdaTables("nop")

	log.Println("wait until all tables are actived")
	cayonlib.WaitUntilAllActive([]string{"singleop", "nop"})

	// beldilib.CreateBaselineTable("bsingleop")
	// beldilib.CreateBaselineTable("bnop")

	// beldilib.WaitUntilAllActive([]string{
	// 	"singleop", "singleop-log", "singleop-collector",
	// 	"nop", "nop-log", "nop-collector",
	// 	"bsingleop", "bnop",
	// })

	// beldilib.CreateTxnTables("tsingleop")
	// beldilib.CreateTxnTables("tnop")

	// time.Sleep(60 * time.Second)
	// beldilib.WriteNRows("singleop", "K", 20)

	log.Println("test write")
	cayonlib.LibWrite("singleop", aws.JSONValue{"K": "K"}, map[expression.NameBuilder]expression.OperandBuilder{
		expression.Name("V"): expression.Value(1),
	})

	// beldilib.LibWrite("tsingleop", aws.JSONValue{"K": "K"}, map[expression.NameBuilder]expression.OperandBuilder{
	// 	expression.Name("V"): expression.Value(1),
	// })
}
