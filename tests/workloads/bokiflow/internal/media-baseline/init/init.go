package main

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/eniac/Beldi/internal/media-baseline/core"
	"github.com/eniac/Beldi/pkg/beldilib"
	"github.com/lithammer/shortuuid"
)

var services = []string{"CastInfo", "ComposeReview", "Frontend", "MovieId", "MovieInfo", "MovieReview", "Page",
	"Plot", "Rating", "ReviewStorage", "Text", "UniqueId", "User", "UserReview"}

func tables(baseline bool) {
	if baseline {
		panic("Not implemented for baseline")
	} else {
		for {
			tablenames := []string{}
			for _, service := range services {
				beldilib.CreateLambdaTables(service)
				tablenames = append(tablenames, service)
			}
			if beldilib.WaitUntilAllActive(tablenames) {
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
			beldilib.DeleteLambdaTables(service)
			// beldilib.WaitUntilAllDeleted([]string{service})
		}
	}
}

func user(baseline bool) {
	for i := 0; i < 1000; i++ {
		userId := fmt.Sprintf("user%d", i)
		username := fmt.Sprintf("username_%d", i)
		password := fmt.Sprintf("password_%d", i)
		hasher := sha512.New()
		salt := shortuuid.New()
		hasher.Write([]byte(password + salt))
		passwordHash := hex.EncodeToString(hasher.Sum(nil))
		user := core.User{
			UserId:    userId,
			FirstName: "firstname",
			LastName:  "lastname",
			Username:  username,
			Password:  passwordHash,
			Salt:      salt,
		}
		beldilib.Populate("User", username, user, baseline)
	}
}

func movie(baseline bool, file string) {
	data, err := ioutil.ReadFile(file)
	beldilib.CHECK(err)
	var movies []core.MovieInfo
	err = json.Unmarshal(data, &movies)
	beldilib.CHECK(err)
	for _, movie := range movies {
		beldilib.Populate("MovieInfo", movie.MovieId, movie, baseline)
		beldilib.Populate("Plot", movie.MovieId, aws.JSONValue{"plotId": movie.MovieId, "plot": "plot"}, baseline)
		beldilib.Populate("MovieId", movie.Title, aws.JSONValue{"movieId": movie.MovieId, "title": movie.Title}, baseline)
	}
}

func populate(baseline bool, file string) {
	user(baseline)
	movie(baseline, file)
}

func health_check() {
	tablename := "MovieId"
	key := "The Highwaymen"
	item := beldilib.LibRead(tablename, aws.JSONValue{"K": key}, []string{"V"})
	log.Printf("[INFO] Read data from DB: %v", item)
	if len(item) == 0 {
		panic("read data from DB failed")
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)

	option := os.Args[1]
	if option == "health_check" {
		health_check()
		return
	}

	baseline := os.Args[2] == "baseline"
	if option == "create" {
		tables(baseline)
	} else if option == "populate" {
		populate(baseline, os.Args[3])
	} else if option == "clean" {
		deleteTables(baseline)
	}
}
