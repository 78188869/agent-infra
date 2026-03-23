# MVP 设计结论汇总

> 日期：2026-03-21
> 状态：已确认

---

## 模块一：整体架构设计

### 架构图

```
┌─────────────────────────────────────────────────────────────────────┐
│                          MVP 技术架构                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                      前端层 (React SPA)                       │   │
│  │  ┌─────────────────┐              ┌─────────────────┐        │   │
│  │  │    用户面板     │              │    管理面板     │        │   │
│  │  │  (React SPA)    │              │  (React SPA)    │        │   │
│  │  └─────────────────┘              └─────────────────┘        │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                │                                     │
│                                ▼                                     │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                      API网关层 (阿里云MSE)                    │   │
│  │            (路由转发、SSL终结、日志)                          │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                │                                     │
│                                ▼                                     │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                    控制面服务 (Go/Gin)                        │   │
│  │  ┌─────────┬─────────┬─────────┬─────────┬─────────┐        │   │
│  │  │  API    │  调度器  │ 执行管理 │ 模板管理 │ 能力管理 │        │   │
│  │  │ Handler │ Scheduler│ Executor │ Template │Capability│       │   │
│  │  └─────────┴─────────┴─────────┴─────────┴─────────┘        │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                │                                     │
│         ┌──────────────────────┼──────────────────────┐             │
│         ▼                      ▼                      ▼             │
│  ┌─────────────┐       ┌─────────────┐       ┌─────────────┐       │
│  │   OceanBase │       │    Redis    │       │  阿里云SLS  │       │
│  │   (元数据)   │       │ (队列/缓存) │       │   (日志)    │       │
│  └─────────────┘       └─────────────┘       └─────────────┘       │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                      执行面 (K8s Pod)                         │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐           │   │
│  │  │  沙箱Pod 1  │  │  沙箱Pod 2  │  │  沙箱Pod N  │           │   │
│  │  │ ┌─────────┐ │  │ ┌─────────┐ │  │ ┌─────────┐ │           │   │
│  │  │ │Claude   │ │  │ │Claude   │ │  │ │Claude   │ │           │   │
│  │  │ │Code CLI │ │  │ │Code CLI │ │  │ │Code CLI │ │           │   │
│  │  │ └─────────┘ │  │ └─────────┘ │  │ └─────────┘ │           │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘           │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                     外部集成                                   │   │
│  │  ┌─────────────┐              ┌─────────────┐                 │   │
│  │  │ 京东行云    │              │  阿里云OSS  │                 │   │
│  │  │ (Git仓库)   │              │ (文件存储)  │                 │   │
│  │  └─────────────┘              └─────────────┘                 │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 核心组件说明

| 组件 | 技术选型 | 职责 |
|------|---------|------|
| 用户面板 | React + TypeScript + Ant Design | 开发者任务操作界面 |
| 管理面板 | React + TypeScript + Ant Design Pro | 管理员运维界面 |
| API网关 | 阿里云MSE | 路由转发、SSL终结 |
| 控制面服务 | Go (Gin) | 核心业务逻辑 |
| OceanBase | MySQL协议兼容 | 任务、模板、用户等元数据 |
| Redis | Redis 6.x | 任务队列、缓存、分布式锁 |
| 阿里云SLS | - | 执行日志、审计日志 |
| 沙箱Pod | Docker in K8s | 任务执行环境 |

---

## 模块二：控制面服务模块设计

### 内部模块结构

```
┌─────────────────────────────────────────────────────────────────────┐
│                     控制面服务 - 内部模块                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │                     API Handler 层                          │     │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐      │     │
│  │  │TaskAPI   │ │TemplateAPI│ │TenantAPI │ │SystemAPI │      │     │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘      │     │
│  └────────────────────────────────────────────────────────────┘     │
│                              │                                       │
│                              ▼                                       │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │                     Service 层                              │     │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐      │     │
│  │  │TaskSvc   │ │TemplateSvc│ │TenantSvc │ │UserSvc   │      │     │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘      │     │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐                   │     │
│  │  │CapabilSvc│ │AuditSvc  │ │NotifySvc │                   │     │
│  │  └──────────┘ └──────────┘ └──────────┘                   │     │
│  └────────────────────────────────────────────────────────────┘     │
│                              │                                       │
│                              ▼                                       │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │                     核心引擎层                              │     │
│  │  ┌────────────────────┐    ┌────────────────────┐         │     │
│  │  │   Task Scheduler   │    │   Task Executor    │         │     │
│  │  │   (任务调度器)      │    │   (执行管理器)      │         │     │
│  │  │  ┌──────────────┐  │    │  ┌──────────────┐  │         │     │
│  │  │  │ 优先级队列    │  │    │  │ K8s客户端    │  │         │     │
│  │  │  │ 限流器        │  │    │  │ Pod生命周期  │  │         │     │
│  │  │  │ 抢占逻辑      │  │    │  │ 日志采集    │  │         │     │
│  │  │  └──────────────┘  │    │  └──────────────┘  │         │     │
│  │  └────────────────────┘    └────────────────────┘         │     │
│  └────────────────────────────────────────────────────────────┘     │
│                              │                                       │
│                              ▼                                       │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │                     基础设施层                              │     │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐      │     │
│  │  │DB Client │ │Redis     │ │SLS       │ │K8s       │      │     │
│  │  │(OB)      │ │Client    │ │Client    │ │Client    │      │     │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘      │     │
│  └────────────────────────────────────────────────────────────┘     │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 核心模块职责

| 模块 | 职责 | 关键功能 |
|------|------|---------|
| **TaskAPI** | 任务相关HTTP接口 | 创建/查询/操作任务 |
| **TaskSvc** | 任务业务逻辑 | 状态管理、参数校验 |
| **Task Scheduler** | 任务调度引擎 | 队列管理、限流、抢占 |
| **Task Executor** | 执行管理 | 创建/销毁沙箱Pod、状态同步 |
| **TemplateSvc** | 模板管理 | CRUD、版本控制、校验 |
| **CapabilitySvc** | 能力管理 | 工具/Skills注册与授权 |

### 调度器核心逻辑

```
任务提交 ──▶ Redis优先级队列(High/Normal/Low)
                    │
                    ▼
            限流器 (租户级配额 + 全局并发控制)
                    │
                    ▼
            调度决策 (抢占判断 + 资源匹配)
                    │
                    ▼
            交给 Executor 执行
```

---

## 模块三：沙箱执行环境设计

### 沙箱Pod内部结构（复用Claude Code配置）

```
┌─────────────────────────────────────────────────────────────────────┐
│                        沙箱Pod 内部结构                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                     沙箱容器 (Docker)                        │   │
│  │                                                              │   │
│  │  ┌────────────────────────────────────────────────────────┐  │   │
│  │  │                  /workspace (代码仓库)                  │  │   │
│  │  │  ┌─────────────────────────────────────────────────┐   │  │   │
│  │  │  │  CLAUDE.md          (Claude Code 项目配置)       │   │  │   │
│  │  │  │  .claude/           (Claude Code 配置目录)       │   │  │   │
│  │  │  │  .mcp.json          (MCP工具配置，可选)          │   │  │   │
│  │  │  │  src/               (业务代码)                   │   │  │   │
│  │  │  └─────────────────────────────────────────────────┘   │  │   │
│  │  └────────────────────────────────────────────────────────┘  │   │
│  │                                                              │   │
│  │  ┌────────────────────────────────────────────────────────┐  │   │
│  │  │              Claude Code CLI (直接调用)                 │  │   │
│  │  │                                                          │  │   │
│  │  │  claude -p "${TASK_PROMPT}" \                           │  │   │
│  │  │        --max-tokens ${MAX_TOKENS} \                     │  │   │
│  │  │        --allowedTools "${TOOLS}" \                      │  │   │
│  │  │        --output-format stream-json                     │  │   │
│  │  │                                                          │  │   │
│  │  └────────────────────────────────────────────────────────┘  │   │
│  │                                                              │   │
│  │  ┌────────────────────────────────────────────────────────┐  │   │
│  │  │              Agent Wrapper (轻量封装)                   │  │   │
│  │  │  • 启动CLI并传递参数                                    │  │   │
│  │  │  • 解析CLI流式输出                                      │  │   │
│  │  │  • 状态上报到控制面                                     │  │   │
│  │  │  • 接收干预指令                                         │  │   │
│  │  └────────────────────────────────────────────────────────┘  │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │              Sidecar: Log Agent (日志采集)                   │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 复用Claude Code现有机制

| 配置项 | Claude Code机制 | 平台映射 |
|--------|----------------|---------|
| **项目指令** | CLAUDE.md | 从模板的任务描述生成 |
| **工具配置** | .mcp.json | 从模板的capabilities生成 |
| **权限控制** | --allowedTools | 从模板的tools列表映射 |
| **资源限制** | --max-tokens | 从模板的resources映射 |
| **输出格式** | --output-format stream-json | 固定使用流式JSON |

### 模板参数 → CLI参数映射

| 模板 YAML | CLI 参数 |
|-----------|---------|
| spec.goal | -p "${goal}" |
| spec.context.initialContext | 注入到CLAUDE.md |
| execution.resources.tokenLimit | --max-tokens |
| execution.capabilities.tools | --allowedTools |
| execution.timeout | 超时由Wrapper控制 |

### 沙箱与控制面通信

| 通信类型 | 方向 | 方式 |
|---------|------|------|
| 状态上报 | 沙箱 → 控制面 | HTTP POST，每状态变更触发 |
| 干预指令 | 控制面 → 沙箱 | HTTP POST，Wrapper暴露端口 |
| 日志采集 | 沙箱 → SLS | Log Agent自动上报 |
| 心跳检测 | 沙箱 → 控制面 | HTTP POST，每5s |

---

## 模块四：数据模型设计

### 核心实体关系

```
Tenant（租户）
    │
    ├── User（用户）
    │       │
    │       ├── Task（任务）───┬── ExecutionLog（执行日志）
    │       │                 ├── Output（输出产物）
    │       │                 ├── Intervention（干预记录）
    │       │                 └── Context（上下文）
    │       │
    │       └── Template（模板）
    │
    ├── Capability（能力）
    │       ├── Tool（工具）
    │       ├── Skill（技能包）
    │       └── AgentRuntime（Agent 运行时）
    │
    └── Knowledge（知识库）- v1.2+
```

### 核心表结构

#### tenants（租户表）
- id: VARCHAR(36) PRIMARY KEY
- name: VARCHAR(128)
- quota_cpu, quota_memory, quota_concurrency, quota_daily_tasks
- status: ENUM('active', 'suspended')
- created_at, updated_at

#### users（用户表）
- id: VARCHAR(36) PRIMARY KEY
- tenant_id: VARCHAR(36) FK
- username: VARCHAR(64)
- role: ENUM('developer', 'admin', 'operator', 'reviewer')
- status: ENUM('active', 'disabled')
- created_at, updated_at

#### api_keys（API密钥表）
- id: VARCHAR(36) PRIMARY KEY
- user_id: VARCHAR(36) FK
- key_hash: VARCHAR(128)
- key_prefix: VARCHAR(8)
- name, expires_at, last_used_at, created_at

#### templates（任务模板表）
- id: VARCHAR(36) PRIMARY KEY
- tenant_id: VARCHAR(36) FK
- name: VARCHAR(128)
- version: VARCHAR(32)
- description: TEXT
- spec: YAML
- status: ENUM('draft', 'published', 'deprecated')
- created_by, created_at, updated_at

#### tasks（任务表）
- id: VARCHAR(36) PRIMARY KEY
- tenant_id: VARCHAR(36) FK
- template_id: VARCHAR(36) FK
- creator_id: VARCHAR(36) FK
- parent_task_id: VARCHAR(36)
- status: ENUM('pending', 'scheduled', 'running', 'paused', 'waiting_approval', 'retrying', 'succeeded', 'failed', 'cancelled')
- params: JSON
- resolved_spec: YAML
- pod_name: VARCHAR(128)
- sandbox_id: VARCHAR(64)
- result: JSON
- error_message: TEXT
- metrics: JSON
- started_at, finished_at, created_at, updated_at

#### execution_logs（执行日志表）
- id: BIGINT AUTO_INCREMENT PRIMARY KEY
- task_id: VARCHAR(36) FK
- event_type: ENUM('status_change', 'tool_call', 'tool_result', 'llm_input', 'llm_output', 'error', 'heartbeat', 'intervention', 'metric')
- content: JSON
- timestamp: TIMESTAMP(3)

#### interventions（人工干预记录表）
- id: VARCHAR(36) PRIMARY KEY
- task_id: VARCHAR(36) FK
- operator_id: VARCHAR(36) FK
- action: ENUM('pause', 'resume', 'cancel', 'inject', 'modify')
- content: JSON
- result: JSON
- created_at

#### capabilities（能力注册表）
- id: VARCHAR(36) PRIMARY KEY
- tenant_id: VARCHAR(36)
- type: ENUM('tool', 'skill', 'agent_runtime')
- name: VARCHAR(64)
- description: TEXT
- config: JSON
- permission_level: ENUM('public', 'restricted', 'admin_only')
- status: ENUM('active', 'inactive')
- created_at

### Redis数据结构

| Key模式 | 类型 | 说明 | TTL |
|---------|------|------|-----|
| queue:tasks:high | List | 高优先级任务队列 | 永久 |
| queue:tasks:normal | List | 普通优先级任务队列 | 永久 |
| queue:tasks:low | List | 低优先级任务队列 | 永久 |
| task:{task_id}:meta | Hash | 任务调度元数据 | 24h |
| tenant:{tenant_id}:quota | Hash | 租户实时配额使用 | 永久 |
| sandbox:{task_id}:heartbeat | String | 沙箱心跳时间戳 | 5min |
| lock:task:{task_id} | String | 任务操作分布式锁 | 30s |

---

## 模块五：API设计概要

### API分层

```
阿里云MSE (API网关)
    • 路由转发
    • SSL终结
    • 日志记录
        │
        ▼
Go服务 API路由
/api/v1/
├── /tasks          任务管理
├── /templates      模板管理
├── /tenants        租户管理 (管理面板)
├── /users          用户管理
├── /capabilities   能力管理 (管理面板)
├── /metrics        监控指标
└── /internal       内部接口 (沙箱回调)
```

### 认证流程（服务内部）

```
Client ──▶ MSE网关(路由转发) ──▶ Go服务
                                    │
                                    ▼
                            Auth Middleware
                            ┌──────────────┐
                            │提取API Key   │
                            │查询Redis缓存 │
                            │查DB验证      │
                            │注入用户上下文│
                            └──────────────┘
                                    │
                                    ▼
                            Handler (业务逻辑)
```

### 核心API列表

#### 任务管理 /api/v1/tasks
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /tasks | 创建任务 |
| GET | /tasks | 任务列表 |
| GET | /tasks/:id | 任务详情 |
| POST | /tasks/:id/pause | 暂停任务 |
| POST | /tasks/:id/resume | 恢复任务 |
| POST | /tasks/:id/cancel | 取消任务 |
| POST | /tasks/:id/inject | 注入指令 |
| GET | /tasks/:id/logs | 获取日志 |
| GET | /tasks/:id/queue-position | 查询排队位置 |

#### 模板管理 /api/v1/templates
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /templates | 模板列表 |
| GET | /templates/:id | 模板详情 |
| POST | /templates | 创建模板 |
| PUT | /templates/:id | 更新模板 |
| POST | /templates/:id/publish | 发布模板 |

#### 内部接口 /api/v1/internal
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /internal/tasks/:id/events | 上报执行事件 |
| POST | /internal/tasks/:id/heartbeat | 心跳上报 |
| POST | /internal/tasks/:id/complete | 任务完成通知 |

### API响应格式

**成功响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

**错误响应**：
```json
{
  "code": 10001,
  "message": "任务不存在",
  "request_id": "req-xxx"
}
```

---

## 模块六：部署架构设计

### Namespace划分

```
┌─────────────────────────────────────────────────────────────────────┐
│                   control-plane Namespace                           │
│                                                                      │
│  控制面服务 (Deployment: control-plane)                             │
│  ┌───────────┐  ┌───────────┐                                       │
│  │  Pod 1    │  │  Pod 2    │   (2副本, 2核4G)                      │
│  └───────────┘  └───────────┘                                       │
│                                                                      │
│  前端服务 (Deployment: frontend)                                    │
│  ┌───────────────────────────────────────────────────┐              │
│  │  合并服务: 用户面板(/user/*) + 管理面板(/admin/*)  │              │
│  └───────────────────────────────────────────────────┘              │
│  ┌───────────┐  ┌───────────┐                                       │
│  │  Pod 1    │  │  Pod 2    │   (2副本, 1核2G)                      │
│  └───────────┘  └───────────┘                                       │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                     sandbox Namespace                               │
│                                                                      │
│  沙箱Pod (动态创建)                                                  │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐                       │
│  │ sandbox-1 │  │ sandbox-2 │  │ sandbox-N │   (按模板配置)        │
│  └───────────┘  └───────────┘  └───────────┘                       │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 资源配额估算（MVP）

| 资源 | 配置 | 说明 |
|------|------|------|
| 控制面Pod | 2核4G x 2副本 | API + 调度 + 执行管理 |
| 前端Pod | 1核2G x 2副本 | 用户面板+管理面板合并 |
| 沙箱Pod | 2核4G ~ 4核8G | 按任务模板配置，动态创建 |
| 最大并发沙箱 | 50个 | 初始配额，按需扩展 |

---

## 模块七：错误处理与测试策略

### 错误码规范

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

### 关键场景错误处理

| 场景 | 处理策略 |
|------|---------|
| 沙箱启动失败 | 重试3次，间隔指数退避，失败后标记任务Failed |
| 沙箱心跳丢失 | 等待15s（3次心跳），仍无响应则标记异常 |
| 任务执行超时 | 根据模板配置：fail或pause |
| 资源配额超限 | 拒绝创建任务，返回429错误 |
| 数据库连接失败 | 重试+告警，优雅降级 |

### 测试策略

```
测试金字塔:
           /‾‾‾‾‾‾\
          /  E2E   \          5%
         /──────────\
        /  集成测试   \        20%
       /──────────────\
      /    单元测试     \      75%
     /────────────────────\
```

| 层级 | 覆盖内容 | 工具 |
|------|---------|------|
| 单元测试 | Service层业务逻辑、工具函数 | Go testing |
| 集成测试 | API端到端、数据库交互 | Go testing + testify |
| E2E测试 | 核心链路（创建任务→执行→完成） | 手动测试 |

### 监控告警

| 监控项 | 告警阈值 | 级别 |
|--------|---------|------|
| 控制面服务不可用 | 持续1min | P0 |
| API错误率 | > 5% | P1 |
| 任务失败率 | > 20% | P1 |
| 沙箱Pod创建失败 | 连续3次 | P1 |
| Redis连接延迟 | > 100ms | P2 |

---

## 模块八：构建与部署配置

### 项目结构

```
agent-infra/
├── cmd/
│   └── control-plane/          # 控制面服务入口
│       └── main.go
├── internal/
│   ├── api/                    # API Handler
│   ├── service/                # 业务逻辑
│   ├── scheduler/              # 任务调度器
│   ├── executor/               # 执行管理器
│   ├── model/                  # 数据模型
│   └── config/                 # 配置管理
├── pkg/                        # 公共库
├── api/
│   └── openapi.yaml           # API定义
├── web/
│   ├── user-panel/            # 用户面板
│   └── admin-panel/           # 管理面板
├── deploy/
│   ├── k8s/                   # K8s配置
│   │   ├── control-plane/
│   │   └── sandbox/
│   └── dockerfiles/
│       ├── Dockerfile.control-plane
│       ├── Dockerfile.frontend
│       └── Dockerfile.sandbox-base
├── scripts/
│   ├── agent-wrapper.sh
│   └── migrate.sh
├── Makefile
├── go.mod
└── go.sum
```

### Dockerfile - 控制面服务

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o control-plane ./cmd/control-plane

# Runtime stage
FROM alpine:3.19
WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /build/control-plane .
RUN adduser -D -u 1000 appuser
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1
ENTRYPOINT ["./control-plane"]
```

### Dockerfile - 前端服务

```dockerfile
# Build stage
FROM node:20-alpine AS builder
WORKDIR /build

# 用户面板构建
COPY web/user-panel/package*.json ./user-panel/
WORKDIR /build/user-panel
RUN npm ci
COPY web/user-panel/ .
RUN npm run build

# 管理面板构建
WORKDIR /build
COPY web/admin-panel/package*.json ./admin-panel/
WORKDIR /build/admin-panel
RUN npm ci
COPY web/admin-panel/ .
RUN npm run build

# Runtime stage
FROM nginx:alpine
COPY --from=builder /build/user-panel/dist /usr/share/nginx/html/user
COPY --from=builder /build/admin-panel/dist /usr/share/nginx/html/admin
COPY deploy/dockerfiles/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

### Dockerfile - 沙箱基础镜像

```dockerfile
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    git curl wget vim jq \
    python3 python3-pip \
    nodejs npm \
    && rm -rf /var/lib/apt/lists/*

# 安装 Claude Code CLI
RUN curl -fsSL https://claude.ai/install.sh | sh

# 安装 Agent Wrapper
COPY scripts/agent-wrapper.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/agent-wrapper.sh

RUN mkdir -p /workspace && chmod 777 /workspace
WORKDIR /workspace

RUN useradd -m -u 1000 sandbox
USER sandbox

ENTRYPOINT ["/usr/local/bin/agent-wrapper.sh"]
```

### Makefile 核心命令

```makefile
# 开发命令
make build              # 构建控制面服务
make run                # 本地运行
make test               # 运行测试
make lint               # 代码检查

# 前端命令
make web-install        # 安装依赖
make web-build          # 构建前端

# Docker命令
make docker-build-all   # 构建所有镜像
make docker-push        # 推送镜像

# K8s命令
make k8s-apply          # 部署到K8s
make k8s-status         # 查看状态
make k8s-logs           # 查看日志

# 指定版本
VERSION=v1.0.0 make docker-build-all
```
