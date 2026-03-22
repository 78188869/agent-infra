.PHONY: build run clean test lint fmt vet

# Binary output directory
BIN_DIR=bin
BINARY=$(BIN_DIR)/control-plane

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Main package
MAIN_PACKAGE=./cmd/control-plane

all: clean deps build

deps:
	$(GOMOD) tidy

build:
	mkdir -p $(BIN_DIR)
	GOPROXY=https://goproxy.cn,direct $(GOBUILD) -o $(BINARY) $(MAIN_PACKAGE)

run: build
	./$(BINARY)

test:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, skipping lint" && exit 0)
	golangci-lint run ./...

fmt:
	$(GOCMD) fmt ./...

vet:
	$(GOCMD) vet ./...

clean:
	$(GOCLEAN)
	rm -rf $(BIN_DIR)

# Development targets
dev: run

# Docker targets (will be used in later tasks)
docker-build:
	@echo "Docker build not implemented yet"

docker-run:
	@echo "Docker run not implemented yet"
