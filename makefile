
***

## File: `Makefile`

```makefile
.PHONY: test test-coverage lint run-example

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run

run-example:
	go run examples/server/main.go

fmt:
	go fmt ./...

tidy:
	go mod tidy

run-example:
	go run examples/server/main.go
