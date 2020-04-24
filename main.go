package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
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
	Number *float64   ` @(Float|Int)`
	Bool   *bool      `| (@"true" | "false")`
	Set    []*value   `| "(" { @@ [ "," ] } ")"`
	List   []*value   `| "[" { @@ [ "," ] } "]"`
	Map    *dynamoMap `| @RawString`
	Binary *binary    `| "{" @String "}"`
	String *string    `| @(Ident|String)`
}

type binary []byte

func (b *binary) Capture(v []string) error {
	if len(v) != 1 {
		return fmt.Errorf("Expected one file name, got %d", len(v))
	}
	raw, err := ioutil.ReadFile(v[0])
	if err != nil {
		return fmt.Errorf("Error reading file: %s", err)
	}
	*b = raw
	return nil
}

type dynamoMap map[string]*dynamodb.AttributeValue

func (d *dynamoMap) Capture(v []string) error {
	if len(v) < 1 {
		return errors.New("Empty string detected, wanted JSON object")
	}
	if len(v) > 1 {
		return errors.New("Multiple JSON objects detected, wanted one")
	}
	var jsonBlob interface{}
	err := json.Unmarshal([]byte(v[0]), &jsonBlob)
	if err != nil {
		return err
	}
	av, err := dynamodbattribute.MarshalMap(jsonBlob)
	*d = dynamoMap(av)
	return nil
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
		} else if ok, binarySet := allBinary(v.Set); ok {
			return &dynamodb.AttributeValue{
				BS: binarySet,
			}
		} else {
			panic("Invalid values found in Set. Must be all strings, all numbers or all binary")
		}
	case v.Number != nil:
		return &dynamodb.AttributeValue{
			N: convertFloatToString(v.Number),
		}
	case v.List != nil:
		return &dynamodb.AttributeValue{
			L: convertListToAttributeValue(v.List),
		}
	case v.Map != nil:
		return &dynamodb.AttributeValue{
			M: map[string]*dynamodb.AttributeValue(*v.Map),
		}
	case v.Binary != nil:
		return &dynamodb.AttributeValue{
			B: []byte(*v.Binary),
		}
	}

	panic("Unable to convert value into AttributeValue")
}

func convertFloatToString(num *float64) *string {
	return aws.String(strconv.FormatFloat(*num, 'E', -1, 64))
}

func convertListToAttributeValue(list []*value) []*dynamodb.AttributeValue {
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

func allBinary(set []*value) (bool, [][]byte) {
	binarySet := [][]byte{}
	for _, v := range set {
		if v.Binary == nil {
			return false, binarySet
		}
		binarySet = append(binarySet, []byte(*v.Binary))
	}
	return true, binarySet
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

	usage := "Usage: ddb -table <table-name> -command <get|set|scan> -statement \"<key='value',key=123>\""
	if *command != "get" && *command != "set" && *command != "scan" {
		panic(usage)
	}
	if *table == "" {
		panic(usage)
	}

	sess := session.New()
	if *endpoint != "" {
		sess.Config.Endpoint = endpoint
	}

	if *command == "scan" {
		result, err := run(ddbArgs{
			Client:  dynamodb.New(sess),
			Table:   *table,
			Command: *command,
		})
		if err != nil {
			panic(err)
		}
		fmt.Println(result)
		return
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

	if *command == "get" && len(attr.Attributes) > 2 {
		panic("Expected one or two key=value pair(s) for a get request")
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
		return get(args.Client, args.Table, args.Arguments.Attributes)
	}
	if args.Command == "scan" {
		return scan(args.Client, args.Table)
	}
	return "", set(args)
}

func scan(c dynamodbiface.DynamoDBAPI, table string) (string, error) {
	var items []map[string]*dynamodb.AttributeValue
	err := c.ScanPages(&dynamodb.ScanInput{
		TableName: &table,
	}, func(output *dynamodb.ScanOutput, _lastPage bool) bool {
		items = append(items, output.Items...)
		return true
	})
	if err != nil {
		return "", err
	}

	var serialisedResult []map[string]interface{}
	err = dynamodbattribute.UnmarshalListOfMaps(items, &serialisedResult)
	if err != nil {
		return "", err
	}
	r, err := json.MarshalIndent(serialisedResult, "", "	")
	return string(r), err
}

func get(c dynamodbiface.DynamoDBAPI, table string, attributes []*attribute) (string, error) {
	key := map[string]*dynamodb.AttributeValue{}

	for _, attr := range attributes {
		k := attr.Key
		v := attr.Value

		key[k] = valueToAttribute(v)
	}

	resp, err := c.GetItem(&dynamodb.GetItemInput{
		TableName: &table,
		Key:       key,
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
