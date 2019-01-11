package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/alecthomas/participle"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

type keyValue struct {
	Attributes []*attribute `@@ { "," @@ }`
}

type attribute struct {
	Key   string `@Ident "="`
	Value *value `@@`
}

type value struct {
	String *string  `  @String`
	Number *float64 `| @(Float|Int)`
	Bool   *bool    `| (@"true" | "false")`
}

func attrify(v *value) dynamodb.AttributeValue {
	if v.String != nil {
		return dynamodb.AttributeValue{
			S: v.String,
		}
	}
	return dynamodb.AttributeValue{
		N: aws.String(strconv.FormatFloat(*v.Number, 'E', -1, 64)),
	}
}

type ddbArgs struct {
	Client    dynamodbiface.DynamoDBAPI
	Table     string
	Command   string
	Arguments *keyValue
}

func main() {
	usage := "Usage: ddb <table-name> <get|set> \"<key='value',key=123>\""
	args := os.Args
	if len(args) < 3 {
		panic(usage)
	}

	command := args[1]
	if command != "get" && command != "set" {
		panic(usage)
	}

	parser, err := participle.Build(&keyValue{})
	if err != nil {
		panic(err)
	}
	attr := &keyValue{}
	parser.ParseString(args[2], attr)

	if command == "get" && len(attr.Attributes) > 1 {
		panic("Expected one key=value pair for a get request")
	}

	result, err := run(ddbArgs{
		Client:    dynamodb.New(session.New()),
		Table:     args[0],
		Command:   args[1],
		Arguments: attr,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(result)
}

func run(args ddbArgs) (string, error) {
	if args.Command == "get" {
		return get(args.Client, args.Table, args.Arguments.Attributes[0].Key, args.Arguments.Attributes[0].Value)
	}
	return "", set(args)
}

func get(c dynamodbiface.DynamoDBAPI, table, key string, value *value) (string, error) {
	k := map[string]*dynamodb.AttributeValue{}
	resp, err := c.GetItem(&dynamodb.GetItemInput{
		TableName: &table,
		Key:       k,
	})
	if err != nil {
		return "", err
	}
	var result map[string]interface{}
	err = dynamodbattribute.UnmarshalMap(resp.Item, &result)
	if err != nil {
		return "", err
	}
	r, err := json.Marshal(result)
	return string(r), err
}

func set(args ddbArgs) error {
	item := make(map[string]*dynamodb.AttributeValue)
	for _, attr := range args.Arguments.Attributes {
		if attr.Value.String != nil {
			item[attr.Key].S = attr.Value.String
		} else if attr.Value.Bool != nil {
			item[attr.Key].BOOL = attr.Value.Bool
		} else if attr.Value.Number != nil {
			item[attr.Key].N = aws.String(strconv.FormatFloat(*attr.Value.Number, 'E', -1, 64))
		}

	}
	_, err := args.Client.PutItem(&dynamodb.PutItemInput{
		TableName: &args.Table,
		Item:      item,
	})
	return err
}
