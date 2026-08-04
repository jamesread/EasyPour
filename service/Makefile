.PHONY: build test lint run

build:
	go build -o easypour-service .

test:
	go test ./...

lint:
	go vet ./...

run:
	go run main.go
