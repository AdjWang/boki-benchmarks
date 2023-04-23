package core

import (
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/eniac/Beldi/pkg/cayonlib"
	"github.com/mitchellh/mapstructure"
)

func ReadPage(env *cayonlib.Env, movieId string) Page {
	var movieInfo MovieInfo
	var reviews []Review
	var castInfos []CastInfo
	var plot string

	res, _ := cayonlib.SyncInvoke(env, TMovieInfo(), RPCInput{
		Function: "ReadMovieInfo",
		Input:    aws.JSONValue{"movieId": movieId},
	})
	cayonlib.CHECK(mapstructure.Decode(res, &movieInfo))
	var ids []string
	for _, cast := range movieInfo.Casts {
		ids = append(ids, cast.CastInfoId)
	}

	stepLog2, invoke2 := cayonlib.ProposeInvoke(env, TCastInfo(), RPCInput{
		Function: "ReadCastInfo",
		Input:    ids,
	})
	stepLog3, invoke3 := cayonlib.ProposeInvoke(env, TPlot(), RPCInput{
		Function: "ReadPlot",
		Input:    aws.JSONValue{"plotId": movieInfo.PlotId},
	})
	stepLog4, invoke4 := cayonlib.ProposeInvoke(env, TMovieReview(), RPCInput{
		Function: "ReadMovieReviews",
		Input:    aws.JSONValue{"movieId": movieId},
	})
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		res, _ := cayonlib.AssignedSyncInvoke(env, TCastInfo(), stepLog2, invoke2)
		cayonlib.CHECK(mapstructure.Decode(res, &castInfos))
	}()
	go func() {
		defer wg.Done()
		res, _ := cayonlib.AssignedSyncInvoke(env, TPlot(), stepLog3, invoke3)
		cayonlib.CHECK(mapstructure.Decode(res, &plot))
	}()
	go func() {
		defer wg.Done()
		res, _ := cayonlib.AssignedSyncInvoke(env, TMovieReview(), stepLog4, invoke4)
		cayonlib.CHECK(mapstructure.Decode(res, &reviews))
	}()
	wg.Wait()
	return Page{CastInfos: castInfos, Reviews: reviews, MovieInfo: movieInfo, Plot: plot}
}
