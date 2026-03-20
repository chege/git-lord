.PHONY: build test lint vet clean format

BINARY_NAME=git-lord
BUILD_DIR=bin

build:
	go build -buildvcs=false -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/git-lord

test:
	go test -count=1 -v ./...

lint:
	golangci-lint run

vet:
	go vet ./...

format:
	gofmt -w .

clean:
	rm -rf $(BUILD_DIR)
