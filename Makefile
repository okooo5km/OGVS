# ogvs - Go implementation of SVGO
# Copyright (c) 2024 okooo5km(十里)

BINARY_NAME := ogvs
BUILD_DIR := bin
GOFLAGS := -v

.PHONY: build test lint fmt vet clean run help

## build: Build the ogvs binary
build:
	go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ogvs

## test: Run all tests with race detection
test:
	go test -v -race -count=1 ./...

## test-cover: Run tests with coverage
test-cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## fmt: Format all Go files
fmt:
	gofmt -w .
	goimports -w .

## vet: Run go vet
vet:
	go vet ./...

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## run: Build and run
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

## help: Show this help message
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
