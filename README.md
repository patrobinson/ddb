# DDB

A Command Line tool for easily interacting with DynamoDB

## Usage

Install:
```
go get -u github.com/patrobinson/ddb
```

Get the contents of an item where foo=bar. Note the quotes are important:
```
ddb -table test -command get -statement 'foo="bar"'
```

## Supported Datatypes

- [x] String
- [x] Number
- [x] Bool
- [x] Number Set
- [x] String Set
- [x] List
- [ ] Binary Set
- [ ] Binary
- [ ] Map

## Local development

```
docker run -d --rm -p 8000:8000 amazon/dynamodb-local

aws dynamodb --endpoint-url http://localhost:8000 create-table --table-name test --attribute-definitions AttributeName=foo,AttributeType=S --key-schema AttributeName=foo,KeyType=HASH --provisioned-throughput ReadCapacityUnits=1,WriteCapacityUnits=1

go run main.go -endpoint http://localhost:8000 -table test -command set -statement 'foo="bar"'
go run main.go -endpoint http://localhost:8000 -table test -command get -statement 'foo="bar"'
```
