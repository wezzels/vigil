# Makefile for VIGIL

.PHONY: all build test clean docker deps lint security

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
BINARY_WINDOWS=$(BINARY_NAME).exe

# Build flags
VERSION?=dev
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD)
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Docker
DOCKER_REGISTRY?=ghcr.io/wezzels
DOCKER_TAG?=latest

all: deps build test

## build: Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/vigil

## build-all: Build for all platforms
build-all:
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/vigil
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/vigil
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/vigil
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_WINDOWS) ./cmd/vigil

## test: Run all tests
test:
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...

## test-short: Run short tests
test-short:
	$(GOTEST) -v -short ./...

## test-e2e: Run end-to-end tests
test-e2e:
	$(GOTEST) -v -run TestE2E ./tests/e2e/...

## test-load: Run load tests
test-load:
	$(GOTEST) -v -run TestLoad ./tests/load/...

## test-chaos: Run chaos tests
test-chaos:
	$(GOTEST) -v -run TestChaos ./tests/chaos/...

## test-benchmark: Run benchmark tests
test-benchmark:
	$(GOTEST) -bench=. -benchmem ./tests/benchmarks/...

## coverage: Generate test coverage report
coverage: test
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## deps: Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## lint: Run linters
lint:
	golangci-lint run ./...

## security: Run security scan
security:
	gosec ./...

## docker-build: Build Docker images
docker-build:
	docker build -t $(DOCKER_REGISTRY)/vigil:$(DOCKER_TAG) .

## docker-push: Push Docker images
docker-push:
	docker push $(DOCKER_REGISTRY)/vigil:$(DOCKER_TAG)

## clean: Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf bin/
	rm -f coverage.out coverage.html

## fmt: Format code
fmt:
	$(GOCMD) fmt ./...

## vet: Run go vet
vet:
	$(GOCMD) vet ./...

## check: Run all checks
check: fmt vet lint security test

## install: Install binary
install:
	$(GOBUILD) -o $$GOPATH/bin/$(BINARY_NAME) ./cmd/vigil

## run: Run locally
run:
	$(GOCMD) run ./cmd/vigil

## db-migrate: Run database migrations
db-migrate:
	@echo "Run migrations with: psql -f db/migrations/*.sql"

## db-reset: Reset database
db-reset:
	@echo "Reset database with: dropdb vigil && createdb vigil"

## help: Show this help
help:
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'