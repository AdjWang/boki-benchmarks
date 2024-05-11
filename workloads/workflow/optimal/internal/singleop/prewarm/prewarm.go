package main

import (
	"os"
	"strconv"

	"github.com/eniac/Beldi/pkg/cayonlib"

	"cs.utexas.edu/zjia/faas"
)

const table = "singleop"

var nKeys = 100

func init() {
	if nk, err := strconv.Atoi(os.Getenv("NUM_KEYS")); err == nil {
		nKeys = nk
	}
}

func Handler(env *cayonlib.Env) interface{} {
	if cayonlib.TYPE == "WRITELOG" {
		for i := 0; i < nKeys; i++ {
			cayonlib.Read(env, table, strconv.Itoa(i))
		}
	}
	return nil
}

func main() {
	faas.Serve(cayonlib.CreateFuncHandlerFactory(Handler))
}
