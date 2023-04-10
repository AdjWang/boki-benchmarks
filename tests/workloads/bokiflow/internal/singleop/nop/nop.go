package main

import (
	// "github.com/aws/aws-lambda-go/lambda"
	"log"
	"os"

	"github.com/eniac/Beldi/pkg/cayonlib"

	"cs.utexas.edu/zjia/faas"
)

func Handler(env *cayonlib.Env) interface{} {
	return 0
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Lshortfile)
	// lambda.Start(beldilib.Wrapper(Handler))
	faas.Serve(cayonlib.CreateFuncHandlerFactory(Handler))
}
