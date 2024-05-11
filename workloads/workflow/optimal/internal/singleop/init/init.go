package main

import (
	"log"
	"os"
	"strconv"

	"github.com/eniac/Beldi/pkg/cayonlib"
)

const table = "singleop"

var nKeys = 100
var value = 1

func init() {
	if nk, err := strconv.Atoi(os.Getenv("NUM_KEYS")); err == nil {
		nKeys = nk
	}
}

func clean() {
	cayonlib.DeleteLambdaTables(table)
	cayonlib.WaitUntilDeleted(table)
}

func create() {
	cayonlib.CreateLambdaTables(table)
	cayonlib.WaitUntilActive(table)
}

func populate() {
	for i := 0; i < nKeys; i++ {
		cayonlib.Populate(table, strconv.Itoa(i), value, false)
	}
}

func health_check() {
	key := strconv.Itoa(0)
	item := cayonlib.LibReadSingleVersion(table, key)
	log.Printf("[INFO] Read data from DB: %v", item)
	if len(item) == 0 {
		panic("read data from DB failed")
	}
}

func main() {
	option := os.Args[1]
	if option == "health_check" {
		health_check()
	} else if option == "clean" {
		clean()
	} else if option == "create" {
		create()
	} else if option == "populate" {
		populate()
	} else {
		panic("unkown option: " + option)
	}
}
