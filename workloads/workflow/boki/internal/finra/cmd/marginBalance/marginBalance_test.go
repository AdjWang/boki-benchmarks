package main

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-cmp/cmp"
)

func TestMarginBalance(t *testing.T) {
	event := aws.JSONValue{
		"lastpx": map[string]interface{}{"body": map[string]interface{}{"portfolio": "1234", "valid": true}, "statusCode": 200},
		"marketdata": map[string]interface{}{"body": map[string]interface{}{
			"marketData": map[string]interface{}{
				"AMZN": 154.85000610351562,
				"GOOG": 143.54310607910156,
				"MSFT": 386.1000061035156}},
			"statusCode": 200},
		"side":    map[string]interface{}{"body": map[string]interface{}{"portfolio": "1234", "valid": true}, "statusCode": 200},
		"trddate": map[string]interface{}{"body": map[string]interface{}{"portfolio": "1234", "valid": true}, "statusCode": 200},
		"volume":  map[string]interface{}{"body": map[string]interface{}{"portfolio": "1234", "valid": true}, "statusCode": 200},
	}
	events := []aws.JSONValue{event}
	res := handler(events)
	expected := aws.JSONValue{
		"statusCode": 200,
		"body": aws.JSONValue{
			"validFormat":     true,
			"marginSatisfied": true,
		}}
	if !cmp.Equal(res, expected) {
		t.Fatal(res, expected)
	}
}
