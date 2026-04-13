.PHONY: all build test lint clean coverage docker-build docker-push help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
BINARY_NAME=vigil
BINARY_UNIX=$(BINARY_NAME)_unix

# Version
VERSION?=0.0.1
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Directories
PKG_DIR=./pkg
CMD_DIR=./cmd
APP_DIR=./apps

# Docker
DOCKER_REGISTRY?=ghcr.io
DOCKER_IMAGE?=vigil
DOCKER_TAG?=$(VERSION)

all: deps build test lint

## build: Build all binaries
build:
	@echo "Building..."
	$(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME) $(CMD_DIR)/vigil/main.go
	@echo "Build complete: bin/$(BINARY_NAME)"

## build-all: Build for all platforms
build-all:
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)/vigil/main.go
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)/vigil/main.go
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)/vigil/main.go
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)/vigil/main.go
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)/vigil/main.go
	@echo "Cross-compilation complete"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...

## test-short: Run short tests
test-short:
	@echo "Running short tests..."
	$(GOTEST) -v -short -race ./...

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -tags=integration -race ./...

## test-all: Run all tests including integration
test-all: test test-integration
	@echo "All tests complete"

## coverage: Generate test coverage report
coverage:
	@echo "Generating coverage report..."
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## coverage-total: Show total coverage
coverage-total:
	@echo "Calculating total coverage..."
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic ./... 2>/dev/null
	$(GOCMD) tool cover -func=coverage.out | grep total

## lint: Run golangci-lint
lint:
	@echo "Running linters..."
	@which golangci-lint > /dev/null 2>&1 || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin
	golangci-lint run ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...
	goimports -w .

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) verify

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf bin/
	rm -f coverage.out coverage.html

## docker-build: Build Docker images
docker-build:
	@echo "Building Docker images..."
	docker build -t $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG) -f Dockerfile .
	docker build -t $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)-opir:$(DOCKER_TAG) -f apps/opir-ingest/Dockerfile ./apps/opir-ingest/ || true
	docker build -t $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)-fusion:$(DOCKER_TAG) -f apps/sensor-fusion/Dockerfile ./apps/sensor-fusion/ || true
	docker build -t $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)-warning:$(DOCKER_TAG) -f apps/missile-warning-engine/Dockerfile ./apps/missile-warning-engine/ || true

## docker-push: Push Docker images
docker-push:
	@echo "Pushing Docker images..."
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG) || true

## security: Run security scan
security:
	@echo "Running security scan..."
	@which gosec > /dev/null 2>&1 || curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin
	gosec ./...

## benchmark: Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

## install: Install binaries to GOPATH/bin
install:
	@echo "Installing..."
	$(GOBUILD) $(LDFLAGS) -o $(shell go env GOPATH)/bin/$(BINARY_NAME) $(CMD_DIR)/vigil/main.go

## proto: Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@which protoc > /dev/null 2>&1 || (echo "Please install protoc" && exit 1)
	protoc --go_out=. --go_opt=paths=source_relative proto/*.proto

## generate: Run go generate
generate:
	@echo "Running go generate..."
	$(GOCMD) generate ./...

## check: Run all checks (test, lint, security)
check: test lint security
	@echo "All checks passed!"

## help: Show this help
help:
	@echo "VIGIL Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'