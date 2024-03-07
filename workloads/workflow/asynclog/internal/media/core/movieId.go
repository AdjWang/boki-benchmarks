package core

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/eniac/Beldi/pkg/cayonlib"
)

func UploadMovie(env *cayonlib.Env, reqId string, title string, rating int32) {
	item := cayonlib.Read(env, TMovieId(), title)
	if item == nil {
		cayonlib.ListTables()

		items := cayonlib.LibScan(TMovieId(), []string{"V"})
		var res []interface{}
		for _, item := range items {
			res = append(res, item["V"])
		}
		panic(fmt.Sprintf("%s doesn't exist. table name=%s items=%+v", TMovieId(), title, res))
	}
	val := item.(map[string]interface{})
	if movieId, exist := val["movieId"].(string); exist {
		cayonlib.AsyncInvoke(env, TComposeReview(), RPCInput{
			Function: "UploadMovieId",
			Input: aws.JSONValue{
				"movieId": movieId,
				"reqId":   reqId,
			},
		})
		cayonlib.AsyncInvoke(env, TRating(), RPCInput{
			Function: "UploadRating2",
			Input: aws.JSONValue{
				"reqId":  reqId,
				"rating": rating,
			},
		})
	}
}

func RegisterMovieId(env *cayonlib.Env, title string, movieId string) {
	cayonlib.Write(env, TMovieId(), title, map[expression.NameBuilder]expression.OperandBuilder{
		expression.Name("V"): expression.Value(aws.JSONValue{"movieId": movieId, "title": title}),
	})
}
