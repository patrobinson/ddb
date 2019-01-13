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

Bool types:
```
ddb -table books -command set -statement 'book="1984",bestseller=true'
```

String sets:
```
ddb -table authors -command set -statement 'author="George Orwell",books=("1984","Animal Farm")'
```

Number sets:
```
ddb -table authors -command set -statement 'author="George Orwell",isbns=(9780143566496,9780141036144,9780241341667)'
```

List:
```
ddb -table cricketers -command set -statement 'name="Sir Donald Bradman",testscores=[18,1,79,112,40,58,123,37]'
```

Map:
```
ddb -table cricketers -command set -statement 'country="Australia",players=`{"Tim Paine":{"Batting Avg": 34.78}}`'
```

Binary:
```
ddb -table cricketers -command set -statement 'country="Australia",players={"players.gz"}'
```

Binary Set:
```
ddb -table cricketers -command set -statement 'country="Australia",players=({"players1.gz"},{"players2.gz"})'
```

## Supported Datatypes

- [x] String
- [x] Number
- [x] Bool
- [x] Number Set
- [x] String Set
- [x] List
- [x] Map
- [ ] Binary Set
- [ ] Binary

## Local development

```
docker run -d --rm -p 8000:8000 amazon/dynamodb-local

aws dynamodb --endpoint-url http://localhost:8000 create-table --table-name test --attribute-definitions AttributeName=foo,AttributeType=S --key-schema AttributeName=foo,KeyType=HASH --provisioned-throughput ReadCapacityUnits=1,WriteCapacityUnits=1

go run main.go -endpoint http://localhost:8000 -table test -command set -statement 'foo="bar"'
go run main.go -endpoint http://localhost:8000 -table test -command get -statement 'foo="bar"'
```
