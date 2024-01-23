package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"cs.utexas.edu/zjia/faas/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/eniac/Beldi/internal/media/core"
	"github.com/eniac/Beldi/pkg/cayonlib"
)

var kMovieDataPath string = os.Getenv("DATA")
var kTablePrefix string = os.Getenv("TABLE_PREFIX")

var gLogSeqNumGenerator chan int

func InitLogSeqNumGenerator() {
	gLogSeqNumGenerator = make(chan int)
	go func() {
		i := 0
		for {
			gLogSeqNumGenerator <- i
			i++
		}
	}()
}

func NextLogSeqNum() int {
	return <-gLogSeqNumGenerator
}

func CondLibWrite(tablename string, key string,
	update map[expression.NameBuilder]expression.OperandBuilder,
	cond expression.ConditionBuilder) {

	logSeqNum := NextLogSeqNum()

	Key, err := dynamodbattribute.MarshalMap(aws.JSONValue{"K": key})
	cayonlib.CHECK(err)
	condBuilder := expression.Or(
		expression.AttributeNotExists(expression.Name("VERSION")),
		expression.Name("VERSION").LessThan(expression.Value(logSeqNum)))
	if _, err = expression.NewBuilder().WithCondition(cond).Build(); err == nil {
		condBuilder = expression.And(condBuilder, cond)
	}
	updateBuilder := expression.UpdateBuilder{}
	for k, v := range update {
		updateBuilder = updateBuilder.Set(k, v)
	}
	updateBuilder = updateBuilder.
		Set(expression.Name("VERSION"), expression.Value(logSeqNum))
	expr, err := expression.NewBuilder().WithCondition(condBuilder).WithUpdate(updateBuilder).Build()
	cayonlib.CHECK(err)

	_, err = cayonlib.DBClient.UpdateItem(&dynamodb.UpdateItemInput{
		TableName:                 aws.String(kTablePrefix + tablename),
		Key:                       Key,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})
	if err != nil {
		// cayonlib.AssertConditionFailure(errors.Wrapf(err, "SeqNum=%d TableName=%s Key=%v AttrNames=%v, Tables=%v",
		// 	logSeqNum, kTablePrefix+tablename, Key, expr.Names(), cayonlib.ListTables()))
		cayonlib.AssertConditionFailure(err)
	}
}

func Read(tablename string, key string) interface{} {
	item := cayonlib.LibRead(tablename, aws.JSONValue{"K": key}, []string{"V"})
	var res interface{}
	if tmp, ok := item["V"]; ok {
		res = tmp
	} else {
		res = nil
	}
	return res
}

func TryComposeAndUpload(reqId string) {
	// item := Read(core.TComposeReview(), reqId)
	// if item == nil {
	// 	return
	// }
	// res := item.(map[string]interface{})
	// if counter, ok := res["counter"].(float64); ok {
	// 	// DEBUG
	// 	log.Printf("[DEBUG] TryComposeAndUpload reqId=%s counter=%d", reqId, int32(counter))

	// 	// if int32(counter) == 5 {
	// 	// }
	// }
}

func UploadReq(reqId string) {
	// DEBUG
	// log.Printf("[DEBUG] UploadReq reqId=%s", reqId)

	CondLibWrite(core.TComposeReview(), reqId, map[expression.NameBuilder]expression.OperandBuilder{
		expression.Name("V"): expression.Value(aws.JSONValue{"reqId": reqId, "counter": 0}),
	}, expression.ConditionBuilder{})
}

func UploadUniqueId(reqId string, reviewId string) {
	// DEBUG
	// log.Printf("[DEBUG] UploadUniqueId reqId=%s reviewId=%s", reqId, reviewId)

	CondLibWrite(core.TComposeReview(), reqId, map[expression.NameBuilder]expression.OperandBuilder{
		expression.Name("V.reviewId"): expression.Value(reviewId),
		expression.Name("V.counter"):  expression.Name("V.counter").Plus(expression.Value(1)),
	}, expression.ConditionBuilder{})
	TryComposeAndUpload(reqId)
}

func UploadText(reqId string, text string) {
	// DEBUG
	// log.Printf("[DEBUG] UploadText reqId=%s text=%s", reqId, text)

	CondLibWrite(core.TComposeReview(), reqId, map[expression.NameBuilder]expression.OperandBuilder{
		expression.Name("V.text"):    expression.Value(text),
		expression.Name("V.counter"): expression.Name("V.counter").Plus(expression.Value(1)),
	}, expression.ConditionBuilder{})
	TryComposeAndUpload(reqId)
}

func UploadRating(reqId string, rating int32) {
	// DEBUG
	// log.Printf("[DEBUG] UploadRating reqId=%s rating=%d", reqId, rating)

	CondLibWrite(core.TComposeReview(), reqId, map[expression.NameBuilder]expression.OperandBuilder{
		expression.Name("V.rating"):  expression.Value(rating),
		expression.Name("V.counter"): expression.Name("V.counter").Plus(expression.Value(1)),
	}, expression.ConditionBuilder{})
	TryComposeAndUpload(reqId)
}

func UploadUserId(reqId string, userId string) {
	// DEBUG
	// log.Printf("[DEBUG] UploadUserId reqId=%s userId=%s", reqId, userId)

	CondLibWrite(core.TComposeReview(), reqId, map[expression.NameBuilder]expression.OperandBuilder{
		expression.Name("V.userId"):  expression.Value(userId),
		expression.Name("V.counter"): expression.Name("V.counter").Plus(expression.Value(1)),
	}, expression.ConditionBuilder{})
	TryComposeAndUpload(reqId)
}

func UploadMovieId(reqId string, movieId string) {
	// DEBUG
	// log.Printf("[DEBUG] UploadMovieId reqId=%s movieId=%s", reqId, movieId)

	CondLibWrite(core.TComposeReview(), reqId, map[expression.NameBuilder]expression.OperandBuilder{
		expression.Name("V.movieId"): expression.Value(movieId),
		expression.Name("V.counter"): expression.Name("V.counter").Plus(expression.Value(1)),
	}, expression.ConditionBuilder{})
	TryComposeAndUpload(reqId)
}

func BenchComposeReview(ctx context.Context, wg *sync.WaitGroup, clients int) {
	reqDataGenerator := NewMovieBenchDataGenerator(kMovieDataPath)
	for i := 0; i < clients; i++ {
		wg.Add(1)
		go func(iClient int) {
			counter := utils.NewCounterCollector(fmt.Sprintf("db_bench_%d", iClient), 10*time.Second)
			stat := utils.NewStatisticsCollector(fmt.Sprintf("db_bench_%d", iClient), 1000 /*reportSamples*/, 10*time.Second)
			for {
				select {
				case <-ctx.Done():
					wg.Done()
					return
				default:
				}
				reqData := reqDataGenerator.GenerateReqData()

				startTs := time.Now()
				UploadReq(reqData.ReqId)
				// sync
				// UploadUniqueId(reqData.ReqId, reqData.ReviewId)
				// UploadText(reqData.ReqId, reqData.Text)
				// UploadRating(reqData.ReqId, reqData.Rating)
				// UploadUserId(reqData.ReqId, reqData.UserId)
				// UploadMovieId(reqData.ReqId, reqData.MovieId)
				// async
				wgUpload := sync.WaitGroup{}
				wgUpload.Add(5)
				go func() {
					UploadUniqueId(reqData.ReqId, reqData.ReviewId)
					wgUpload.Done()
				}()
				go func() {
					UploadText(reqData.ReqId, reqData.Text)
					wgUpload.Done()
				}()
				go func() {
					UploadRating(reqData.ReqId, reqData.Rating)
					wgUpload.Done()
				}()
				go func() {
					UploadUserId(reqData.ReqId, reqData.UserId)
					wgUpload.Done()
				}()
				go func() {
					UploadMovieId(reqData.ReqId, reqData.MovieId)
					wgUpload.Done()
				}()
				wgUpload.Wait()
				// stat
				counter.Tick(1)
				stat.AddSample(float64(time.Since(startTs).Microseconds()))
			}
		}(i)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	InitLogSeqNumGenerator()

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	clients := 2
	cayonlib.EnableDBTrace = true
	BenchComposeReview(ctx, &wg, clients)
	time.Sleep(30 * time.Second)
	cancel()
	wg.Wait()
}
