.PHONY: build test lint clean format

BINARY_NAME=git-lord
BUILD_DIR=bin

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/git-lord

test:
	go test -v ./...

lint:
	golangci-lint run

format:
	gofmt -w .

clean:
	rm -rf $(BUILD_DIR)
