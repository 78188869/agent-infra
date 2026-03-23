# MVP 技术决策记录

> 日期：2026-03-21
> 状态：已确认

---

## 1. Agent 运行时

| 决策项 | 选择 | 说明 |
|--------|------|------|
| Agent运行时 | Claude Code CLI 直接封装 | 在沙箱容器中直接运行Claude Code CLI，通过Wrapper脚本控制输入输出 |
| 配置方式 | 复用Claude Code现有配置 | 使用CLAUDE.md、.mcp.json等现有机制，不自创配置格式 |

**模板参数 → CLI参数映射**：

| 模板配置 | CLI参数 |
|---------|--------|
| spec.goal | `-p "${goal}"` |
| spec.context.initialContext | 注入到CLAUDE.md |
| execution.resources.tokenLimit | `--max-tokens` |
| execution.capabilities.tools | `--allowedTools` |
| execution.timeout | 由Wrapper控制超时 |

---

## 2. 部署环境

| 决策项 | 选择 | 说明 |
|--------|------|------|
| 容器编排 | Kubernetes (阿里云ACK) | 使用K8s原生资源管理沙箱Pod |
| 沙箱隔离 | 普通Docker容器 | MVP阶段使用标准容器隔离，预留升级接口 |
| 日志存储 | 阿里云SLS | 执行日志、审计日志存储 |

---

## 3. 技术栈

| 层级 | 技术选型 |
|------|---------|
| 后端 | Go + Gin框架 |
| 前端 | React + TypeScript + Ant Design |
| 数据库 | OceanBase |
| 缓存/队列 | Redis |
| 日志 | 阿里云SLS |
| API网关 | 阿里云MSE |
| 文件存储 | 阿里云OSS |

---

## 4. 架构设计（模块化单体）

### 4.1 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         MVP 技术架构                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  前端层 (React SPA)                                              │
│  ├── 用户面板 (/user/*)                                          │
│  └── 管理面板 (/admin/*)                                         │
│              │                                                   │
│              ▼                                                   │
│  API网关层 (阿里云MSE) - 路由转发、SSL终结                        │
│              │                                                   │
│              ▼                                                   │
│  控制面服务 (Go单体)                                              │
│  ├── API Handler层 (TaskAPI, TemplateAPI, TenantAPI, UserAPI)   │
│  ├── Service层 (TaskSvc, TemplateSvc, TenantSvc, UserSvc)       │
│  ├── 核心引擎层 (Task Scheduler, Task Executor)                 │
│  └── 基础设施层 (DB, Redis, SLS, K8s Client)                    │
│              │                                                   │
│              ▼                                                   │
│  存储层 (OceanBase + Redis + SLS + OSS)                         │
│              │                                                   │
│              ▼                                                   │
│  执行面 (K8s沙箱Pod)                                             │
│  └── Claude Code CLI + Agent Wrapper + Log Agent                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 控制面服务模块

| 模块 | 职责 |
|------|------|
| TaskAPI | 任务相关HTTP接口（创建/查询/操作） |
| TaskSvc | 任务业务逻辑、状态管理、参数校验 |
| Task Scheduler | 任务调度引擎（队列管理、限流、抢占） |
| Task Executor | 执行管理（创建/销毁沙箱Pod、状态同步） |
| TemplateSvc | 模板管理（CRUD、版本控制、校验） |
| TenantSvc | 租户管理（资源配额、配额使用统计） |
| UserSvc | 用户管理（角色权限、API Key管理） |
| CapabilitySvc | 能力管理（工具/Skills注册与授权） |

### 4.3 调度器设计

```
任务提交 → Redis优先级队列 → 限流器(租户配额/全局并发) → 调度决策(抢占/资源匹配) → Executor执行
                ↓
         排队状态查询(API)
```

**Redis数据结构**：

| Key模式 | 类型 | 说明 |
|---------|------|------|
| `queue:tasks:high` | List | 高优先级任务队列 |
| `queue:tasks:normal` | List | 普通优先级任务队列 |
| `queue:tasks:low` | List | 低优先级任务队列 |
| `task:{task_id}:meta` | Hash | 任务调度元数据 |
| `tenant:{tenant_id}:quota` | Hash | 租户实时配额使用 |
| `sandbox:{task_id}:heartbeat` | String | 沙箱心跳时间戳 |

---

## 5. 沙箱执行环境

### 5.1 沙箱Pod结构

```
沙箱容器 (Docker)
├── /workspace (代码仓库)
│   ├── CLAUDE.md (Claude Code项目配置)
│   ├── .claude/ (Claude Code配置目录)
│   └── .mcp.json (MCP工具配置，可选)
├── Claude Code CLI (直接调用)
├── Agent Wrapper (轻量封装)
│   ├── 启动CLI并传递参数
│   ├── 解析CLI流式输出
│   ├── 状态上报到控制面
│   └── 接收干预指令
└── 资源限制 (K8s Resource Limits)

Sidecar: Log Agent (日志采集到SLS)
```

### 5.2 沙箱与控制面通信

| 通信方式 | 方向 | 说明 |
|---------|------|------|
| 状态上报 | 沙箱 → 控制面 | HTTP POST，状态变更时触发 |
| 干预指令 | 控制面 → 沙箱 | HTTP POST，Wrapper暴露端口 |
| 日志采集 | 沙箱 → SLS | Log Agent自动上报 |
| 心跳检测 | 沙箱 → 控制面 | HTTP POST，每5s |

### 5.3 沙箱Pod生命周期

| 阶段 | 控制面动作 | 超时处理 |
|------|-----------|---------|
| Pending → Running | 等待Pod启动 | 5min超时标记失败 |
| Running | 监控心跳 | 心跳丢失3次标记异常 |
| Running → Succeeded | 收集结果 | - |
| Running → Failed | 记录错误、触发重试 | - |

---

## 6. 数据模型

### 6.1 核心实体

```
Tenant (租户)
    │
    ├── User (用户)
    │       │
    │       ├── Task (任务)
    │       │     ├── ExecutionLog (执行日志)
    │       │     ├── Intervention (干预记录)
    │       │     └── Output (输出产物)
    │       │
    │       └── APIKey (API密钥)
    │
    ├── Template (模板)
    │       └── TemplateVersion (版本历史)
    │
    └── Capability (能力)
            ├── Tool (工具)
            └── Skill (技能包)
```

### 6.2 核心表

| 表名 | 说明 |
|------|------|
| tenants | 租户信息、资源配额 |
| users | 用户信息、角色 |
| api_keys | API密钥管理 |
| templates | 任务模板定义 |
| tasks | 任务实例 |
| execution_logs | 执行日志索引（完整日志在SLS） |
| interventions | 人工干预记录 |
| capabilities | 能力注册（工具/Skills） |

---

## 7. API设计

### 7.1 认证流程

```
Client → MSE网关(路由转发) → Go服务
                              │
                              ▼
                         Auth Middleware
                         ├── 提取API Key
                         ├── 查询Redis缓存
                         ├── 查DB验证
                         └── 注入用户上下文
```

**认证中间件职责**：
- 从Header提取API Key (`Authorization: Bearer <api_key>`)
- 先查Redis缓存，未命中查数据库
- 验证通过后注入 user_id, tenant_id, role 到请求上下文

### 7.2 核心API

| 路径 | 说明 |
|------|------|
| `/api/v1/tasks` | 任务管理（CRUD、暂停/恢复/取消/注入） |
| `/api/v1/templates` | 模板管理 |
| `/api/v1/tenants` | 租户管理（管理面板） |
| `/api/v1/users` | 用户管理 |
| `/api/v1/capabilities` | 能力管理（管理面板） |
| `/api/v1/metrics` | 监控指标 |
| `/api/v1/internal` | 内部接口（沙箱回调） |

### 7.3 响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

---

## 8. 部署架构

### 8.1 Namespace划分

| Namespace | 资源 | 副本数 | 配置 |
|-----------|------|--------|------|
| control-plane | Deployment: control-plane | 2 | 2核4G |
| control-plane | Deployment: frontend | 2 | 1核2G |
| sandbox | Pod: sandbox-{task-id} | 动态 | 按模板配置 |

### 8.2 前端服务合并

用户面板和管理面板合并为一个服务，通过路由区分：
- `/` → 用户面板
- `/admin/*` → 管理面板
- `/api/*` → 代理到控制面服务

### 8.3 资源配额估算（MVP）

| 资源 | 配置 |
|------|------|
| 控制面Pod | 2核4G x 2副本 |
| 前端Pod | 1核2G x 2副本 |
| 沙箱Pod | 2核4G ~ 4核8G（按模板配置） |
| 最大并发沙箱 | 50个 |

---

## 9. 错误处理与测试

### 9.1 错误码规范

| 错误码范围 | 类别 |
|-----------|------|
| 0 | 成功 |
| 400xx | 请求参数错误 |
| 401xx | 认证错误 |
| 403xx | 权限错误 |
| 404xx | 资源不存在 |
| 429xx | 限流错误 |
| 500xx | 服务内部错误 |
| 503xx | 服务不可用 |

### 9.2 测试策略

| 层级 | 占比 | 工具 |
|------|------|------|
| 单元测试 | 75% | Go testing |
| 集成测试 | 20% | Go testing + testify |
| E2E测试 | 5% | 手动测试 |

---

## 10. 构建与部署

### 10.1 项目结构

```
agent-infra/
├── cmd/control-plane/          # 控制面服务入口
├── internal/                   # 内部模块
│   ├── api/                    # API Handler
│   ├── service/                # 业务逻辑
│   ├── scheduler/              # 任务调度器
│   ├── executor/               # 执行管理器
│   ├── model/                  # 数据模型
│   └── config/                 # 配置管理
├── pkg/                        # 公共库
├── web/                        # 前端代码
│   ├── user-panel/
│   └── admin-panel/
├── deploy/
│   ├── k8s/                   # K8s配置
│   └── dockerfiles/           # Dockerfile
├── scripts/
├── Makefile
└── go.mod
```

### 10.2 Dockerfile

需要3个Dockerfile：
- `Dockerfile.control-plane` - 控制面服务（多阶段构建）
- `Dockerfile.frontend` - 前端服务（Nginx托管静态资源）
- `Dockerfile.sandbox-base` - 沙箱基础镜像（含Claude Code CLI）

### 10.3 Makefile常用命令

```bash
# 本地开发
make build              # 构建服务
make run                # 本地运行
make test               # 运行测试
make lint               # 代码检查

# 前端
make web-install        # 安装依赖
make web-build          # 构建前端

# Docker
make docker-build-all   # 构建所有镜像
make docker-push        # 推送镜像

# K8s
make k8s-apply          # 部署到K8s
make k8s-status         # 查看状态
make k8s-logs           # 查看日志
```

---

## 11. 决策追溯

| 问题 | 用户选择 |
|------|---------|
| Agent运行时实现方式 | 方案1：Claude Code CLI直接封装 |
| 部署环境 | 方案1：Kubernetes (K8s) |
| 后端技术栈 | 方案1：Go (Gin/Kratos) |
| 前端技术栈 | 方案1：React + TypeScript |
| 沙箱隔离级别 | 方案1：普通容器隔离 |
| 日志存储 | 阿里云SLS |
| 任务调度（考虑限流、抢占、排队可视化） | 方案1：Redis队列 + 自研调度器 |
| 认证方式 | 先用API Key，后对接企业IAM系统 |
| Git集成 | 单一平台（京东行云），支持git协议 |
| 交付预期 | 1-2个月快速验证 |
| 前端服务 | 合并为一个服务 |
| Namespace划分 | 前端与控制面同一namespace，执行面单独namespace |
| 架构模式 | 模块化单体（预留微服务拆分接口） |
| API认证位置 | 服务内部（MSE只做路由转发） |

---

## 12. 外部集成

| 集成项 | 平台 | 说明 |
|--------|------|------|
| Git仓库 | 京东行云平台 | 支持git协议 |
| 企业IAM | 后续对接 | 预留扩展接口 |

---

## 13. 待办事项

- [ ] 输出完整TRD文档
- [ ] 详细数据库表结构设计
- [ ] API接口详细定义（OpenAPI）
- [ ] 前端UI原型设计
