test:
	@go test -timeout 30s

get:
	@go get

release:
	GOOS=darwin GOARCH=amd64 go build -o ddb-darwin-amd64
	GOOS=linux GOARCH=amd64 go build -o ddb-linux-amd64
	GOOS=windows GOARCH=amd64 go build -o ddb-windows-amd64
