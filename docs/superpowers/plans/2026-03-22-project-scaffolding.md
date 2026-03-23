# Project Scaffolding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create the basic project scaffolding with hello-world level code, Makefile, Dockerfile, and Helm charts for the Agentic Coding Platform.

**Architecture:** Go backend with Gin framework, React frontend with Vite, containerized with Docker, deployed via Kubernetes Helm charts.

**Tech Stack:** Go 1.22 + Gin 1.9 + React 18 + TypeScript 5 + Ant Design 5 + Vite 5 + Docker + Kubernetes (Helm)

---

## File Structure

### Backend Files
- Create: `cmd/control-plane/main.go` - Service entry point
- Create: `cmd/control-plane/config.yaml` - Configuration file
- Create: `internal/api/handler/health.go` - Health check handler
- Create: `internal/api/middleware/logger.go` - Request logging middleware
- Create: `internal/api/router/router.go` - Route registration
- Create: `go.mod` - Go module definition
- Create: `go.sum` - Go dependencies lock

### Frontend Files
- Create: `web/package.json` - NPM dependencies
- Create: `web/vite.config.ts` - Vite configuration
- Create: `web/tsconfig.json` - TypeScript configuration
- Create: `web/index.html` - HTML entry
- Create: `web/src/main.tsx` - React entry point
- Create: `web/src/App.tsx` - Root component
- Create: `web/src/pages/Home.tsx` - Home page component

### Build & Deploy Files
- Create: `Makefile` - Build automation
- Create: `deploy/dockerfiles/Dockerfile.control-plane` - Backend Dockerfile
- Create: `deploy/dockerfiles/Dockerfile.web` - Frontend Dockerfile
- Create: `deploy/k8s/helm/Chart.yaml` - Helm chart definition
- Create: `deploy/k8s/helm/values.yaml` - Helm values
- Create: `deploy/k8s/helm/templates/_helpers.tpl` - Helm helpers
- Create: `deploy/k8s/helm/templates/control-plane-deployment.yaml` - Backend deployment
- Create: `deploy/k8s/helm/templates/control-plane-service.yaml` - Backend service
- Create: `deploy/k8s/helm/templates/web-deployment.yaml` - Frontend deployment
- Create: `deploy/k8s/helm/templates/web-service.yaml` - Frontend service
- Create: `deploy/k8s/helm/templates/ingress.yaml` - Ingress configuration
- Create: `.gitignore` - Git ignore rules
- Create: `.dockerignore` - Docker ignore rules

---

## Task 1: Backend Scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/control-plane/main.go`
- Create: `internal/api/handler/health.go`
- Create: `internal/api/middleware/logger.go`
- Create: `internal/api/router/router.go`
- Create: `cmd/control-plane/config.yaml`

- [ ] **Step 1: Create go.mod file**

```go
module github.com/example/agent-infra

go 1.22

require (
	github.com/gin-gonic/gin v1.9.1
	gopkg.in/yaml.v3 v3.0.1
)
```

- [ ] **Step 2: Create health check handler**

File: `internal/api/handler/health.go`

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck returns the service health status
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "control-plane",
		"version": "0.1.0",
	})
}

// ReadyCheck returns the service readiness status
func ReadyCheck(c *gin.Context) {
	// TODO: Add actual readiness checks (DB, Redis, etc.)
	c.JSON(http.StatusOK, gin.H{
		"ready": true,
	})
}
```

- [ ] **Step 3: Create logger middleware**

File: `internal/api/middleware/logger.go`

```go
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns a gin middleware for request logging
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		// Simple structured logging
		// TODO: Replace with proper structured logger (zap/zerolog)
		gin.DefaultWriter.Write([]byte(
			"[GIN] " + time.Now().Format("2006/01/02 - 15:04:05") +
				" | " + string(rune(status)) +
				" | " + latency.String() +
				" | " + c.ClientIP() +
				" | " + method + " " + path + "\n",
		))
	}
}
```

- [ ] **Step 4: Create router**

File: `internal/api/router/router.go`

```go
package router

import (
	"github.com/example/agent-infra/internal/api/handler"
	"github.com/example/agent-infra/internal/api/middleware"
	"github.com/gin-gonic/gin"
)

// Setup initializes the gin router with all routes
func Setup() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())

	// Health check endpoints
	r.GET("/health", handler.HealthCheck)
	r.GET("/ready", handler.ReadyCheck)

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// TODO: Add actual API routes
		v1.GET("/hello", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "Hello, Agentic Coding Platform!",
			})
		})
	}

	return r
}
```

- [ ] **Step 5: Create configuration file**

File: `cmd/control-plane/config.yaml`

```yaml
server:
  port: 8080
  mode: debug  # debug, release, test

database:
  host: localhost
  port: 3306
  name: agent_infra
  user: root
  password: ""

redis:
  host: localhost
  port: 6379
  db: 0

log:
  level: info
  format: json
```

- [ ] **Step 6: Create main entry point**

File: `cmd/control-plane/main.go`

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/example/agent-infra/internal/api/router"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int    `yaml:"port"`
		Mode string `yaml:"mode"`
	} `yaml:"server"`
}

func main() {
	// Load configuration
	cfg, err := loadConfig("cmd/control-plane/config.yaml")
	if err != nil {
		log.Printf("Warning: failed to load config, using defaults: %v", err)
		cfg = &Config{}
		cfg.Server.Port = 8080
		cfg.Server.Mode = "debug"
	}

	// Set gin mode
	gin.SetMode(cfg.Server.Mode)

	// Setup router
	r := router.Setup()

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting control-plane server on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}
```

- [ ] **Step 7: Initialize Go dependencies**

Run: `cd /Users/yang/workspace/learning/agent-infra && go mod tidy`
Expected: Dependencies downloaded successfully

- [ ] **Step 8: Verify backend compiles and runs**

Run: `cd /Users/yang/workspace/learning/agent-infra && go build -o bin/control-plane ./cmd/control-plane`
Expected: Binary created at `bin/control-plane`

- [ ] **Step 9: Commit backend scaffolding**

```bash
git add go.mod go.sum cmd/ internal/
git commit -m "$(cat <<'EOF'
feat: add backend scaffolding with hello-world API

- Add control-plane service entry point
- Add health check endpoints (/health, /ready)
- Add request logging middleware
- Add YAML configuration support
- Add sample /api/v1/hello endpoint

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Frontend Scaffolding

**Files:**
- Create: `web/package.json`
- Create: `web/vite.config.ts`
- Create: `web/tsconfig.json`
- Create: `web/tsconfig.node.json`
- Create: `web/index.html`
- Create: `web/src/main.tsx`
- Create: `web/src/App.tsx`
- Create: `web/src/App.css`
- Create: `web/src/vite-env.d.ts`
- Create: `web/src/pages/Home.tsx`

- [ ] **Step 1: Create package.json**

File: `web/package.json`

```json
{
  "name": "agent-infra-web",
  "private": true,
  "version": "0.1.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "lint": "eslint . --ext ts,tsx --report-unused-disable-directives --max-warnings 0",
    "preview": "vite preview"
  },
  "dependencies": {
    "antd": "^5.15.0",
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.22.0"
  },
  "devDependencies": {
    "@types/react": "^18.2.55",
    "@types/react-dom": "^18.2.19",
    "@typescript-eslint/eslint-plugin": "^7.0.2",
    "@typescript-eslint/parser": "^7.0.2",
    "@vitejs/plugin-react": "^4.2.1",
    "eslint": "^8.56.0",
    "eslint-plugin-react-hooks": "^4.6.0",
    "eslint-plugin-react-refresh": "^0.4.5",
    "typescript": "^5.3.3",
    "vite": "^5.1.4"
  }
}
```

- [ ] **Step 2: Create vite.config.ts**

File: `web/vite.config.ts`

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
})
```

- [ ] **Step 3: Create tsconfig.json**

File: `web/tsconfig.json`

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,

    /* Bundler mode */
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",

    /* Linting */
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,

    /* Paths */
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  },
  "include": ["src"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

- [ ] **Step 4: Create tsconfig.node.json**

File: `web/tsconfig.node.json`

```json
{
  "compilerOptions": {
    "composite": true,
    "skipLibCheck": true,
    "module": "ESNext",
    "moduleResolution": "bundler",
    "allowSyntheticDefaultImports": true,
    "strict": true
  },
  "include": ["vite.config.ts"]
}
```

- [ ] **Step 5: Create index.html**

File: `web/index.html`

```html
<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Agentic Coding Platform</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 6: Create vite-env.d.ts**

File: `web/src/vite-env.d.ts`

```typescript
/// <reference types="vite/client" />
```

- [ ] **Step 7: Create main.tsx entry point**

File: `web/src/main.tsx`

```tsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import App from './App'
import './App.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ConfigProvider locale={zhCN}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </ConfigProvider>
  </React.StrictMode>,
)
```

- [ ] **Step 8: Create App.css**

File: `web/src/App.css`

```css
#root {
  min-height: 100vh;
}

.app-layout {
  min-height: 100vh;
}

.app-content {
  padding: 24px;
  background: #f0f2f5;
  min-height: calc(100vh - 64px);
}
```

- [ ] **Step 9: Create App.tsx root component**

File: `web/src/App.tsx`

```tsx
import { Routes, Route } from 'react-router-dom'
import { Layout } from 'antd'
import Home from './pages/Home'

const { Header, Content } = Layout

function App() {
  return (
    <Layout className="app-layout">
      <Header style={{ display: 'flex', alignItems: 'center', background: '#001529' }}>
        <div style={{ color: '#fff', fontSize: '18px', fontWeight: 'bold' }}>
          Agentic Coding Platform
        </div>
      </Header>
      <Content className="app-content">
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/user/*" element={<Home />} />
          <Route path="/admin/*" element={<Home />} />
        </Routes>
      </Content>
    </Layout>
  )
}

export default App
```

- [ ] **Step 10: Create Home page component**

File: `web/src/pages/Home.tsx`

```tsx
import { Card, Typography, Space } from 'antd'

const { Title, Paragraph } = Typography

function Home() {
  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Card>
        <Title level={2}>Welcome to Agentic Coding Platform</Title>
        <Paragraph>
          A universal agent task execution platform that wraps Claude Code CLI
          to provide a managed execution environment for coding tasks.
        </Paragraph>
      </Card>

      <Card title="Quick Start">
        <Paragraph>
          This is a hello-world page. The platform will support:
        </Paragraph>
        <ul>
          <li>Task template management</li>
          <li>Task execution and monitoring</li>
          <li>Sandbox environment management</li>
          <li>Human intervention mechanisms</li>
        </ul>
      </Card>
    </Space>
  )
}

export default Home
```

- [ ] **Step 11: Install frontend dependencies**

Run: `cd /Users/yang/workspace/learning/agent-infra/web && npm install`
Expected: Dependencies installed successfully

- [ ] **Step 12: Verify frontend builds**

Run: `cd /Users/yang/workspace/learning/agent-infra/web && npm run build`
Expected: Build succeeds, output in `web/dist/`

- [ ] **Step 13: Commit frontend scaffolding**

```bash
git add web/
git commit -m "$(cat <<'EOF'
feat: add frontend scaffolding with React + Ant Design

- Add Vite + TypeScript configuration
- Add React 18 with Ant Design 5 setup
- Add basic layout with header and routing
- Add hello-world Home page
- Configure API proxy to backend

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Makefile

**Files:**
- Create: `Makefile`
- Create: `.gitignore`
- Create: `.dockerignore`

- [ ] **Step 1: Create .gitignore**

File: `.gitignore`

```gitignore
# Binaries
bin/
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary
*.test

# Output of go coverage
*.out
coverage.html

# Go workspace
go.work

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Frontend
web/node_modules/
web/dist/
web/.vite/

# Environment
.env
.env.local
.env.*.local
*.env

# Logs
logs/
*.log

# Temp files
tmp/
temp/
```

- [ ] **Step 2: Create .dockerignore**

File: `.dockerignore`

```dockerignore
# Git
.git
.gitignore

# Documentation
docs/
*.md
!README.md

# IDE
.idea/
.vscode/

# Binaries
bin/

# Frontend build artifacts
web/node_modules/
web/dist/

# Git
.git/

# Docker
Dockerfile*
docker-compose*

# K8s
deploy/k8s/

# Test files
*_test.go
**/*_test.go

# Coverage
*.out
coverage.*

# OS
.DS_Store

# Environment files
.env
.env.*
```

- [ ] **Step 3: Create Makefile**

File: `Makefile`

```makefile
# Makefile for Agentic Coding Platform
# =====================================

# Variables
APP_NAME := agent-infra
VERSION := 0.1.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Directories
BIN_DIR := bin
CMD_DIR := cmd/control-plane
WEB_DIR := web

# Binary
BINARY_NAME := control-plane
BINARY := $(BIN_DIR)/$(BINARY_NAME)

# Docker
DOCKER_REGISTRY ?= registry.example.com
DOCKER_TAG := $(DOCKER_REGISTRY)/$(APP_NAME):$(VERSION)-$(GIT_COMMIT)
DOCKER_TAG_LATEST := $(DOCKER_REGISTRY)/$(APP_NAME):latest

# Helm
HELM_RELEASE := agent-infra
HELM_NAMESPACE := control-plane

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m

.PHONY: all help build run test clean lint fmt vet deps

# Default target
all: deps build

# Help target
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development:"
	@echo "  build          Build the backend binary"
	@echo "  run            Run the backend server"
	@echo "  test           Run all tests"
	@echo "  lint           Run linters"
	@echo "  fmt            Format code"
	@echo "  vet            Run go vet"
	@echo "  clean          Clean build artifacts"
	@echo ""
	@echo "Frontend:"
	@echo "  web-deps       Install frontend dependencies"
	@echo "  web-build      Build frontend"
	@echo "  web-dev        Start frontend dev server"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build   Build Docker images"
	@echo "  docker-push    Push Docker images to registry"
	@echo ""
	@echo "Kubernetes:"
	@echo "  helm-install   Install Helm chart"
	@echo "  helm-upgrade   Upgrade Helm release"
	@echo "  helm-uninstall Uninstall Helm release"
	@echo "  k8s-apply      Apply K8s manifests (alternative to Helm)"
	@echo "  k8s-status     Show K8s deployment status"

# =====================================
# Development targets
# =====================================

deps:
	@echo "$(YELLOW)Downloading Go dependencies...$(NC)"
	$(GOMOD) download
	$(GOMOD) tidy

build: deps
	@echo "$(YELLOW)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) \
		-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)" \
		-o $(BINARY) ./$(CMD_DIR)
	@echo "$(GREEN)Build complete: $(BINARY)$(NC)"

run: build
	@echo "$(YELLOW)Starting $(BINARY_NAME) server...$(NC)"
	./$(BINARY)

test:
	@echo "$(YELLOW)Running tests...$(NC)"
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "$(GREEN)Tests complete$(NC)"

lint:
	@echo "$(YELLOW)Running linters...$(NC)"
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.56.2)
	golangci-lint run ./...

fmt:
	@echo "$(YELLOW)Formatting code...$(NC)"
	$(GOCMD) fmt ./...

vet:
	@echo "$(YELLOW)Running go vet...$(NC)"
	$(GOCMD) vet ./...

clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@rm -rf $(BIN_DIR)
	$(GOCLEAN)

# =====================================
# Frontend targets
# =====================================

web-deps:
	@echo "$(YELLOW)Installing frontend dependencies...$(NC)"
	cd $(WEB_DIR) && npm install

web-build: web-deps
	@echo "$(YELLOW)Building frontend...$(NC)"
	cd $(WEB_DIR) && npm run build

web-dev:
	@echo "$(YELLOW)Starting frontend dev server...$(NC)"
	cd $(WEB_DIR) && npm run dev

# =====================================
# Docker targets
# =====================================

docker-build: build web-build
	@echo "$(YELLOW)Building Docker images...$(NC)"
	docker build -f deploy/dockerfiles/Dockerfile.control-plane -t $(DOCKER_TAG) -t $(DOCKER_TAG_LATEST) .
	docker build -f deploy/dockerfiles/Dockerfile.web -t $(DOCKER_REGISTRY)/$(APP_NAME)-web:$(VERSION)-$(GIT_COMMIT) .
	@echo "$(GREEN)Docker images built$(NC)"

docker-push:
	@echo "$(YELLOW)Pushing Docker images...$(NC)"
	docker push $(DOCKER_TAG)
	docker push $(DOCKER_TAG_LATEST)
	docker push $(DOCKER_REGISTRY)/$(APP_NAME)-web:$(VERSION)-$(GIT_COMMIT)
	@echo "$(GREEN)Docker images pushed$(NC)"

# =====================================
# Kubernetes targets
# =====================================

helm-install:
	@echo "$(YELLOW)Installing Helm chart...$(NC)"
	helm install $(HELM_RELEASE) deploy/k8s/helm \
		--namespace $(HELM_NAMESPACE) \
		--create-namespace \
		--set image.repository=$(DOCKER_REGISTRY)/$(APP_NAME) \
		--set image.tag=$(VERSION)-$(GIT_COMMIT)

helm-upgrade:
	@echo "$(YELLOW)Upgrading Helm release...$(NC)"
	helm upgrade $(HELM_RELEASE) deploy/k8s/helm \
		--namespace $(HELM_NAMESPACE) \
		--set image.repository=$(DOCKER_REGISTRY)/$(APP_NAME) \
		--set image.tag=$(VERSION)-$(GIT_COMMIT)

helm-uninstall:
	@echo "$(YELLOW)Uninstalling Helm release...$(NC)"
	helm uninstall $(HELM_RELEASE) --namespace $(HELM_NAMESPACE)

k8s-apply:
	@echo "$(YELLOW)Applying K8s manifests...$(NC)"
	kubectl apply -f deploy/k8s/manifests/ --namespace $(HELM_NAMESPACE)

k8s-status:
	@echo "$(YELLOW)K8s deployment status:$(NC)"
	@kubectl get pods -n $(HELM_NAMESPACE)
	@kubectl get services -n $(HELM_NAMESPACE)
	@kubectl get ingress -n $(HELM_NAMESPACE)
```

- [ ] **Step 4: Verify Makefile works**

Run: `cd /Users/yang/workspace/learning/agent-infra && make build`
Expected: Binary built successfully

- [ ] **Step 5: Commit Makefile and ignore files**

```bash
git add Makefile .gitignore .dockerignore
git commit -m "$(cat <<'EOF'
feat: add Makefile and ignore files

- Add comprehensive Makefile with build, test, docker, helm targets
- Add .gitignore for Go, Node, IDE artifacts
- Add .dockerignore for efficient Docker builds

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Dockerfiles

**Files:**
- Create: `deploy/dockerfiles/Dockerfile.control-plane`
- Create: `deploy/dockerfiles/Dockerfile.web`

- [ ] **Step 1: Create backend Dockerfile**

File: `deploy/dockerfiles/Dockerfile.control-plane`

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/

# Build the binary
ARG VERSION=0.1.0
ARG GIT_COMMIT=unknown
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT}" \
    -o /app/bin/control-plane ./cmd/control-plane

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/control-plane /app/
COPY --from=builder /app/cmd/control-plane/config.yaml /app/config.yaml

# Set ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
ENTRYPOINT ["/app/control-plane"]
```

- [ ] **Step 2: Create frontend Dockerfile**

File: `deploy/dockerfiles/Dockerfile.web`

```dockerfile
# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files first for better caching
COPY web/package.json web/package-lock.json* ./

# Install dependencies
RUN npm ci

# Copy source code
COPY web/ .

# Build the application
RUN npm run build

# Runtime stage with nginx
FROM nginx:alpine

# Copy custom nginx config
COPY deploy/dockerfiles/nginx.conf /etc/nginx/conf.d/default.conf

# Copy built assets from builder
COPY --from=builder /app/dist /usr/share/nginx/html

# Expose port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:80 || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
```

- [ ] **Step 3: Create nginx config for frontend**

File: `deploy/dockerfiles/nginx.conf`

```nginx
server {
    listen 80;
    server_name localhost;
    root /usr/share/nginx/html;
    index index.html;

    # Gzip compression
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;

    # Cache static assets
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # API proxy to backend
    location /api/ {
        proxy_pass http://control-plane-service:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }

    # Health check endpoint
    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }

    # SPA fallback - serve index.html for all routes
    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

- [ ] **Step 4: Commit Dockerfiles**

```bash
git add deploy/dockerfiles/
git commit -m "$(cat <<'EOF'
feat: add Dockerfiles for backend and frontend

- Add multi-stage Dockerfile for Go backend
- Add multi-stage Dockerfile for React frontend with nginx
- Add nginx configuration with API proxy and SPA support
- Include health checks in both images

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Helm Charts

**Files:**
- Create: `deploy/k8s/helm/Chart.yaml`
- Create: `deploy/k8s/helm/values.yaml`
- Create: `deploy/k8s/helm/templates/_helpers.tpl`
- Create: `deploy/k8s/helm/templates/control-plane-deployment.yaml`
- Create: `deploy/k8s/helm/templates/control-plane-service.yaml`
- Create: `deploy/k8s/helm/templates/web-deployment.yaml`
- Create: `deploy/k8s/helm/templates/web-service.yaml`
- Create: `deploy/k8s/helm/templates/ingress.yaml`
- Create: `deploy/k8s/helm/templates/configmap.yaml`
- Create: `deploy/k8s/helm/templates/namespace.yaml`

- [ ] **Step 1: Create Helm Chart.yaml**

File: `deploy/k8s/helm/Chart.yaml`

```yaml
apiVersion: v2
name: agent-infra
description: A Helm chart for Agentic Coding Platform

type: application

version: 0.1.0

appVersion: "0.1.0"

maintainers:
  - name: Platform Team
    email: platform@example.com

keywords:
  - agent
  - coding
  - automation
  - claude

home: https://github.com/example/agent-infra

sources:
  - https://github.com/example/agent-infra
```

- [ ] **Step 2: Create Helm values.yaml**

File: `deploy/k8s/helm/values.yaml`

```yaml
# Default values for agent-infra.

# Global settings
global:
  namespace: control-plane

# Control Plane (Backend) Configuration
controlPlane:
  replicaCount: 2

  image:
    repository: registry.example.com/agent-infra
    pullPolicy: IfNotPresent
    tag: "0.1.0"

  service:
    type: ClusterIP
    port: 8080
    targetPort: 8080

  resources:
    limits:
      cpu: 1000m
      memory: 1Gi
    requests:
      cpu: 100m
      memory: 256Mi

  # Environment variables
  env:
    - name: GIN_MODE
      value: "release"
    - name: LOG_LEVEL
      value: "info"

  # Configuration from values
  config:
    server:
      port: 8080
      mode: release
    database:
      host: ""
      port: 3306
      name: agent_infra
      user: ""
      password: ""
    redis:
      host: ""
      port: 6379
      db: 0

  # Liveness and readiness probes
  livenessProbe:
    httpGet:
      path: /health
      port: 8080
    initialDelaySeconds: 10
    periodSeconds: 30
    timeoutSeconds: 5
    failureThreshold: 3

  readinessProbe:
    httpGet:
      path: /ready
      port: 8080
    initialDelaySeconds: 5
    periodSeconds: 10
    timeoutSeconds: 3
    failureThreshold: 3

# Web (Frontend) Configuration
web:
  replicaCount: 2

  image:
    repository: registry.example.com/agent-infra-web
    pullPolicy: IfNotPresent
    tag: "0.1.0"

  service:
    type: ClusterIP
    port: 80
    targetPort: 80

  resources:
    limits:
      cpu: 500m
      memory: 512Mi
    requests:
      cpu: 50m
      memory: 128Mi

  livenessProbe:
    httpGet:
      path: /health
      port: 80
    initialDelaySeconds: 5
    periodSeconds: 30
    timeoutSeconds: 3
    failureThreshold: 3

  readinessProbe:
    httpGet:
      path: /health
      port: 80
    initialDelaySeconds: 3
    periodSeconds: 10
    timeoutSeconds: 2
    failureThreshold: 3

# Ingress Configuration
ingress:
  enabled: true
  className: "nginx"
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
  hosts:
    - host: agent-infra.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: agent-infra-tls
      hosts:
        - agent-infra.example.com

# Image pull secrets for private registry
imagePullSecrets: []

# Service account
serviceAccount:
  create: true
  annotations: {}
  name: ""

# Pod security context
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000

# Container security context
securityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
  readOnlyRootFilesystem: true
  runAsNonRoot: true
  runAsUser: 1000

# Node selector
nodeSelector: {}

# Tolerations
tolerations: []

# Affinity
affinity: {}
```

- [ ] **Step 3: Create Helm helpers**

File: `deploy/k8s/helm/templates/_helpers.tpl`

```tpl
{{/*
Expand the name of the chart.
*/}}
{{- define "agent-infra.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "agent-infra.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "agent-infra.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "agent-infra.labels" -}}
helm.sh/chart: {{ include "agent-infra.chart" . }}
{{ include "agent-infra.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "agent-infra.selectorLabels" -}}
app.kubernetes.io/name: {{ include "agent-infra.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "agent-infra.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "agent-infra.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Control plane labels
*/}}
{{- define "agent-infra.controlPlane.labels" -}}
{{ include "agent-infra.labels" . }}
app.kubernetes.io/component: control-plane
{{- end }}

{{/*
Control plane selector labels
*/}}
{{- define "agent-infra.controlPlane.selectorLabels" -}}
{{ include "agent-infra.selectorLabels" . }}
app.kubernetes.io/component: control-plane
{{- end }}

{{/*
Web labels
*/}}
{{- define "agent-infra.web.labels" -}}
{{ include "agent-infra.labels" . }}
app.kubernetes.io/component: web
{{- end }}

{{/*
Web selector labels
*/}}
{{- define "agent-infra.web.selectorLabels" -}}
{{ include "agent-infra.selectorLabels" . }}
app.kubernetes.io/component: web
{{- end }}
```

- [ ] **Step 4: Create namespace template**

File: `deploy/k8s/helm/templates/namespace.yaml`

```yaml
{{- if .Values.global.namespace }}
apiVersion: v1
kind: Namespace
metadata:
  name: {{ .Values.global.namespace }}
  labels:
    {{- include "agent-infra.labels" . | nindent 4 }}
    name: {{ .Values.global.namespace }}
{{- end }}
```

- [ ] **Step 5: Create ConfigMap template**

File: `deploy/k8s/helm/templates/configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "agent-infra.fullname" . }}-config
  namespace: {{ .Values.global.namespace }}
  labels:
    {{- include "agent-infra.controlPlane.labels" . | nindent 4 }}
data:
  config.yaml: |
    server:
      port: {{ .Values.controlPlane.config.server.port }}
      mode: {{ .Values.controlPlane.config.server.mode }}
    database:
      host: {{ .Values.controlPlane.config.database.host }}
      port: {{ .Values.controlPlane.config.database.port }}
      name: {{ .Values.controlPlane.config.database.name }}
    redis:
      host: {{ .Values.controlPlane.config.redis.host }}
      port: {{ .Values.controlPlane.config.redis.port }}
      db: {{ .Values.controlPlane.config.redis.db }}
    log:
      level: info
      format: json
```

- [ ] **Step 6: Create control-plane deployment**

File: `deploy/k8s/helm/templates/control-plane-deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "agent-infra.fullname" . }}-control-plane
  namespace: {{ .Values.global.namespace }}
  labels:
    {{- include "agent-infra.controlPlane.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.controlPlane.replicaCount }}
  selector:
    matchLabels:
      {{- include "agent-infra.controlPlane.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "agent-infra.controlPlane.selectorLabels" . | nindent 8 }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "agent-infra.serviceAccountName" . }}
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      containers:
        - name: control-plane
          securityContext:
            {{- toYaml .Values.securityContext | nindent 12 }}
          image: "{{ .Values.controlPlane.image.repository }}:{{ .Values.controlPlane.image.tag | default .Chart.AppVersion }}"
          imagePullPolicy: {{ .Values.controlPlane.image.pullPolicy }}
          ports:
            - name: http
              containerPort: {{ .Values.controlPlane.service.targetPort }}
              protocol: TCP
          livenessProbe:
            {{- toYaml .Values.controlPlane.livenessProbe | nindent 12 }}
          readinessProbe:
            {{- toYaml .Values.controlPlane.readinessProbe | nindent 12 }}
          resources:
            {{- toYaml .Values.controlPlane.resources | nindent 12 }}
          env:
            {{- toYaml .Values.controlPlane.env | nindent 12 }}
          volumeMounts:
            - name: config
              mountPath: /app/config.yaml
              subPath: config.yaml
              readOnly: true
      volumes:
        - name: config
          configMap:
            name: {{ include "agent-infra.fullname" . }}-config
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
```

- [ ] **Step 7: Create control-plane service**

File: `deploy/k8s/helm/templates/control-plane-service.yaml`

```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "agent-infra.fullname" . }}-control-plane
  namespace: {{ .Values.global.namespace }}
  labels:
    {{- include "agent-infra.controlPlane.labels" . | nindent 4 }}
spec:
  type: {{ .Values.controlPlane.service.type }}
  ports:
    - port: {{ .Values.controlPlane.service.port }}
      targetPort: http
      protocol: TCP
      name: http
  selector:
    {{- include "agent-infra.controlPlane.selectorLabels" . | nindent 4 }}
```

- [ ] **Step 8: Create web deployment**

File: `deploy/k8s/helm/templates/web-deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "agent-infra.fullname" . }}-web
  namespace: {{ .Values.global.namespace }}
  labels:
    {{- include "agent-infra.web.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.web.replicaCount }}
  selector:
    matchLabels:
      {{- include "agent-infra.web.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "agent-infra.web.selectorLabels" . | nindent 8 }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "agent-infra.serviceAccountName" . }}
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      containers:
        - name: web
          securityContext:
            {{- toYaml .Values.securityContext | nindent 12 }}
          image: "{{ .Values.web.image.repository }}:{{ .Values.web.image.tag | default .Chart.AppVersion }}"
          imagePullPolicy: {{ .Values.web.image.pullPolicy }}
          ports:
            - name: http
              containerPort: {{ .Values.web.service.targetPort }}
              protocol: TCP
          livenessProbe:
            {{- toYaml .Values.web.livenessProbe | nindent 12 }}
          readinessProbe:
            {{- toYaml .Values.web.readinessProbe | nindent 12 }}
          resources:
            {{- toYaml .Values.web.resources | nindent 12 }}
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
```

- [ ] **Step 9: Create web service**

File: `deploy/k8s/helm/templates/web-service.yaml`

```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "agent-infra.fullname" . }}-web
  namespace: {{ .Values.global.namespace }}
  labels:
    {{- include "agent-infra.web.labels" . | nindent 4 }}
spec:
  type: {{ .Values.web.service.type }}
  ports:
    - port: {{ .Values.web.service.port }}
      targetPort: http
      protocol: TCP
      name: http
  selector:
    {{- include "agent-infra.web.selectorLabels" . | nindent 4 }}
```

- [ ] **Step 10: Create ingress**

File: `deploy/k8s/helm/templates/ingress.yaml`

```yaml
{{- if .Values.ingress.enabled -}}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "agent-infra.fullname" . }}
  namespace: {{ .Values.global.namespace }}
  labels:
    {{- include "agent-infra.labels" . | nindent 4 }}
  {{- with .Values.ingress.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
spec:
  {{- if .Values.ingress.className }}
  ingressClassName: {{ .Values.ingress.className }}
  {{- end }}
  {{- if .Values.ingress.tls }}
  tls:
    {{- range .Values.ingress.tls }}
    - hosts:
        {{- range .hosts }}
        - {{ . | quote }}
        {{- end }}
      secretName: {{ .secretName }}
    {{- end }}
  {{- end }}
  rules:
    {{- range .Values.ingress.hosts }}
    - host: {{ .host | quote }}
      http:
        paths:
          {{- range .paths }}
          - path: {{ .path }}
            pathType: {{ .pathType }}
            backend:
              service:
                name: {{ include "agent-infra.fullname" $ }}-web
                port:
                  number: {{ $.Values.web.service.port }}
          {{- end }}
    {{- end }}
{{- end }}
```

- [ ] **Step 11: Verify Helm chart syntax**

Run: `helm lint /Users/yang/workspace/learning/agent-infra/deploy/k8s/helm`
Expected: `lint passes`

- [ ] **Step 12: Template Helm chart to verify rendering**

Run: `helm template test /Users/yang/workspace/learning/agent-infra/deploy/k8s/helm --debug`
Expected: Valid YAML output with all resources

- [ ] **Step 13: Commit Helm charts**

```bash
git add deploy/k8s/helm/
git commit -m "$(cat <<'EOF'
feat: add Helm charts for Kubernetes deployment

- Add Chart.yaml with metadata
- Add values.yaml with configurable parameters
- Add control-plane deployment and service
- Add web (frontend) deployment and service
- Add ingress with TLS support
- Add ConfigMap for application configuration
- Add namespace template
- Include health checks and resource limits

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Final Verification

- [ ] **Step 1: Run full build**

Run: `cd /Users/yang/workspace/learning/agent-infra && make build`
Expected: Backend binary created

- [ ] **Step 2: Run backend server**

Run: `cd /Users/yang/workspace/learning/agent-infra && make run &`
Expected: Server starts on port 8080

- [ ] **Step 3: Test health endpoint**

Run: `curl http://localhost:8080/health`
Expected: `{"status":"healthy","service":"control-plane","version":"0.1.0"}`

- [ ] **Step 4: Test API endpoint**

Run: `curl http://localhost:8080/api/v1/hello`
Expected: `{"message":"Hello, Agentic Coding Platform!"}`

- [ ] **Step 5: Build frontend**

Run: `cd /Users/yang/workspace/learning/agent-infra && make web-build`
Expected: Frontend built successfully

- [ ] **Step 6: Verify Docker builds (optional)**

Run: `cd /Users/yang/workspace/learning/agent-infra && docker build -f deploy/dockerfiles/Dockerfile.control-plane -t agent-infra:test .`
Expected: Docker image built successfully

- [ ] **Step 7: Final status check**

```bash
echo "=== Project Structure ==="
tree -L 3 /Users/yang/workspace/learning/agent-infra --dirsfirst -I 'node_modules|bin|dist'

echo "=== Go Build ==="
ls -la /Users/yang/workspace/learning/agent-infra/bin/

echo "=== Helm Lint ==="
helm lint /Users/yang/workspace/learning/agent-infra/deploy/k8s/helm
```

---

## Summary

This plan creates a complete project scaffolding for the Agentic Coding Platform with:

1. **Backend**: Go + Gin with health check endpoints and hello-world API
2. **Frontend**: React + TypeScript + Ant Design with basic layout
3. **Build**: Makefile with comprehensive targets
4. **Docker**: Multi-stage Dockerfiles for both services
5. **Kubernetes**: Helm charts for production deployment

All components follow the architecture specified in the TRD and can be deployed independently.
