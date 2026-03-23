# Agent Guide: Agentic Coding Platform

> **Version**: v1.2
> **Last Updated**: 2026-03-22
> **Target Audience**: Coding Agents (Claude Code, etc.)

---

## 1. Project Overview

**Agentic Coding Platform** - A universal agent task execution platform that wraps Claude Code CLI to provide a managed execution environment for coding tasks.

| Attribute | Value |
|-----------|-------|
| **Code Location** | `/Users/yang/workspace/learning/agent-infra/` |
| **Current Phase** | MVP (v1.0) |
| **Delivery Target** | 1-2 months |

**Design Principles**: Modular Monolith | Reuse First | Fast Validation

---

## 2. Knowledge Index

| Document | Path | Purpose |
|----------|------|---------|
| **BRD** | `docs/BRD.md` | Business requirements |
| **PRD** | `docs/PRD.md` | User stories, features |
| **TRD** | `docs/plans/2026-03-22-mvp-trd.md` | Architecture, data model, API |
| **Tech Decisions** | `docs/plans/2026-03-21-mvp-technical-decisions.md` | Decision records |

**Reading Order**: This file → TRD §1-3 → PRD §4 → TRD §6/7 (for DB/API work)

---

## 3. Tech Stack

| Layer | Technology |
|-------|------------|
| Frontend | React 18 + TypeScript 5 + Ant Design 5 + Vite 5 |
| Backend | Go 1.22 + Gin 1.9 + GORM 1.25 |
| Database | OceanBase (MySQL compatible) |
| Cache/Queue | Redis 6 |
| Container | Kubernetes (ACK) + Docker |
| Execution | Claude Code CLI |

---

## 4. Project Structure

```
agent-infra/
├── cmd/control-plane/       # Service entry
├── internal/
│   ├── api/handler/         # HTTP handlers
│   ├── api/middleware/      # Auth, rate limit, logging
│   ├── service/             # Business logic
│   ├── scheduler/           # Task scheduling
│   ├── executor/            # Job management
│   └── model/               # Data models
├── pkg/                     # Shared utilities
├── web/                     # Frontend (React)
└── deploy/k8s/              # K8s manifests
```

---

## 5. Coding Standards

> **Follow external standards. See §10 for all reference links.**

| Language | Key Points |
|----------|------------|
| **Go** | Use `gofmt`/`goimports`; wrap errors with `%w`; interface names: verb+er |
| **TypeScript** | Functional components + hooks; strict mode; Ant Design components |
| **Database** | Tables: snake_case plural; Models: PascalCase singular |

**Git Commits**: `<type>(<scope>): <subject>` (feat, fix, docs, style, refactor, test, chore)

---

## 6. Architecture Constraints

**Layer Rules**: Presentation → Gateway → Application → Data/Execution

**Prohibited**:
- ✗ Data layer calling Application
- ✗ Execution accessing database directly

| Module | Responsibility |
|--------|----------------|
| Handler | HTTP handling, validation |
| Service | Business rules, transactions |
| Scheduler | Queue, rate limiting |
| Executor | Job lifecycle |

---

## 7. Quick Lookup Tables

### API Response

| Code | Meaning |
|------|---------|
| 0 | Success |
| 400xx | Request errors |
| 401xx | Auth errors |
| 403xx | Forbidden |
| 404xx | Not found |
| 500xx | Server errors |

### Task Status

```
Pending → Scheduled → Running → Succeeded
                   ↓          ↓
               Paused      Failed → Retrying
```

### Redis Keys

| Pattern | Description |
|---------|-------------|
| `queue:tasks:{high\|normal\|low}` | Priority queues |
| `task:{id}:meta` | Task metadata |
| `tenant:{id}:quota:used` | Quota usage |

---

## 8. Common Tasks

### Add API Endpoint
1. Define types in `internal/api/handler/`
2. Create handler with validation
3. Add service method
4. Register route
5. Write tests

### Add Database Model
1. Define struct in `internal/model/`
2. Create migration
3. Update service

### Add Frontend Page
1. Create component in `web/src/`
2. Add API client
3. Register route

---

## 9. Commands

```bash
# Backend
make run test lint

# Frontend
cd web && npm run dev

# Deploy
make docker-build-all k8s-apply k8s-status
```

---

## 10. External References

| Category | Resource | URL |
|----------|----------|-----|
| **Go** | Google Go Style Guide | https://google.github.io/styleguide/go/ |
| **Go** | Effective Go | https://go.dev/doc/effective_go |
| **Go** | Uber Go Style Guide | https://github.com/uber-go/guide/blob/master/style.md |
| **TypeScript** | TypeScript Handbook | https://www.typescriptlang.org/docs/handbook/ |
| **React** | React Documentation | https://react.dev/learn |
| **React** | Airbnb React Style Guide | https://github.com/airbnb/javascript/tree/master/react |
| **UI** | Ant Design Docs | https://ant.design/docs/react/introduce |
| **Backend** | Gin Documentation | https://gin-gonic.com/docs/ |
| **Backend** | GORM Guide | https://gorm.io/docs/ |
| **Infra** | Kubernetes Documentation | https://kubernetes.io/docs/ |
| **Infra** | Docker Best Practices | https://docs.docker.com/develop/develop-images/dockerfile_best-practices/ |
| **Security** | OWASP API Security | https://owasp.org/www-project-api-security/ |
| **Architecture** | 12-Factor App | https://12factor.net/ |
| **Agent** | Claude Code CLI Docs | https://docs.anthropic.com/claude/docs/claude-code |
| **Agent** | MCP Specification | https://modelcontextprotocol.io/ |

---

## 11. Changelog

| Version | Changes |
|---------|---------|
| v1.2 | Removed duplicate references, further simplified |
| v1.1 | Simplified: reference external standards |
| v1.0 | Initial version |
