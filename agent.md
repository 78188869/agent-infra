# Agent Guide: Agentic Coding Platform

> **Version**: v2.0
> **Last Updated**: 2026-03-28
> **Target Audience**: Coding Agents (Claude Code, etc.)

---

## 1. Project Overview

**Agentic Coding Platform** - A universal agent task execution platform that wraps Claude Code CLI to provide a managed execution environment for coding tasks.

| Attribute | Value |
|-----------|-------|
| **Current Phase** | MVP (v1.0) |
| **Delivery Target** | 1-2 months |

**Design Principles**: Modular Monolith | Reuse First | Fast Validation

---

## 2. Tech Stack

| Layer | Technology |
|-------|------------|
| Frontend | React 18 + TypeScript 5 + Ant Design 5 + Vite 5 |
| Backend | Go 1.21 + Gin 1.9 + GORM 1.25 |
| Database | OceanBase (MySQL compatible) |
| Cache/Queue | Redis 6 |
| Container | Kubernetes (ACK) + Docker |
| Execution | Claude Code CLI |

---

## 3. Project Structure

```
agent-infra/
├── cmd/                     # 程序入口
├── internal/
│   ├── api/                 # HTTP 层：handler（参数校验）、middleware（认证/限流）、router、response
│   ├── service/             # 业务逻辑核心：编排各模块，事务边界在此
│   ├── repository/          # 数据访问层：封装数据库查询，不包含业务逻辑
│   ├── model/               # GORM 模型定义，只定义数据结构
│   ├── scheduler/           # 调度引擎：队列管理、限流、任务分发
│   ├── executor/            # 执行引擎：Job 生命周期、Pod 管理
│   ├── config/              # 配置加载与管理
│   ├── migration/           # 数据库迁移脚本
│   ├── monitoring/          # 监控指标采集
│   └── seed/                # 数据库种子数据
├── pkg/                     # 公共工具：aliyun（SLS 日志）、errors（错误码）等
├── configs/                 # 配置文件（config.yaml）
├── scripts/                 # 工具脚本（worktree、cli-runner）
├── web/                     # 前端：React + Ant Design
└── deploy/                  # 部署：K8s manifests、Dockerfiles
```

---

## 4. Architecture Constraints

**Layer Rules**: Handler → Service → Repository → Model

**Prohibited**:
- ✗ handler 直接调用 repository
- ✗ repository 调用 service
- ✗ scheduler/executor 直接操作数据库

**目录规则**：
- handler → 只做参数校验和响应格式化，调用 service 处理业务
- service → 业务逻辑核心，通过 repository 操作数据，事务边界在此
- repository → 数据访问层，封装数据库查询，不包含业务逻辑
- model → 只定义数据结构和表映射，不含业务逻辑
- scheduler/executor → 独立模块，通过 service 层交互，不直接操作数据库

---

## 5. Coding Standards

遵循主流规范（Google Go Style Guide / TypeScript Handbook / Ant Design Docs），以下是本项目规范：

### 通用规范

| Language | Key Points |
|----------|------------|
| **Go** | Use `gofmt`/`goimports`; wrap errors with `%w`; interface names: verb+er |
| **TypeScript** | Functional components + hooks; strict mode; Ant Design components |
| **Database** | Tables: snake_case plural; Models: PascalCase singular |

**Git Commits**: `<type>(<scope>): <subject>` (feat, fix, docs, style, refactor, test, chore)

### 项目补充规则

| 维度 | 规则 |
|------|------|
| 前端 UI | 统一使用 Ant Design 组件，不引入其他 UI 库 |
| 错误处理 | Go 统一用 `%w` wrap；前端统一用 Ant Design Message 提示 |
| API 响应 | 遵循 `docs/knowledge/quick-reference.md` 中的 response code 体系 |
| Context | 所有外部调用传 ctx，不传 nil |
| 数据库迁移 | 新增字段用 Nullable 或设默认值，不破坏现有数据 |

---

## 6. Documentation Structure

知识库按功能域划分、持续演进、不随版本快照，每个模块记录变更历史。

```
docs/
├── knowledge/               # 独立知识库（按功能域划分，持续演进，详见 §7 模块索引）
├── v{version}/              # 版本目录（按版本隔离）
│   ├── BRD.md / PRD.md / TRD.md
│   ├── decisions/           # ADR（含 README.md 索引和 adr-template.md）
│   ├── issues/              # Issue 摘要（含 README.md 模板）
│   └── plans/               # 执行计划（含 README.md 模板）
├── current -> v{version}/   # 软链接指向当前活跃版本
└── BRD.md                   # 业务需求文档（根级别）
```

### 阅读顺序

```
1. agent.md (本文档) → 了解项目概况与开发流程
2. docs/current/BRD.md → 了解业务背景
3. docs/current/PRD.md → 了解产品需求
4. docs/current/TRD.md → 了解技术设计
5. docs/current/decisions/ → 了解架构决策
6. docs/knowledge/{modules}.md → 深入模块细节
```

---

## 7. Issue Development Workflow

完整流程从拉取代码到清理环境共 15 步：

| Step | 操作 | 要点 |
|------|------|------|
| 0 | 拉取最新代码 | `git pull origin main` |
| 1 | 创建 Worktree | `git worktree add -b feature/issue-{N} .claude/worktrees/issue-{N} main` |
| 2 | 创建 Issue Summary | `docs/current/issues/issue-{N}-{title}.md`，模板见 `issues/README.md` |
| 3 | 获取背景知识 | 读 TRD.md → decisions/ → knowledge/{modules}.md，按下方模块选择表确定范围 |
| 3.5 | 设计与文档回流 | **（条件）** brainstorming 后将决策写入 ADR → 更新 TRD/knowledge 文档 → commit |
| 4 | 创建执行计划 | `docs/current/plans/{YYYY-MM-DD}-{title}.md`，模板见 `plans/README.md` |
| 5 | 开发实现 (TDD) | 按 Plan 逐步实现，编写测试用例，覆盖率 > 80% |
| 6 | 测试验证 | `make test && make lint && go test -cover ./internal/...` |
| 7 | 代码审查 | 验证所有测试通过，修复发现的问题 |
| 8 | 提交推送 | `git commit -m "feat(scope): description" && git push -u origin <branch>` |
| 9 | 创建 PR | `gh pr create --base main --title "feat: description"` |
| 10 | 等待合并（人工） | 等待人工审核并合并 PR |
| 11 | 拉取合并代码 | `git checkout main && git pull origin main` |
| 12 | 关闭 Issue | `gh issue close {N} --repo {repo}` + 更新 Issue Summary 状态 |
| 13 | 清理环境 | `git worktree remove .claude/worktrees/issue-{N}` |

> **计划结构要求**：Step 4 的执行计划必须覆盖 Step 5-9（实现、测试、审查、推送、PR），每个 Step 对应一个 Task；Step 0-4 和 Step 10-13 在开发前创建完整的 15 步 TaskCreate 列表跟踪进度。

### 模块知识选择

| 工作类型 | 加载的知识模块 | 相关 TRD 章节 |
|---------|---------------|--------------|
| API 开发 | core-api, database | TRD §5 API 设计 |
| 调度逻辑 | scheduler, database | TRD §3.2 调度引擎 |
| 执行管理 | executor, provider | TRD §3.2 执行引擎 |
| 能力管理 | capability, provider | TRD §3.2 能力注册 |
| 干预功能 | intervention, executor | TRD §3.2 人工干预 |
| 监控告警 | monitoring | TRD §7 监控告警 |

### 知识模块索引

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

---

## 8. Quick Lookup Tables

详见 `docs/knowledge/quick-reference.md`（API Response Codes、Task Status、Redis Keys）。

---

## 9. Commands

```bash
# Backend
make run test lint
make test-coverage        # 覆盖率报告
make db-migrate           # 数据库迁移

# Frontend
cd web && npm run dev

# Deploy
make docker-build-all k8s-apply k8s-status
```
