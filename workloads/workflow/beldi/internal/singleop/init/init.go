package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/eniac/Beldi/pkg/beldilib"
	"log"
	"os"
)

func ClearAll() {
	beldilib.DeleteLambdaTables("singleop")
	beldilib.DeleteLambdaTables("nop")
	beldilib.DeleteTable("bsingleop")
	beldilib.DeleteTable("bnop")
	// beldilib.DeleteLambdaTables("tsingleop")
	// beldilib.DeleteLambdaTables("tnop")

	beldilib.WaitUntilAllDeleted([]string{
		"singleop", "singleop-log", "singleop-collector",
		"nop", "nop-log", "nop-collector",
		// "tsingleop", "tsingleop-log", "tsingleop-collector",
		// "tnop", "tnop-log", "tnop-collector",
		"bsingleop", "bnop",
	})
}

func health_check() {
	tablename := "bsingleop"
	key := "K"
	item := beldilib.LibRead(tablename, aws.JSONValue{"K": key}, []string{"V"})
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
	} else if option == "create" {
		beldilib.CreateLambdaTables("singleop")
		beldilib.CreateLambdaTables("nop")

		beldilib.CreateBaselineTable("bsingleop")
		beldilib.CreateBaselineTable("bnop")

		beldilib.WaitUntilAllActive([]string{
			"singleop", "singleop-log", "singleop-collector",
			"nop", "nop-log", "nop-collector",
			"bsingleop", "bnop",
		})

		// beldilib.CreateTxnTables("tsingleop")
		// beldilib.CreateTxnTables("tnop")
		return
	} else if option == "populate" {
		beldilib.WriteNRows("singleop", "K", 20)

		beldilib.LibWrite("bsingleop", aws.JSONValue{"K": "K"}, map[expression.NameBuilder]expression.OperandBuilder{
			expression.Name("V"): expression.Value(1),
		})

		// beldilib.LibWrite("tsingleop", aws.JSONValue{"K": "K"}, map[expression.NameBuilder]expression.OperandBuilder{
		// 	expression.Name("V"): expression.Value(1),
		// })
		return
	} else {
		panic(fmt.Sprintf("unknown option: %s", option))
	}
}
