package main

import (
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/eniac/Beldi/internal/txnbench/internal/common"
	"github.com/eniac/Beldi/pkg/cayonlib"
)

var services = []string{"readonly", "writeonly"}
var defaultKey string = "ByteStream"

func tables(baseline bool) {
	if baseline {
		panic("Not implemented for baseline")
	} else {
		for {
			tablenames := []string{}
			for _, service := range services {
				err := cayonlib.CreateLambdaTables(service)
				if err != nil {
					panic(err)
				}
				tablenames = append(tablenames, service)
			}
			if cayonlib.WaitUntilAllActive(tablenames) {
				break
			}
		}
	}
}

func deleteTables(baseline bool) {
	if baseline {
		panic("Not implemented for baseline")
	} else {
		for _, service := range services {
			cayonlib.DeleteLambdaTables(service)
			// cayonlib.WaitUntilAllDeleted([]string{service})
		}
	}
}

func readonly(baseline bool) {
	data := []byte{}
	for i := 0; i < common.DataSize; i++ {
		data = append(data, byte(i))
	}
	cayonlib.Populate("readonly", defaultKey, common.ReadOnlyData{ByteStream: string(data)}, baseline)
}

func writeonly(baseline bool) {
	data := []byte{}
	for i := 0; i < common.DataSize; i++ {
		data = append(data, byte(i))
	}
	cayonlib.Populate("writeonly", defaultKey, common.WriteOnlyData{ByteStream: string(data)}, baseline)
}

func populate(baseline bool) {
	readonly(baseline)
	writeonly(baseline)
}

func health_check() {
	tablename := "readonly"
	item := cayonlib.LibRead(tablename, aws.JSONValue{"K": defaultKey}, []string{"V"})
	log.Printf("[INFO] Read data from DB: %v", item)
	if len(item) == 0 {
		panic("read data from DB failed")
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	option := os.Args[1]
	log.Printf("running: %s - %s", os.Args[0], os.Args[1])
	if option == "health_check" {
		health_check()
		return
	}

	baseline := os.Args[2] == "baseline"
	if option == "create" {
		tables(baseline)
	} else if option == "populate" {
		populate(baseline)
	} else if option == "clean" {
		deleteTables(baseline)
	}
}
