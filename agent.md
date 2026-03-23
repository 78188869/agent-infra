# Agent Guide: Agentic Coding Platform

> **Version**: v1.4
> **Last Updated**: 2026-03-23
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

## 2. Directory Structure

```
docs/
├── knowledge/                 # 独立知识库（持续演进）
│   ├── core-api.md           # 任务/模板管理 API
│   ├── database.md           # 数据模型与存储
│   ├── scheduler.md          # 任务调度引擎
│   ├── executor.md           # 沙箱执行引擎
│   ├── provider.md           # Agent 运行时配置
│   ├── capability.md         # 能力注册管理
│   ├── intervention.md       # 人工干预机制
│   └── monitoring.md         # 监控告警设计
│
├── v{version}/               # 版本目录（按版本隔离）
│   ├── BRD.md                # 业务需求文档
│   ├── PRD.md                # 产品需求文档
│   ├── TRD.md                # 技术设计文档
│   ├── decisions/            # 架构决策记录（ADR）
│   │   ├── README.md         # ADR 索引和指南
│   │   └── adr-template.md   # ADR 模板
│   ├── issues/               # Issue 摘要
│   │   └── README.md         # Issue 管理指南
│   └── plans/                # 执行计划
│       └── README.md         # 计划管理指南
│
├── current -> v{version}/    # 软链接指向当前活跃版本
│
└── BRD.md                    # 业务需求文档（根级别）
```

---

## 3. Issue Development Workflow

### 3.1 标准工作流程

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Issue Development Workflow                         │
└─────────────────────────────────────────────────────────────────────┘

Step 1: 创建 Issue Summary
    │
    ├── 在 docs/current/issues/ 创建 issue-{number}-{title}.md
    ├── 使用 issues/README.md 中的模板
    └── 填写 Summary、Impact、Related 等字段
    │
    ▼
Step 2: 获取背景知识
    │
    ├── 读取 docs/current/TRD.md → 了解架构设计
    ├── 读取 docs/current/decisions/ → 了解相关架构决策
    ├── 读取 docs/knowledge/{modules}.md → 获取模块知识
    └── 根据模块选择表确定需要读取的知识
    │
    ▼
Step 3: 生成执行计划
    │
    ├── 在 docs/current/plans/ 创建 {YYYY-MM-DD}-{title}.md
    ├── 使用 plans/README.md 中的模板
    └── 包含 Context、Objectives、Tasks、Dependencies
    │
    ▼
Step 4: 执行开发
    │
    ├── 按照 Plan 中的 Tasks 逐项执行
    ├── 更新 knowledge 模块的 Change History
    └── 如有架构决策，创建新的 ADR
    │
    ▼
Step 5: 完成并更新
    │
    ├── 更新 Issue 状态为 Resolved
    ├── 更新 Plan 状态为 Completed
    └── 记录 Resolution 和关键变更
```

### 3.2 模块知识选择

| 工作类型 | 加载的知识模块 | 相关 TRD 章节 |
|---------|---------------|--------------|
| API 开发 | core-api, database | TRD §5 API 设计 |
| 调度逻辑 | scheduler, database | TRD §3.2 调度引擎 |
| 执行管理 | executor, provider | TRD §3.2 执行引擎 |
| 能力管理 | capability, provider | TRD §3.2 能力注册 |
| 干预功能 | intervention, executor | TRD §3.2 人工干预 |
| 监控告警 | monitoring | TRD §7 监控告警 |

### 3.3 Issue Summary 模板

创建 `docs/current/issues/issue-{number}-{title}.md`:

```markdown
# Issue #{number}: {Title}

## Summary
<!-- Issue 摘要：简述问题或需求 -->

## Impact
<!-- 影响范围：涉及的模块、用户、系统 -->

## Status
{Open | In Progress | Resolved | Closed}

## Related
- PRD: [链接到相关用户故事]
- TRD: [链接到相关技术设计]
- ADR: [链接到相关架构决策]
- Knowledge: [需要加载的知识模块]

## Resolution
<!-- 解决方案（完成后填写） -->

## Change History
| 日期 | 变更内容 |
|------|---------|
| YYYY-MM-DD | 创建 Issue |
```

### 3.4 Plan 模板

创建 `docs/current/plans/{YYYY-MM-DD}-{title}.md`:

```markdown
# Plan: {Title}

## Context
<!-- 计划背景：Issue 摘要、相关知识背景 -->

## Objectives
1. {目标1}
2. {目标2}

## Knowledge Required
<!-- 需要预读的知识 -->
- [ ] docs/knowledge/{module1}.md
- [ ] docs/current/decisions/adr-{number}.md

## Tasks

### Phase 1: {Phase Name}
- [ ] {Task 1}
- [ ] {Task 2}

### Phase 2: {Phase Name}
- [ ] {Task 1}

## Dependencies
<!-- 依赖项：其他 Issue、外部资源 -->

## Risks
<!-- 风险项：技术风险、时间风险 -->

## Status
{Not Started | In Progress | Completed | Blocked}
```

---

## 4. Knowledge Usage Guide

### 4.1 知识库特点

| 特点 | 说明 |
|------|------|
| **持续演进** | knowledge/ 不随版本快照，始终更新 |
| **模块化** | 按功能域划分，按需加载 |
| **Change History** | 每个模块记录变更历史 |
| **双向链接** | 与 TRD、ADR 相互引用 |

### 4.2 知识模块索引

| Module | Purpose | Key Sections |
|--------|---------|--------------|
| `core-api` | 任务/模板管理 API | Endpoints, Data Models |
| `database` | 数据模型与存储 | Schema, Migrations |
| `scheduler` | 任务调度引擎 | Queue, Rate Limiting |
| `executor` | 沙箱执行引擎 | Pod Lifecycle, Resource Limits |
| `provider` | Agent 运行时配置 | Claude Code Config, MCP |
| `capability` | 能力注册管理 | Tool Registration |
| `intervention` | 人工干预机制 | Checkpoints, Approvals |
| `monitoring` | 监控告警设计 | Metrics, Alerts |

### 4.3 阅读顺序

```
0. readme.md → 项目入口，快速了解项目
1. agent.md (本文档) → 了解项目概况与开发流程
2. docs/current/BRD.md → 了解业务背景
3. docs/current/PRD.md → 了解产品需求
4. docs/current/TRD.md → 了解技术设计
5. docs/current/decisions/ → 了解架构决策
6. docs/knowledge/{modules}.md → 深入模块细节
```

---

## 5. Tech Stack

| Layer | Technology |
|-------|------------|
| Frontend | React 18 + TypeScript 5 + Ant Design 5 + Vite 5 |
| Backend | Go 1.22 + Gin 1.9 + GORM 1.25 |
| Database | OceanBase (MySQL compatible) |
| Cache/Queue | Redis 6 |
| Container | Kubernetes (ACK) + Docker |
| Execution | Claude Code CLI |

---

## 6. Project Structure

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

## 7. Coding Standards

> **Follow external standards. See §11 for all reference links.**

| Language | Key Points |
|----------|------------|
| **Go** | Use `gofmt`/`goimports`; wrap errors with `%w`; interface names: verb+er |
| **TypeScript** | Functional components + hooks; strict mode; Ant Design components |
| **Database** | Tables: snake_case plural; Models: PascalCase singular |

**Git Commits**: `<type>(<scope>): <subject>` (feat, fix, docs, style, refactor, test, chore)

---

## 8. Architecture Constraints

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

## 9. Quick Lookup Tables

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

## 10. Commands

```bash
# Backend
make run test lint

# Frontend
cd web && npm run dev

# Deploy
make docker-build-all k8s-apply k8s-status
```

---

## 11. External References

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

## 12. Changelog

| Version | Changes |
|---------|---------|
| v1.4 | 添加 Issue Development Workflow；更新目录结构说明；新增 Issue/Plan 模板 |
| v1.3 | Added knowledge module index, updated doc paths to v1.0-mvp/ |
| v1.2 | Removed duplicate references, further simplified |
| v1.1 | Simplified: reference external standards |
| v1.0 | Initial version |
