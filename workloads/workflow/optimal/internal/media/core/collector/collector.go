package main

import (
	"github.com/aws/aws-lambda-go/lambda"
)

func Handler() {
	// var wg sync.WaitGroup
	// wg.Add(2)
	// go func() {
	// 	defer wg.Done()
	// 	beldilib.RestartAll("Frontend")
	// }()
	// go func() {
	// 	defer wg.Done()
	// 	beldilib.RestartAll("ComposeReview")
	// }()
	// wg.Wait()
	panic("not implemented")
}

func main() {
	lambda.Start(Handler)
}
