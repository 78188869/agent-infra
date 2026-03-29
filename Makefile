.PHONY: all build run clean test lint fmt vet deps local
.PHONY: docker-build-all docker-push docker-build-control-plane docker-build-frontend docker-build-sandbox
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

# Test specific package: make test-pkg PKG=internal/service
test-pkg:
	GOPROXY=$(GOPROXY) $(GOTEST) -v -race $(PKG)

# Test specific module shortcuts
test-model:
	GOPROXY=$(GOPROXY) $(GOTEST) -v -race ./internal/model/...

test-repo:
	GOPROXY=$(GOPROXY) $(GOTEST) -v -race ./internal/repository/...

test-service:
	GOPROXY=$(GOPROXY) $(GOTEST) -v -race ./internal/service/...

test-handler:
	GOPROXY=$(GOPROXY) $(GOTEST) -v -race ./internal/api/handler/...

test-coverage: test
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-short:
	GOPROXY=$(GOPROXY) $(GOTEST) -v -short ./...

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

# Local development (SQLite + miniredis, no external deps)
local:
	@mkdir -p data logs
	APP_ENV=local go run ./cmd/control-plane

# =============================================================================
# Docker targets
# =============================================================================

docker-build-all: docker-build-control-plane docker-build-frontend docker-build-sandbox

docker-push:
	docker push $(REGISTRY)/control-plane:$(VERSION)
	docker push $(REGISTRY)/frontend:$(VERSION)
	docker push $(REGISTRY)/sandbox:$(VERSION)

docker-build-control-plane:
	docker build -f deploy/dockerfiles/Dockerfile.control-plane -t $(REGISTRY)/control-plane:$(VERSION) .

docker-build-frontend:
	docker build -f deploy/dockerfiles/Dockerfile.frontend -t $(REGISTRY)/frontend:$(VERSION) ./web

docker-build-sandbox:
	docker build -f scripts/wrapper/Dockerfile -t $(REGISTRY)/sandbox:$(VERSION) .

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

go-version:
	@echo "Go version: $$(go version)"
	@echo "GOROOT: $$(go env GOROOT)"
	@echo "GOPATH: $$(go env GOPATH)"

# =============================================================================
# Worktree targets
# =============================================================================

WORKTREE_DIR?=./worktrees

# Create worktree for an issue
worktree-create:
	@read -p "Issue number: " ISSUE; \
	./scripts/create-worktree.sh $$ISSUE

# List all worktrees
worktree-list:
	git worktree list

# Clean up merged worktrees
worktree-clean:
	@echo "Cleaning up merged worktrees..."
	@git worktree list --porcelain | grep "^worktree" | cut -d' ' -f2 | while read -r dir; do \
		basename=$$(basename "$$dir"); \
		branch=$${basename#issue-}; \
		if git branch --merged main | grep -q "$$branch"; then \
			echo "Removing worktree: $$dir"; \
			git worktree remove "$$dir"; \
			git branch -d "$$branch" 2>/dev/null || true; \
		fi \
	done

# Prune deleted worktrees
worktree-prune:
	git worktree prune
