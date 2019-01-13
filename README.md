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

Create an Item with string and number types:
```
ddb -table books -command set -statement 'book="1984",author="George Orwell",isbn=9780143566496'
```

Create an Item with bool types:
```
ddb -table books -command set -statement 'book="1984",bestseller=true'
```

Create an Item with string sets:
```
ddb -table authors -command set -statement 'author="George Orwell",books=("1984","Animal Farm")'
```

Create an Item with number sets:
```
ddb -table authors -command set -statement 'author="George Orwell",isbns=(9780143566496,9780141036144,9780241341667)'
```

Create an Item with a list:
```
ddb -table cricketers -command set -statement 'name="Sir Donald Bradman,testscores=[18,1,79,112,40,58,123,37]'
```

## Supported Datatypes

- [x] String
- [x] Number
- [x] Bool
- [x] Number Set
- [x] String Set
- [x] List
- [ ] Map
- [ ] Binary Set
- [ ] Binary

## Local development

```
docker run -d --rm -p 8000:8000 amazon/dynamodb-local

aws dynamodb --endpoint-url http://localhost:8000 create-table --table-name test --attribute-definitions AttributeName=foo,AttributeType=S --key-schema AttributeName=foo,KeyType=HASH --provisioned-throughput ReadCapacityUnits=1,WriteCapacityUnits=1

go run main.go -endpoint http://localhost:8000 -table test -command set -statement 'foo="bar"'
go run main.go -endpoint http://localhost:8000 -table test -command get -statement 'foo="bar"'
```
