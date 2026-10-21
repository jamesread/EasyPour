.PHONY: build test lint run

build:
	go build -o easypour-service .

test:
	go test ./...

lint:
	golangci-lint run ./...

run:
	go run .
