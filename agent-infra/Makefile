.PHONY: all build run clean test lint fmt vet deps
.PHONY: docker-build-all docker-push docker-build-control-plane docker-build-frontend docker-build-cli-runner docker-build-wrapper
.PHONY: k8s-apply k8s-delete k8s-logs k8s-status
.PHONY: db-migrate db-rollback

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

# Go proxy settings
GOPROXY=https://goproxy.cn,direct

# Main package
MAIN_PACKAGE=./cmd/control-plane

# Docker settings
# TODO: Replace with actual container registry URL before deployment
REGISTRY?=registry.example.com/agent-infra
VERSION?=latest

# K8s settings
KUBECTL?=kubectl
NAMESPACE?=control-plane

# Database settings
MIGRATE_CMD?=$(GOCMD) run -tags migrate ./cmd/migrate

all: clean deps build

deps:
	$(GOMOD) tidy

build:
	mkdir -p $(BIN_DIR)
	GOPROXY=$(GOPROXY) $(GOBUILD) -o $(BINARY) $(MAIN_PACKAGE)

run: build
	./$(BINARY)

test:
	GOPROXY=$(GOPROXY) $(GOTEST) -v -race -coverprofile=coverage.out ./...

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

# =============================================================================
# Docker targets
# =============================================================================

docker-build-all: docker-build-control-plane docker-build-frontend docker-build-cli-runner docker-build-wrapper

docker-push:
	docker push $(REGISTRY)/control-plane:$(VERSION)
	docker push $(REGISTRY)/frontend:$(VERSION)
	docker push $(REGISTRY)/cli-runner:$(VERSION)
	docker push $(REGISTRY)/agent-wrapper:$(VERSION)

docker-build-control-plane:
	docker build -f deploy/dockerfiles/Dockerfile.control-plane -t $(REGISTRY)/control-plane:$(VERSION) .

docker-build-frontend:
	docker build -f deploy/dockerfiles/Dockerfile.frontend -t $(REGISTRY)/frontend:$(VERSION) ./web

docker-build-cli-runner:
	docker build -f deploy/dockerfiles/Dockerfile.cli-runner -t $(REGISTRY)/cli-runner:$(VERSION) .

docker-build-wrapper:
	docker build -f deploy/dockerfiles/Dockerfile.agent-wrapper -t $(REGISTRY)/agent-wrapper:$(VERSION) .

# =============================================================================
# Kubernetes targets
# =============================================================================

k8s-apply:
	$(KUBECTL) apply -f deploy/k8s/control-plane/ -n $(NAMESPACE)
	$(KUBECTL) apply -f deploy/k8s/sandbox/

k8s-delete:
	$(KUBECTL) delete -f deploy/k8s/control-plane/ -n $(NAMESPACE) --ignore-not-found
	$(KUBECTL) delete -f deploy/k8s/sandbox/ --ignore-not-found

k8s-logs:
	$(KUBECTL) logs -f deployment/control-plane -n $(NAMESPACE)

k8s-status:
	$(KUBECTL) get pods -n $(NAMESPACE)
	$(KUBECTL) get services -n $(NAMESPACE)

# =============================================================================
# Database targets
# =============================================================================

db-migrate:
	@echo "Running database migrations..."
	$(MIGRATE_CMD) up

db-rollback:
	@echo "Rolling back database migrations..."
	$(MIGRATE_CMD) down

# =============================================================================
# Utility targets
# =============================================================================

# Verify Go version
go-version:
	@echo "Go version: $$(go version)"
	@echo "GOROOT: $$(go env GOROOT)"
	@echo "GOPATH: $$(go env GOPATH)"
