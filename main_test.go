package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/alecthomas/participle"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

func parserSetup(attributes string) (*keyValue, error) {
	attr := &keyValue{}
	parser, err := participle.Build(&keyValue{})
	if err != nil {
		return attr, err
	}
	err = parser.ParseString(attributes, attr)
	return attr, err
}

func TestParserSimpleString(t *testing.T) {
	ast, err := parserSetup(`key="value"`)
	if err != nil {
		t.Fatal(err)
	}
	if len(ast.Attributes) != 1 {
		t.Fatalf("Expected one attribute, got %d", len(ast.Attributes))
	}
	if ast.Attributes[0].Key != "key" {
		t.Errorf("Expected key to be 'key', got '%s'", ast.Attributes[0].Key)
	}
	if *ast.Attributes[0].Value.String != "value" {
		t.Errorf("Expected Value to be 'value', got '%s'", *ast.Attributes[0].Value.String)
	}
	if ast.Attributes[0].Value.Number != nil {
		t.Errorf("Expected Number to be nil")
	}
}

func TestParserStringSingleQuotes(t *testing.T) {
	ast, err := parserSetup(`key='value'`)
	if err != nil {
		t.Fatal(err)
	}
	if len(ast.Attributes) != 1 {
		t.Fatalf("Expected one attribute, got %d", len(ast.Attributes))
	}
	if ast.Attributes[0].Key != "key" {
		t.Errorf("Expected key to be 'key', got '%s'", ast.Attributes[0].Key)
	}
	if *ast.Attributes[0].Value.String != "value" {
		t.Errorf("Expected Value to be 'value', got '%s'", *ast.Attributes[0].Value.String)
	}
	if ast.Attributes[0].Value.Number != nil {
		t.Errorf("Expected Number to be nil")
	}
}

func TestParserSimpleInt(t *testing.T) {
	ast, err := parserSetup(`key=123`)
	if err != nil {
		t.Fatal(err)
	}
	if len(ast.Attributes) != 1 {
		t.Fatalf("Expected one attribute, got %d", len(ast.Attributes))
	}
	if ast.Attributes[0].Key != "key" {
		t.Errorf("Expected key to be 'key', got '%s'", ast.Attributes[0].Key)
	}
	if *ast.Attributes[0].Value.Number != 123 {
		t.Errorf("Expected Value to be '123', got '%f'", *ast.Attributes[0].Value.Number)
	}
	if ast.Attributes[0].Value.String != nil {
		t.Error("Expected String to be nil")
	}
}

func TestParserSimpleBool(t *testing.T) {
	ast, err := parserSetup(`key=true`)
	if err != nil {
		t.Fatal(err)
	}
	if len(ast.Attributes) != 1 {
		t.Fatalf("Expected one attribute, got %d", len(ast.Attributes))
	}
	if ast.Attributes[0].Key != "key" {
		t.Errorf("Expected key to be 'key', got '%s'", ast.Attributes[0].Key)
	}
	if *ast.Attributes[0].Value.Bool != true {
		t.Errorf("Expected Value to be 'true', got '%v'", *ast.Attributes[0].Value.Bool)
	}
	if ast.Attributes[0].Value.Number != nil {
		t.Errorf("Expected Number to be nil")
	}
	if ast.Attributes[0].Value.String != nil {
		t.Errorf("Expected Number to be nil")
	}
}

func TestParserSimpleFloat(t *testing.T) {
	ast, err := parserSetup(`key=1.2`)
	if err != nil {
		t.Fatal(err)
	}
	if len(ast.Attributes) != 1 {
		t.Fatalf("Expected one attribute, got %d", len(ast.Attributes))
	}
	if ast.Attributes[0].Key != "key" {
		t.Errorf("Expected key to be 'key', got '%s'", ast.Attributes[0].Key)
	}
	if *ast.Attributes[0].Value.Number != 1.2 {
		t.Errorf("Expected Value to be '1.2', got '%f'", *ast.Attributes[0].Value.Number)
	}
}

func TestParserSimpleMap(t *testing.T) {
	ast, err := parserSetup("key=`{\"a\":\"b\"}`")
	if err != nil {
		t.Fatal(err)
	}
	if len(ast.Attributes) != 1 {
		t.Fatalf("Expected one attribute, got %d", len(ast.Attributes))
	}
	if ast.Attributes[0].Key != "key" {
		t.Errorf("Expected key to be 'key', got '%s'", ast.Attributes[0].Key)
	}
	expected := dynamoMap{
		"a": &dynamodb.AttributeValue{
			S: aws.String("b"),
		},
	}
	if !reflect.DeepEqual(*ast.Attributes[0].Value.Map, expected) {
		t.Errorf(`Expected Value to be '{"a":"b"}', got '%v'`, *ast.Attributes[0].Value.Map)
	}
}

func TestParserSimpleBinary(t *testing.T) {
	ast, err := parserSetup(`key={"fixtures/binary"}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(ast.Attributes) != 1 {
		t.Fatalf("Expected one attribute, got %d", len(ast.Attributes))
	}
	if ast.Attributes[0].Key != "key" {
		t.Errorf("Expected key to be 'key', got '%s'", ast.Attributes[0].Key)
	}
	reader := bytes.NewReader(*ast.Attributes[0].Value.Binary)
	bufferedReader := bufio.NewReader(reader)
	gzipReader, err := gzip.NewReader(bufferedReader)
	if err != nil {
		t.Fatalf("Error gzip decoding bytes %s", err)
	}
	uncompressed, err := ioutil.ReadAll(gzipReader)
	if err != nil {
		t.Fatalf("Error reading gzip bytes %s", err)
	}
	if string(uncompressed) != "Hello World\n" {
		t.Errorf("Expected uncompressed bytes to be 'Hello World', got %s", uncompressed)
	}
}

func TestParserMultipleStrings(t *testing.T) {
	ast, err := parserSetup(`key="foo",bar="baz"`)
	if err != nil {
		t.Fatal(err)
	}
	if len(ast.Attributes) != 2 {
		t.Fatalf("Expected two attributes, got %d", len(ast.Attributes))
	}
	if ast.Attributes[0].Key != "key" {
		t.Errorf("Expected key to be 'key', got '%s'", ast.Attributes[0].Key)
	}
	if *ast.Attributes[0].Value.String != "foo" {
		t.Errorf("Expected Value to be 'foo', got '%s'", *ast.Attributes[0].Value.String)
	}
	if ast.Attributes[1].Key != "bar" {
		t.Errorf("Expected key to be 'bar', got '%s'", ast.Attributes[1].Key)
	}
	if *ast.Attributes[1].Value.String != "baz" {
		t.Errorf("Expected Value to be 'baz', got '%s'", *ast.Attributes[1].Value.String)
	}
}

func TestParserMultipleStringsWithComma(t *testing.T) {
	ast, err := parserSetup(`key="foo,",bar="baz"`)
	if err != nil {
		t.Fatal(err)
	}
	if len(ast.Attributes) != 2 {
		t.Fatalf("Expected two attributes, got %d", len(ast.Attributes))
	}
	if ast.Attributes[0].Key != "key" {
		t.Errorf("Expected key to be 'key', got '%s'", ast.Attributes[0].Key)
	}
	if *ast.Attributes[0].Value.String != "foo," {
		t.Errorf("Expected Value to be 'foo', got '%s'", *ast.Attributes[0].Value.String)
	}
	if ast.Attributes[1].Key != "bar" {
		t.Errorf("Expected key to be 'bar', got '%s'", ast.Attributes[1].Key)
	}
	if *ast.Attributes[1].Value.String != "baz" {
		t.Errorf("Expected Value to be 'baz', got '%s'", *ast.Attributes[1].Value.String)
	}
}

func TestParserStringSet(t *testing.T) {
	ast, err := parserSetup(`key=("foo", "bar")`)
	if err != nil {
		t.Fatal(err)
	}
	if len(ast.Attributes) != 1 {
		t.Fatalf("Expected one attributes, got %d", len(ast.Attributes))
	}
	if ast.Attributes[0].Key != "key" {
		t.Errorf("Expected key to be 'key', got '%s'", ast.Attributes[0].Key)
	}
	if len((*ast.Attributes[0].Value).Set) != 2 {
		t.Errorf("Expected Set to contain 2 values, got %d", len((*ast.Attributes[0].Value).Set))
	}
	if *(*ast.Attributes[0].Value).Set[0].String != "foo" {
		t.Errorf("Expected Set's first value to be foo, got %s", *(*ast.Attributes[0].Value).Set[0].String)
	}
}

func TestParserNumberSet(t *testing.T) {
	ast, err := parserSetup(`key=(123, 45.1)`)
	if err != nil {
		t.Fatal(err)
	}
	if len(ast.Attributes) != 1 {
		t.Fatalf("Expected one attributes, got %d", len(ast.Attributes))
	}
	if ast.Attributes[0].Key != "key" {
		t.Errorf("Expected key to be 'key', got '%s'", ast.Attributes[0].Key)
	}
	if len((*ast.Attributes[0].Value).Set) != 2 {
		t.Errorf("Expected Set to contain 2 values, got %d", len((*ast.Attributes[0].Value).Set))
	}
	if *(*ast.Attributes[0].Value).Set[1].Number != 45.1 {
		t.Errorf("Expected Set's first value to be 45.1, got %f", *(*ast.Attributes[0].Value).Set[0].Number)
	}
}

func TestParserList(t *testing.T) {
	ast, err := parserSetup(`key=["a",12,3.0,true,[1,2,"b"]]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(ast.Attributes) != 1 {
		t.Fatalf("Expected one attributes, got %d", len(ast.Attributes))
	}
	if ast.Attributes[0].Key != "key" {
		t.Errorf("Expected key to be 'key', got '%s'", ast.Attributes[0].Key)
	}
	if len((*ast.Attributes[0].Value).List) != 5 {
		t.Errorf("Expected Set to contain 2 values, got %d", len((*ast.Attributes[0].Value).Set))
	}
	if *(*ast.Attributes[0].Value).List[4].List[0].Number != 1 {
		t.Errorf("Expected Set's value to be 1, got %f", *(*ast.Attributes[0].Value).List[4].List[0].Number)
	}
}

func TestParserMultipleInts(t *testing.T) {
	ast, err := parserSetup(`key=12,bar=2.1`)
	if err != nil {
		t.Fatal(err)
	}
	if len(ast.Attributes) != 2 {
		t.Fatalf("Expected two attributes, got %d", len(ast.Attributes))
	}
	if ast.Attributes[0].Key != "key" {
		t.Errorf("Expected key to be 'key', got '%s'", ast.Attributes[0].Key)
	}
	if *ast.Attributes[0].Value.Number != 12 {
		t.Errorf("Expected Value to be '12', got '%f'", *ast.Attributes[0].Value.Number)
	}
	if ast.Attributes[1].Key != "bar" {
		t.Errorf("Expected key to be 'bar', got '%s'", ast.Attributes[1].Key)
	}
	if *ast.Attributes[1].Value.Number != 2.1 {
		t.Errorf("Expected Value to be 'baz', got '%f'", *ast.Attributes[1].Value.Number)
	}
}

type mockDynamo struct {
	dynamodbiface.DynamoDBAPI
}

func (d *mockDynamo) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	return &dynamodb.GetItemOutput{
		Item: map[string]*dynamodb.AttributeValue{
			"string": {
				S: aws.String("bar"),
			},
			"number": {
				N: aws.String("123.4"),
			},
		},
	}, nil
}

func (d *mockDynamo) PutItem(*dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	return &dynamodb.PutItemOutput{}, nil
}

func TestGetString(t *testing.T) {
	args := ddbArgs{
		Client:  &mockDynamo{},
		Command: "get",
		Arguments: &keyValue{
			Attributes: []*attribute{
				{
					Key: "string",
					Value: &value{
						String: aws.String("bar"),
					},
				},
			},
		},
		Table: "testing",
	}
	output, err := run(args)
	if err != nil {
		t.Fatalf("Expected no error, but got %s", err)
	}
	if output != `{"number":123.4,"string":"bar"}` {
		t.Errorf("Expected result to be 'bar', got '%s'", output)
	}
}

func TestSetStringSet(t *testing.T) {
	args := ddbArgs{
		Client:  &mockDynamo{},
		Command: "set",
		Arguments: &keyValue{
			Attributes: []*attribute{
				{
					Key: "stringset",
					Value: &value{
						Set: []*value{
							{
								String: aws.String("foo"),
							},
							{
								String: aws.String("bar"),
							},
						},
					},
				},
			},
		},
		Table: "testing",
	}
	_, err := run(args)
	if err != nil {
		t.Fatalf("Expected no error, but got %s", err)
	}
}

func TestSetMap(t *testing.T) {
	args := ddbArgs{
		Client:  &mockDynamo{},
		Command: "set",
		Arguments: &keyValue{
			Attributes: []*attribute{
				{
					Key: "map",
					Value: &value{
						Map: &dynamoMap{
							"key": &dynamodb.AttributeValue{
								S: aws.String("foo"),
							},
						},
					},
				},
			},
		},
		Table: "testing",
	}
	_, err := run(args)
	if err != nil {
		t.Fatalf("Expected no error, but got %s", err)
	}
}

func TestSetBinary(t *testing.T) {
	compressedBytes := binary([]byte{31, 139, 8, 8, 236, 18, 59, 92, 0, 3, 116, 101, 115, 116, 0, 243, 72, 205, 201, 201, 87, 8, 207, 47, 202, 73, 225, 2, 0, 227, 229, 149, 176, 12, 0, 0, 0})
	args := ddbArgs{
		Client:  &mockDynamo{},
		Command: "set",
		Arguments: &keyValue{
			Attributes: []*attribute{
				{
					Key: "binary",
					Value: &value{
						Binary: &compressedBytes,
					},
				},
			},
		},
		Table: "testing",
	}
	_, err := run(args)
	if err != nil {
		t.Fatalf("Expected no error, but got %s", err)
	}
}

func TestSetBinarySet(t *testing.T) {
	compressedBytes := binary([]byte{31, 139, 8, 8, 236, 18, 59, 92, 0, 3, 116, 101, 115, 116, 0, 243, 72, 205, 201, 201, 87, 8, 207, 47, 202, 73, 225, 2, 0, 227, 229, 149, 176, 12, 0, 0, 0})
	args := ddbArgs{
		Client:  &mockDynamo{},
		Command: "set",
		Arguments: &keyValue{
			Attributes: []*attribute{
				{
					Key: "binarySet",
					Value: &value{
						Set: []*value{
							{
								Binary: &compressedBytes,
							},
							{
								Binary: &compressedBytes,
							},
						},
					},
				},
			},
		},
		Table: "testing",
	}
	_, err := run(args)
	if err != nil {
		t.Fatalf("Expected no error, but got %s", err)
	}
}

func TestSetList(t *testing.T) {
	args := ddbArgs{
		Client:  &mockDynamo{},
		Command: "set",
		Arguments: &keyValue{
			Attributes: []*attribute{
				{
					Key: "list",
					Value: &value{
						List: []*value{
							{
								String: aws.String("foo"),
							},
							{
								Number: aws.Float64(1),
							},
						},
					},
				},
			},
		},
		Table: "testing",
	}
	_, err := run(args)
	if err != nil {
		t.Fatalf("Expected no error, but got %s", err)
	}
}

func TestSetNestedList(t *testing.T) {
	args := ddbArgs{
		Client:  &mockDynamo{},
		Command: "set",
		Arguments: &keyValue{
			Attributes: []*attribute{
				{
					Key: "list",
					Value: &value{
						List: []*value{
							{
								String: aws.String("foo"),
							},
							{
								List: []*value{
									{
										String: aws.String("bar"),
									},
									{
										Number: aws.Float64(12.0),
									},
								},
							},
						},
					},
				},
			},
		},
		Table: "testing",
	}
	_, err := run(args)
	if err != nil {
		t.Fatalf("Expected no error, but got %s", err)
	}
}
