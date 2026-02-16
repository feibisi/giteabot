.PHONY: build test fmt tidy lint

build:
	go build ./...

test:
	go test ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

lint:
	go vet ./...
