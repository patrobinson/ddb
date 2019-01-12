package main

import (
	"encoding/json"
	"flag"
	"fmt"
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
	Set    []*value `| "(" { @@ [ "," ] } ")"`
	List   []*value `| "[" { @@ [ "," ] } "]"`
}

func valueToAttribute(v *value) *dynamodb.AttributeValue {
	switch {
	case v.String != nil:
		return &dynamodb.AttributeValue{
			S: v.String,
		}
	case v.Bool != nil:
		return &dynamodb.AttributeValue{
			BOOL: v.Bool,
		}
	case v.Set != nil:
		if ok, stringSet := allString(v.Set); ok {
			return &dynamodb.AttributeValue{
				SS: stringSet,
			}
		} else if ok, numberSet := allNumber(v.Set); ok {
			return &dynamodb.AttributeValue{
				NS: numberSet,
			}
		} else {
			panic("Invalid values found in Set. Must be all strings or all numbers")
		}
	case v.Number != nil:
		return &dynamodb.AttributeValue{
			N: convertFloatToString(v.Number),
		}
	case v.List != nil:
		return &dynamodb.AttributeValue{
			L: convertList(v.List),
		}
	}

	panic("Unable to convert value into AttributeValue")
}

func convertFloatToString(num *float64) *string {
	return aws.String(strconv.FormatFloat(*num, 'E', -1, 64))
}

func convertList(list []*value) []*dynamodb.AttributeValue {
	listValue := []*dynamodb.AttributeValue{}
	for _, a := range list {
		listValue = append(listValue, valueToAttribute(a))
	}
	return listValue
}

func allString(set []*value) (bool, []*string) {
	stringSet := []*string{}
	for _, v := range set {
		if v.String == nil {
			return false, stringSet
		}
		stringSet = append(stringSet, v.String)
	}
	return true, stringSet
}

func allNumber(set []*value) (bool, []*string) {
	numberSet := []*string{}
	for _, v := range set {
		if v.Number == nil {
			return false, numberSet
		}
		numberSet = append(numberSet, convertFloatToString(v.Number))
	}
	return true, numberSet
}

type ddbArgs struct {
	Client    dynamodbiface.DynamoDBAPI
	Table     string
	Command   string
	Arguments *keyValue
}

func main() {
	table := flag.String("table", "", "The name of the table")
	command := flag.String("command", "get", "The command, to get or set values")
	statement := flag.String("statement", "", "A comma seperated list of key=value pairs to get or set in dynamo. Strings must be quoted (remember to escape them from your shell).")
	endpoint := flag.String("endpoint", "", "Endpoint URL for DynamoDB. Useful for testing with local DynamoDB")
	flag.Parse()

	usage := "Usage: ddb -table <table-name> -command <get|set> -statement \"<key='value',key=123>\""
	if *command != "get" && *command != "set" {
		panic(usage)
	}
	if *table == "" {
		panic(usage)
	}
	if *statement == "" {
		panic(usage)
	}

	parser, err := participle.Build(&keyValue{})
	if err != nil {
		panic(err)
	}
	attr := &keyValue{}
	parser.ParseString(*statement, attr)

	for _, a := range attr.Attributes {
		if a.Value == nil {
			panic(fmt.Sprintf("Invalid statement %s", *statement))
		}
	}

	if *command == "get" && len(attr.Attributes) > 1 {
		panic("Expected one key=value pair for a get request")
	}

	sess := session.New()
	if *endpoint != "" {
		sess.Config.Endpoint = endpoint
	}

	result, err := run(ddbArgs{
		Client:    dynamodb.New(sess),
		Table:     *table,
		Command:   *command,
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

func get(c dynamodbiface.DynamoDBAPI, table, key string, v *value) (string, error) {
	k := map[string]*dynamodb.AttributeValue{
		key: valueToAttribute(v),
	}

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
		item[attr.Key] = valueToAttribute(attr.Value)
	}
	_, err := args.Client.PutItem(&dynamodb.PutItemInput{
		TableName: &args.Table,
		Item:      item,
	})
	return err
}
