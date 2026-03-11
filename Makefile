.PHONY: build test lint clean format

BINARY_NAME=git-lord
BUILD_DIR=bin

build:
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/git-lord

test:
	go test -count=1 -v ./...

lint:
	golangci-lint run

format:
	gofmt -w .

clean:
	rm -rf $(BUILD_DIR)
