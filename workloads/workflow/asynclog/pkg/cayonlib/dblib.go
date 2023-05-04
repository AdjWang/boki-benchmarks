package cayonlib

import (
	"log"

	// "fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"

	// "github.com/mitchellh/mapstructure"
	// "strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	// "github.com/lithammer/shortuuid"
)

func LibRead(tablename string, key aws.JSONValue, projection []string) aws.JSONValue {
	Key, err := dynamodbattribute.MarshalMap(key)
	CHECK(err)
	var res *dynamodb.GetItemOutput
	if len(projection) == 0 {
		res, err = DBClient.GetItem(&dynamodb.GetItemInput{
			TableName:      aws.String(kTablePrefix + tablename),
			Key:            Key,
			ConsistentRead: aws.Bool(true),
		})
		CHECK(err)
	} else {
		expr, err := expression.NewBuilder().WithProjection(BuildProjection(projection)).Build()
		CHECK(err)
		res, err = DBClient.GetItem(&dynamodb.GetItemInput{
			TableName:                aws.String(kTablePrefix + tablename),
			Key:                      Key,
			ProjectionExpression:     expr.Projection(),
			ExpressionAttributeNames: expr.Names(),
			ConsistentRead:           aws.Bool(true),
		})
		CHECK(err)
	}
	item := aws.JSONValue{}
	err = dynamodbattribute.UnmarshalMap(res.Item, &item)
	CHECK(err)
	return item
}

func LibWrite(tablename string, key aws.JSONValue,
	update map[expression.NameBuilder]expression.OperandBuilder) {
	Key, err := dynamodbattribute.MarshalMap(key)
	CHECK(err)
	if len(update) == 0 {
		panic("update never be empty")
	}
	updateBuilder := expression.UpdateBuilder{}
	for k, v := range update {
		updateBuilder = updateBuilder.Set(k, v)
	}
	builder := expression.NewBuilder().WithUpdate(updateBuilder)
	expr, err := builder.Build()
	CHECK(err)
	_, err = DBClient.UpdateItem(&dynamodb.UpdateItemInput{
		TableName:                 aws.String(kTablePrefix + tablename),
		Key:                       Key,
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})
	CHECK(err)
}

func LibScanWithLast(tablename string, projection []string, last map[string]*dynamodb.AttributeValue) []aws.JSONValue {
	var res *dynamodb.ScanOutput
	var err error
	if last == nil {
		if len(projection) == 0 {
			expr, err := expression.NewBuilder().Build()
			CHECK(err)
			res, err = DBClient.Scan(&dynamodb.ScanInput{
				TableName:                 aws.String(kTablePrefix + tablename),
				ExpressionAttributeNames:  expr.Names(),
				ExpressionAttributeValues: expr.Values(),
				FilterExpression:          expr.Filter(),
				ConsistentRead:            aws.Bool(true),
			})
			CHECK(err)
		} else {
			expr, err := expression.NewBuilder().WithProjection(BuildProjection(projection)).Build()
			CHECK(err)
			res, err = DBClient.Scan(&dynamodb.ScanInput{
				TableName:                 aws.String(kTablePrefix + tablename),
				ExpressionAttributeNames:  expr.Names(),
				ExpressionAttributeValues: expr.Values(),
				FilterExpression:          expr.Filter(),
				ProjectionExpression:      expr.Projection(),
				ConsistentRead:            aws.Bool(true),
			})
			CHECK(err)
		}
	} else {
		if len(projection) == 0 {
			expr, err := expression.NewBuilder().Build()
			CHECK(err)
			res, err = DBClient.Scan(&dynamodb.ScanInput{
				TableName:                 aws.String(kTablePrefix + tablename),
				ExpressionAttributeNames:  expr.Names(),
				ExpressionAttributeValues: expr.Values(),
				FilterExpression:          expr.Filter(),
				ConsistentRead:            aws.Bool(true),
				ExclusiveStartKey:         last,
			})
			CHECK(err)
		} else {
			expr, err := expression.NewBuilder().WithProjection(BuildProjection(projection)).Build()
			CHECK(err)
			res, err = DBClient.Scan(&dynamodb.ScanInput{
				TableName:                 aws.String(kTablePrefix + tablename),
				ExpressionAttributeNames:  expr.Names(),
				ExpressionAttributeValues: expr.Values(),
				FilterExpression:          expr.Filter(),
				ProjectionExpression:      expr.Projection(),
				ConsistentRead:            aws.Bool(true),
				ExclusiveStartKey:         last,
			})
			CHECK(err)
		}
	}
	var item []aws.JSONValue
	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &item)
	CHECK(err)
	if res.LastEvaluatedKey == nil || len(res.LastEvaluatedKey) == 0 {
		return item
	}
	log.Printf("[DEBUG] Exceed Scan limit")
	item = append(item, LibScanWithLast(tablename, projection, res.LastEvaluatedKey)...)
	return item
}

func LibScan(tablename string, projection []string) []aws.JSONValue {
	return LibScanWithLast(tablename, projection, nil)
}

func CondWrite(env *Env, tablename string, key string,
	update map[expression.NameBuilder]expression.OperandBuilder,
	cond expression.ConditionBuilder) {
	fnGetLoggedStepResult := func(preWriteLog *IntentLogEntry) bool {
		CheckLogDataField(preWriteLog, "type", "PreWrite")
		CheckLogDataField(preWriteLog, "table", tablename)
		CheckLogDataField(preWriteLog, "key", key)
		log.Printf("[INFO] Seen PreWrite log for step %d", preWriteLog.StepNumber)
		resultLog := FetchStepResultLog(env, preWriteLog.StepNumber /* catch= */, false)
		if resultLog != nil {
			CheckLogDataField(resultLog, "type", "PostWrite")
			CheckLogDataField(resultLog, "table", tablename)
			CheckLogDataField(resultLog, "key", key)
			log.Printf("[INFO] Seen PostWrite log for step %d", preWriteLog.StepNumber)
			return true
		} else {
			return false
		}
	}
	stepFuture, preWriteLog := AsyncProposeNextStep(env, aws.JSONValue{
		"type":  "PreWrite",
		"key":   key,
		"table": tablename,
	}, env.AsyncLogCtx.GetLastStepLogMeta())
	if stepFuture == nil {
		if fnGetLoggedStepResult(preWriteLog) {
			return
		} else {
			panic("unreachable")
		}
	}
	env.AsyncLogCtx.ChainStep(stepFuture.GetMeta())
	// DEBUG
	// log.Printf("[DEBUG] CondWrite before sync: %v", env.AsyncLogCtx)
	// sync
	env.LogTracer.TraceStart()
	err := env.AsyncLogCtx.Sync(gSyncTimeout)
	CHECK(err)
	// resolve cond
	logEntry, err := env.FaasEnv.AsyncSharedLogRead(env.FaasCtx, stepFuture.GetMeta())
	CHECK(err)
	if applied := ResolveLog(env, logEntry.Tags, logEntry.TagBuildMeta, logEntry.SeqNum); !applied {
		// discarded
		if ok := fnGetLoggedStepResult(preWriteLog); ok {
			return
		} else {
			panic("unreachable")
		}
	}
	env.LogTracer.TraceEnd()
	// DEBUG
	// log.Printf("[DEBUG] CondWrite resolved fsms: %v", env.FsmHub)

	Key, err := dynamodbattribute.MarshalMap(aws.JSONValue{"K": key})
	CHECK(err)
	condBuilder := expression.Or(
		expression.AttributeNotExists(expression.Name("VERSION")),
		expression.Name("VERSION").LessThan(expression.Value(preWriteLog.SeqNum)))
	if _, err = expression.NewBuilder().WithCondition(cond).Build(); err == nil {
		condBuilder = expression.And(condBuilder, cond)
	}
	updateBuilder := expression.UpdateBuilder{}
	for k, v := range update {
		updateBuilder = updateBuilder.Set(k, v)
	}
	updateBuilder = updateBuilder.
		Set(expression.Name("VERSION"), expression.Value(preWriteLog.SeqNum))
	expr, err := expression.NewBuilder().WithCondition(condBuilder).WithUpdate(updateBuilder).Build()
	CHECK(err)

	_, err = DBClient.UpdateItem(&dynamodb.UpdateItemInput{
		TableName:                 aws.String(kTablePrefix + tablename),
		Key:                       Key,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})
	if err != nil {
		AssertConditionFailure(err)
	}

	env.AsyncLogCtx.ChainStep(AsyncLogStepResult(env, env.InstanceId, preWriteLog.StepNumber, aws.JSONValue{
		"type":  "PostWrite",
		"key":   key,
		"table": tablename,
	}, stepFuture.GetMeta()).GetMeta())
}

func Write(env *Env, tablename string, key string, update map[expression.NameBuilder]expression.OperandBuilder) {
	CondWrite(env, tablename, key, update, expression.ConditionBuilder{})
}

func Read(env *Env, tablename string, key string) interface{} {
	step := env.StepNumber
	if intentLog := env.FsmHub.GetInstanceStepFsm().GetStepLog(step); intentLog != nil {
		env.StepNumber += 1

		CheckLogDataField(intentLog, "type", "Read")
		CheckLogDataField(intentLog, "key", key)
		CheckLogDataField(intentLog, "table", tablename)
		log.Printf("[INFO] Seen Read log for step %d", intentLog.StepNumber)
		return intentLog.Data["result"]
	} else {
		// log.Printf("[INFO] Read data from DB")
		item := LibRead(tablename, aws.JSONValue{"K": key}, []string{"V"})
		var res interface{}
		if tmp, ok := item["V"]; ok {
			res = tmp
		} else {
			res = nil
		}

		newLogFuture, intentReadLog := AsyncProposeNextStep(env, aws.JSONValue{
			"type":   "Read",
			"key":    key,
			"table":  tablename,
			"result": res,
		}, env.AsyncLogCtx.GetLastStepLogMeta())
		if newLogFuture == nil {
			CheckLogDataField(intentReadLog, "type", "Read")
			CheckLogDataField(intentReadLog, "key", key)
			CheckLogDataField(intentReadLog, "table", tablename)
			log.Printf("[INFO] Seen Read log for step %d", intentReadLog.StepNumber)
		} else {
			env.AsyncLogCtx.ChainStep(newLogFuture.GetMeta())
		}
		return intentReadLog.Data["result"]
	}
}

func Scan(env *Env, tablename string) interface{} {
	step := env.StepNumber
	if intentLog := env.FsmHub.GetInstanceStepFsm().GetStepLog(step); intentLog != nil {
		env.StepNumber += 1

		CheckLogDataField(intentLog, "type", "Scan")
		CheckLogDataField(intentLog, "table", tablename)
		log.Printf("[INFO] Seen Scan log for step %d", intentLog.StepNumber)
		return intentLog.Data["result"]
	} else {
		// log.Printf("[INFO] Scan data from DB")
		items := LibScan(tablename, []string{"V"})
		var res []interface{}
		for _, item := range items {
			res = append(res, item["V"])
		}

		newLogFuture, intentScanLog := AsyncProposeNextStep(env, aws.JSONValue{
			"type":   "Scan",
			"table":  tablename,
			"result": res,
		}, env.AsyncLogCtx.GetLastStepLogMeta())
		if newLogFuture == nil {
			CheckLogDataField(intentScanLog, "type", "Scan")
			CheckLogDataField(intentScanLog, "table", tablename)
			log.Printf("[INFO] Seen Scan log for step %d", intentScanLog.StepNumber)
		} else {
			env.AsyncLogCtx.ChainStep(newLogFuture.GetMeta())
		}
		return intentScanLog.Data["result"]
	}
}

func BuildProjection(names []string) expression.ProjectionBuilder {
	if len(names) == 0 {
		panic("Projection must > 0")
	}
	var builder expression.ProjectionBuilder
	for _, name := range names {
		builder = builder.AddNames(expression.Name(name))
	}
	return builder
}

func AssertConditionFailure(err error) {
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case dynamodb.ErrCodeConditionalCheckFailedException:
			return
		case dynamodb.ErrCodeResourceNotFoundException:
			log.Printf("ERROR: DyanombDB ResourceNotFound")
			return
		default:
			log.Printf("ERROR: %s", aerr)
			panic("ERROR detected")
		}
	} else {
		log.Printf("ERROR: %s", err)
		panic("ERROR detected")
	}
}
