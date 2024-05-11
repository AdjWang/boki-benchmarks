package main

import (
	"context"
	"fmt"

	// "github.com/eniac/Beldi/pkg/beldilib"
	"cs.utexas.edu/zjia/faas"
	"cs.utexas.edu/zjia/faas/types"
)

type gcHandler struct {
	env types.Environment
}

type gcHandlerFactory struct{}

func (h *gcHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	// var jsonInput aws.JSONValue
	// err := json.Unmarshal(input, &jsonInput)
	// beldilib.CHECK(err)
	// if _, ok := jsonInput["account"]; !ok {
	// 	service := jsonInput["service"].(string)
	// 	static := jsonInput["static"].(bool)
	// 	log.Printf("Start GC: %s", service)
	// 	if static {
	// 		beldilib.StaticGC(service)
	// 	} else {
	// 		beldilib.GC(service)
	// 	}
	// 	return []byte("OK"), nil
	// }
	// services := []string{"ComposeReview", "UserReview", "MovieReview", "ReviewStorage"}
	// statics := []string{"Frontend", "MovieId", "UniqueId", "Plot", "MovieInfo", "User", "Rating", "Text"}
	// for _, service := range services {
	// 	args := aws.JSONValue{
	// 		"service": service,
	// 		"static":  false,
	// 	}
	// 	stream, err := json.Marshal(args)
	// 	beldilib.CHECK(err)
	// 	err = h.env.InvokeFuncAsync(ctx, "mediagc", stream)
	// 	beldilib.CHECK(err)
	// }
	// for _, service := range statics {
	// 	args := aws.JSONValue{
	// 		"service": service,
	// 		"static":  true,
	// 	}
	// 	stream, err := json.Marshal(args)
	// 	beldilib.CHECK(err)
	// 	err = h.env.InvokeFuncAsync(ctx, "mediagc", stream)
	// 	beldilib.CHECK(err)
	// }
	// return []byte("OK"), nil
	panic("not implemented")
}

func (f *gcHandlerFactory) New(env types.Environment, funcName string) (types.FuncHandler, error) {
	return &gcHandler{env: env}, nil
}

func (f *gcHandlerFactory) GrpcNew(env types.Environment, service string) (types.GrpcFuncHandler, error) {
	return nil, fmt.Errorf("Not implemented")
}

func main() {
	faas.Serve(&gcHandlerFactory{})
}
