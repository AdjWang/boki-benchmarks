package main

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-cmp/cmp"
)

func TestLastpx(t *testing.T) {
	event := aws.JSONValue{"portfolio": "1234"}
	res := handler(event)
	expected := aws.JSONValue{
		"statusCode": 200,
		"body": aws.JSONValue{
			"valid":     true,
			"portfolio": "1234",
		}}
	if !cmp.Equal(res, expected) {
		t.Fatal(res, expected)
	}
}
