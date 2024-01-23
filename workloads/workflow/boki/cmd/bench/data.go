package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/lithammer/shortuuid"
)

type Cast struct {
	CastId     string `json:"CastId"`
	Character  string `json:"Character"`
	CastInfoId string `json:"CastInfoId"`
}
type MovieInfo struct {
	MovieId      string   `json:"MovieId"`
	Title        string   `json:"Title"`
	Casts        []Cast   `json:"Casts"`
	PlotId       string   `json:"PlotId"`
	ThumbnailIds []string `json:"ThumbnailIds"`
	PhotoIds     []string `json:"PhotoIds"`
	VideoIds     []string `json:"VideoIds"`
	AvgRating    float64  `json:"AvgRating"`
	NumRating    int32    `json:"NumRating"`
}

type MovieBenchData struct {
	ReqId    string
	ReviewId string
	Text     string
	MovieId  string
	Rating   int32
	UserId   string
}

type MovieBenchDataGenerator struct {
	movieInfos []MovieInfo
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func NewMovieBenchDataGenerator(dataFile string) *MovieBenchDataGenerator {
	res := &MovieBenchDataGenerator{}

	rawData, err := os.ReadFile(dataFile)
	if err != nil {
		panic(err)
	}
	var movieInfos []MovieInfo
	err = json.Unmarshal(rawData, &movieInfos)
	if err != nil {
		panic(err)
	}
	if len(movieInfos) != 1000 {
		log.Panicf("movie data read failed from %s len(rawData)=%d", dataFile, len(rawData))
	}
	res.movieInfos = movieInfos

	return res
}

func (gen *MovieBenchDataGenerator) GenerateReqData() MovieBenchData {
	randomMovie := gen.movieInfos[rand.Intn(len(gen.movieInfos))]
	return MovieBenchData{
		ReqId:    shortuuid.New(),
		ReviewId: shortuuid.New(),
		Text:     RandStringBytes(256),
		MovieId:  randomMovie.MovieId,
		Rating:   rand.Int31n(11),
		UserId:   fmt.Sprintf("user%02d", rand.Intn(100)),
	}
}
