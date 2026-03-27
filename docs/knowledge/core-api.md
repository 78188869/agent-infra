# Core API Knowledge

> **Last Updated**: 2026-03-26
> **PRD Version**: v0.7-draft
> **TRD Version**: v2.4

## 1. Overview

Core API 模块负责租户管理、任务管理、模板管理、Provider管理、Capability管理等核心 HTTP API 的设计与实现。

**模块职责**：
- 租户 CRUD 操作与配额管理
- 任务 CRUD 操作与状态管理
- 模板 CRUD 操作与版本管理
- Provider CRUD 操作与连接测试
- Capability CRUD 操作与激活管理
- 健康检查与就绪检查
- API Key 认证与权限校验
- 请求参数校验与响应序列化

**核心概念**：
- **Tenant**: 租户，平台的多租户隔离单元，包含资源配额
- **Task**: 一次 Agent 执行的抽象单元，是平台的核心业务对象
- **Template**: 可复用的任务配置，定义任务类型和执行策略
- **Provider**: AI 能力提供者，定义底层 AI 服务的连接和配置
- **Capability**: 平台能力，包括工具、技能和 Agent 运行时
- **API Key**: 用户认证凭证，用于 API 调用身份验证

## 2. Product Requirements (from PRD)

### 2.1 用户故事

| 故事ID | 描述 | 验收标准 |
|--------|------|---------|
| US-D01a | 基于模板创建任务 | 可浏览搜索模板、选择后配置参数、校验通过后预览提交 |
| US-D01b | 任务调试 | 支持查看中间状态、注入测试指令、调试操作记录到日志 |
| US-D02 | 任务执行监控 | 实时显示进度阶段、流式输出日志、显示资源消耗 |
| US-D03 | 人工干预任务 | 支持暂停/恢复/取消、注入指令、修改参数、记录干预历史 |
| US-A02 | 任务模板管理 | 创建/编辑/发布/废弃模板、YAML格式定义、版本化管理 |
| US-A03 | 租户管理 | 创建/编辑/删除租户、配额管理、状态管理 |
| US-A04 | Provider管理 | 注册/配置/测试Provider、设置默认Provider |
| US-A05 | Capability管理 | 注册/激活/停用Capability、权限级别管理 |

### 2.2 业务规则

1. **租户管理**：
   - 租户名称必须唯一
   - 配额值必须为正数
   - 暂停的租户无法创建新任务

2. **任务创建**：
   - 必须选择有效的模板（可选）
   - 参数必须通过模板定义的校验规则
   - 必须通过配额检查（租户级）

3. **任务状态转换**：
   - 只能从当前状态转换到合法的目标状态
   - 终态任务（succeeded/failed/cancelled）不可再变更

4. **模板发布**：
   - 只有 draft 状态的模板可以发布
   - 发布后的模板不可删除，只能废弃

5. **Provider管理**：
   - Provider 有三种作用域：system、tenant、user
   - tenant 作用域必须关联 tenant_id
   - user 作用域必须关联 user_id

6. **Capability管理**：
   - Capability 有三种类型：tool、skill、agent_runtime
   - 租户级 Capability 必须关联 tenant_id
   - 全局 Capability 的 tenant_id 为 NULL

## 3. Technical Design (from TRD)

### 3.1 架构设计

```
control-plane/
├── internal/
│   ├── api/
│   │   ├── handler/
│   │   │   ├── tenant.go      # 租户 HTTP 处理器
│   │   │   ├── task.go        # 任务 HTTP 处理器
│   │   │   ├── template.go    # 模板 HTTP 处理器
│   │   │   ├── provider.go    # Provider HTTP 处理器
│   │   │   ├── capability.go  # Capability HTTP 处理器
│   │   │   ├── health.go      # 健康检查处理器
│   │   │   └── user.go        # 用户 HTTP 处理器
│   │   ├── middleware/
│   │   │   ├── auth.go        # API Key 认证
│   │   │   ├── ratelimit.go   # 限流
│   │   │   └── logger.go      # 日志
│   │   └── router.go          # 路由注册
│   └── service/
│       ├── tenant_service.go    # 租户业务逻辑
│       ├── task_service.go      # 任务业务逻辑
│       ├── template_service.go  # 模板业务逻辑
│       ├── provider_service.go  # Provider 业务逻辑
│       └── capability_service.go # Capability 业务逻辑
```

### 3.2 通用规范

#### 3.2.1 Base URL

所有 API 均在 `/api/v1/` 路径下。

#### 3.2.2 通用响应格式

```go
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

type PaginatedData struct {
    Items    interface{} `json:"items"`
    Total    int64       `json:"total"`
    Page     int         `json:"page"`
    PageSize int         `json:"page_size"`
}
```

#### 3.2.3 错误码定义

| Code | HTTP Status | 说明 |
|------|-------------|------|
| 0 | 200 | 成功 |
| 400 | 400 | 请求参数错误 |
| 401 | 401 | 未授权 |
| 404 | 404 | 资源不存在 |
| 500 | 500 | 内部服务器错误 |

### 3.3 认证机制

**API Key 认证流程**：
1. 请求头携带 `Authorization: Bearer <api_key>`
2. Auth 中间件提取并验证 API Key
3. 计算密钥 SHA256 哈希值，查询数据库匹配
4. 验证通过后注入用户上下文到请求

**API Key 存储规范**：
- 只存储 SHA256 哈希值，不存储明文
- 存储前 8 位前缀用于识别
- 支持过期时间和撤销操作

### 3.4 API 规范

#### 3.4.1 租户管理 `/api/v1/tenants`

| 方法 | 路径 | 说明 | 角色 |
|------|------|------|------|
| POST | /tenants | 创建租户 | admin |
| GET | /tenants | 租户列表 | admin |
| GET | /tenants/:id | 租户详情 | admin |
| PUT | /tenants/:id | 更新租户 | admin |
| DELETE | /tenants/:id | 删除租户 | admin |

**租户模型**：

```go
type Tenant struct {
    BaseModel                              // ID, CreatedAt, UpdatedAt
    Name             string `json:"name"`  // Required, 唯一
    QuotaCPU         int    `json:"quota_cpu"`         // Default: 4
    QuotaMemory      int64  `json:"quota_memory"`      // GB, Default: 16
    QuotaConcurrency int    `json:"quota_concurrency"` // Default: 10
    QuotaDailyTasks  int    `json:"quota_daily_tasks"` // Default: 100
    Status           string `json:"status"`            // active, suspended
}
```

**创建租户请求**：

```http
POST /api/v1/tenants
Content-Type: application/json

{
    "name": "acme-corp",
    "quota_cpu": 8,
    "quota_memory": 32,
    "quota_concurrency": 20,
    "quota_daily_tasks": 500
}
```

**成功响应**：

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "acme-corp",
        "quota_cpu": 8,
        "quota_memory": 32,
        "quota_concurrency": 20,
        "quota_daily_tasks": 500,
        "status": "active",
        "created_at": "2026-03-26T10:00:00Z",
        "updated_at": "2026-03-26T10:00:00Z"
    }
}
```

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页数量，默认 10，最大 100 |
| status | string | 状态过滤：active, suspended |
| search | string | 名称搜索 |

**列表响应**：

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "items": [...],
        "total": 100,
        "page": 1,
        "page_size": 20
    }
}
```

**更新租户请求**：

```http
PUT /api/v1/tenants/:id
Content-Type: application/json

{
    "name": "acme-corp-updated",
    "quota_cpu": 16,
    "status": "active"
}
```

**业务规则**：
- name 必填且唯一
- quota 值必须为正数
- status 必须为 active 或 suspended

#### 3.4.2 任务管理 `/api/v1/tasks`

> **实现状态说明**：
> - ✅ 基础 CRUD 已实现
> - 🔄 控制类端点（pause/resume/cancel 等）计划在后续版本实现

| 方法 | 路径 | 说明 | 角色 | 状态 |
|------|------|------|------|------|
| POST | /tasks | 创建任务 | developer | ✅ |
| GET | /tasks | 任务列表 | developer | ✅ |
| GET | /tasks/:id | 任务详情 | developer | ✅ |
| PUT | /tasks/:id | 更新任务 | developer | ✅ |
| DELETE | /tasks/:id | 删除任务 | developer | ✅ |
| POST | /tasks/:id/pause | 暂停任务 | developer, operator | 🔄 |
| POST | /tasks/:id/resume | 恢复任务 | developer, operator | 🔄 |
| POST | /tasks/:id/cancel | 取消任务 | developer, operator | 🔄 |
| POST | /tasks/:id/inject | 注入指令 | developer, operator | 🔄 |
| POST | /tasks/:id/retry | 重试任务 | developer, operator | 🔄 |
| GET | /tasks/:id/logs | 获取日志 | developer, operator | 🔄 |
| GET | /tasks/:id/queue-position | 查询排队位置 | developer | 🔄 |
| GET | /tasks/:id/metrics | 获取执行指标 | developer | 🔄 |

**任务模型**：

```go
type Task struct {
    BaseModel
    TenantID     string         `json:"tenant_id"`
    TemplateID   *string        `json:"template_id,omitempty"`
    CreatorID    string         `json:"creator_id"`
    ProviderID   string         `json:"provider_id"`
    Name         string         `json:"name"`
    Status       string         `json:"status"`       // pending, scheduled, running, paused, waiting_approval, retrying, succeeded, failed, cancelled
    Priority     string         `json:"priority"`     // high, normal, low
    Params       datatypes.JSON `json:"params,omitempty"`
    Description  string         `json:"description,omitempty"`
    ErrorMessage string         `json:"error_message,omitempty"`
    Result       datatypes.JSON `json:"result,omitempty"`
}
```

**创建任务请求**：

```http
POST /api/v1/tasks
Content-Type: application/json

{
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "template_id": "660e8400-e29b-41d4-a716-446655440001",
    "creator_id": "770e8400-e29b-41d4-a716-446655440002",
    "provider_id": "880e8400-e29b-41d4-a716-446655440003",
    "name": "Fix authentication bug",
    "priority": "high",
    "description": "修复登录页面的认证问题",
    "params": {
        "repo": "example/repo",
        "issue": 123
    }
}
```

**成功响应**：

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "id": "990e8400-e29b-41d4-a716-446655440004",
        "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
        "template_id": "660e8400-e29b-41d4-a716-446655440001",
        "creator_id": "770e8400-e29b-41d4-a716-446655440002",
        "provider_id": "880e8400-e29b-41d4-a716-446655440003",
        "name": "Fix authentication bug",
        "status": "pending",
        "priority": "high",
        "description": "修复登录页面的认证问题",
        "params": {
            "repo": "example/repo",
            "issue": 123
        },
        "created_at": "2026-03-26T10:00:00Z",
        "updated_at": "2026-03-26T10:00:00Z"
    }
}
```

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页数量，默认 10，最大 100 |
| status | string | 状态过滤 |
| priority | string | 优先级过滤 |
| tenant_id | string | 租户ID过滤 |
| search | string | 名称搜索 |

**状态转换规则**：

| 当前状态 | 可转换目标状态 |
|----------|----------------|
| pending | scheduled, cancelled |
| scheduled | running, cancelled |
| running | paused, succeeded, failed, cancelled |
| paused | running, cancelled |
| waiting_approval | running, cancelled |
| retrying | running, failed, cancelled |
| succeeded | - (终态) |
| failed | - (终态) |
| cancelled | - (终态) |

**业务规则**：
- tenant_id、creator_id、provider_id、name 必填
- template_id 可选
- priority: high, normal, low（默认 normal）
- 终态任务不可变更

#### 3.4.3 模板管理 `/api/v1/templates`

> **实现状态说明**：
> - ✅ 基础 CRUD 已实现
> - 🔄 生命周期管理端点（publish/deprecate 等）计划在后续版本实现

| 方法 | 路径 | 说明 | 角色 | 状态 |
|------|------|------|------|------|
| GET | /templates | 模板列表 | developer | ✅ |
| GET | /templates/:id | 模板详情 | developer | ✅ |
| POST | /templates | 创建模板 | admin | ✅ |
| PUT | /templates/:id | 更新模板 | admin | ✅ |
| DELETE | /templates/:id | 删除模板 | admin | ✅ |
| POST | /templates/:id/publish | 发布模板 | admin | 🔄 |
| POST | /templates/:id/deprecate | 废弃模板 | admin | 🔄 |
| GET | /templates/:id/versions | 版本历史 | admin | 🔄 |
| POST | /templates/:id/validate | 校验模板 | admin | 🔄 |

**模板模型**：

```go
type Template struct {
    BaseModel
    TenantID   string  `json:"tenant_id"`
    Name       string  `json:"name"`
    Version    string  `json:"version"`      // Default: 1.0.0
    Spec       string  `json:"spec"`         // YAML format
    SceneType  string  `json:"scene_type"`   // coding, ops, analysis, content, custom
    Status     string  `json:"status"`       // draft, published, deprecated
    ProviderID *string `json:"provider_id,omitempty"`
}
```

**创建模板请求**：

```http
POST /api/v1/templates
Content-Type: application/json

{
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Code Review Template",
    "version": "1.0.0",
    "spec": "apiVersion: agent/v1\nkind: Template\nmetadata:\n  name: code-review\nspec:\n  steps:\n    - name: analyze\n      action: analyze_code",
    "scene_type": "coding",
    "provider_id": "880e8400-e29b-41d4-a716-446655440003"
}
```

**成功响应**：

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "id": "a90e8400-e29b-41d4-a716-446655440005",
        "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "Code Review Template",
        "version": "1.0.0",
        "spec": "apiVersion: agent/v1\nkind: Template\n...",
        "scene_type": "coding",
        "status": "draft",
        "provider_id": "880e8400-e29b-41d4-a716-446655440003",
        "created_at": "2026-03-26T10:00:00Z",
        "updated_at": "2026-03-26T10:00:00Z"
    }
}
```

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页数量，默认 10，最大 100 |
| status | string | 状态过滤：draft, published, deprecated |
| scene_type | string | 场景类型过滤 |
| tenant_id | string | 租户ID过滤 |

**业务规则**：
- name、tenant_id 必填
- scene_type: coding, ops, analysis, content, custom
- status: draft, published, deprecated
- spec 必须为有效的 YAML 格式
- 只有 draft 状态的模板可以删除

#### 3.4.4 Provider 管理 `/api/v1/providers`

| 方法 | 路径 | 说明 | 角色 |
|------|------|------|------|
| POST | /providers | 创建 Provider | admin |
| GET | /providers | Provider 列表 | developer, admin |
| GET | /providers/available | 可用 Provider 列表 | developer, admin |
| GET | /providers/:id | Provider 详情 | developer, admin |
| PUT | /providers/:id | 更新 Provider | admin |
| DELETE | /providers/:id | 删除 Provider | admin |
| POST | /providers/:id/test | 测试连接 | admin |
| PUT | /providers/:id/set-default | 设置默认 | admin |

**Provider 模型**：

```go
type Provider struct {
    ID             string         `json:"id"`
    Scope          ProviderScope  `json:"scope"`          // system, tenant, user
    TenantID       *string        `json:"tenant_id"`
    UserID         *string        `json:"user_id"`
    Name           string         `json:"name"`
    Type           ProviderType   `json:"type"`           // claude_code, anthropic_compatible, openai_compatible, custom
    Description    string         `json:"description"`
    APIEndpoint    string         `json:"api_endpoint"`
    APIKeyRef      string         `json:"api_key_ref"`
    ModelMapping   datatypes.JSON `json:"model_mapping"`
    RuntimeType    RuntimeType    `json:"runtime_type"`   // cli, api, sdk
    RuntimeImage   string         `json:"runtime_image"`
    RuntimeCommand datatypes.JSON `json:"runtime_command"`
    EnvVars        datatypes.JSON `json:"env_vars"`
    Permissions    datatypes.JSON `json:"permissions"`
    EnabledPlugins datatypes.JSON `json:"enabled_plugins"`
    ExtraParams    datatypes.JSON `json:"extra_params"`
    Status         ProviderStatus `json:"status"`         // active, inactive, deprecated
    CreatedAt      time.Time      `json:"created_at"`
    UpdatedAt      time.Time      `json:"updated_at"`
}
```

**创建 Provider 请求**：

```http
POST /api/v1/providers
Content-Type: application/json

{
    "name": "Claude Code Provider",
    "type": "claude_code",
    "scope": "tenant",
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "description": "Primary Claude Code instance for Acme Corp",
    "api_endpoint": "https://api.anthropic.com",
    "api_key_ref": "k8s-secret://claude-api-key",
    "model_mapping": {
        "default": "claude-3-opus",
        "sonnet": "claude-3-sonnet",
        "haiku": "claude-3-haiku"
    },
    "runtime_type": "cli",
    "runtime_image": "claude-code:latest",
    "runtime_command": {
        "entrypoint": "/usr/local/bin/claude",
        "args": ["--model", "${MODEL}"]
    },
    "env_vars": {
        "ANTHROPIC_API_KEY": "${API_KEY_REF}"
    },
    "permissions": {
        "allow_network": true,
        "allow_filesystem": true
    },
    "enabled_plugins": ["git", "docker"]
}
```

**成功响应**：

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "id": "ba0e8400-e29b-41d4-a716-446655440006",
        "scope": "tenant",
        "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "Claude Code Provider",
        "type": "claude_code",
        "description": "Primary Claude Code instance for Acme Corp",
        "api_endpoint": "https://api.anthropic.com",
        "status": "active",
        "created_at": "2026-03-26T10:00:00Z",
        "updated_at": "2026-03-26T10:00:00Z"
    }
}
```

**测试连接请求**：

```http
POST /api/v1/providers/:id/test
Content-Type: application/json
```

**测试连接响应**：

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "success": true,
        "message": "Connection successful",
        "response_time_ms": 150,
        "details": {
            "api_reachable": true,
            "auth_valid": true,
            "models_available": ["claude-3-opus", "claude-3-sonnet", "claude-3-haiku"]
        }
    }
}
```

**设置默认 Provider**：

```http
PUT /api/v1/providers/:id/set-default
Content-Type: application/json
```

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页数量，默认 10，最大 100 |
| scope | string | 作用域过滤：system, tenant, user |
| type | string | 类型过滤 |
| status | string | 状态过滤：active, inactive, deprecated |

**业务规则**：
- name 必填
- scope: system, tenant, user
- tenant 作用域必须提供 tenant_id
- user 作用域必须提供 user_id
- type: claude_code, anthropic_compatible, openai_compatible, custom
- status: active, inactive, deprecated

#### 3.4.5 Capability 管理 `/api/v1/capabilities`

| 方法 | 路径 | 说明 | 角色 |
|------|------|------|------|
| POST | /capabilities | 创建 Capability | admin |
| GET | /capabilities | Capability 列表 | developer, admin |
| GET | /capabilities/:id | Capability 详情 | developer, admin |
| PUT | /capabilities/:id | 更新 Capability | admin |
| DELETE | /capabilities/:id | 删除 Capability | admin |
| POST | /capabilities/:id/activate | 激活 Capability | admin |
| POST | /capabilities/:id/deactivate | 停用 Capability | admin |

**Capability 模型**：

```go
type Capability struct {
    ID              string           `json:"id"`
    TenantID        *string          `json:"tenant_id"`       // NULL = global
    Type            CapabilityType   `json:"type"`            // tool, skill, agent_runtime
    Name            string           `json:"name"`
    Description     string           `json:"description"`
    Version         string           `json:"version"`         // Default: 1.0.0
    Config          datatypes.JSON   `json:"config"`
    Schema          datatypes.JSON   `json:"schema"`
    PermissionLevel PermissionLevel  `json:"permission_level"` // public, restricted, admin_only
    Status          CapabilityStatus `json:"status"`           // active, inactive
    CreatedAt       time.Time        `json:"created_at"`
    UpdatedAt       time.Time        `json:"updated_at"`
}
```

**创建 Capability 请求**：

```http
POST /api/v1/capabilities
Content-Type: application/json

{
    "type": "tool",
    "name": "Code Linter",
    "description": "Lint code files using ESLint and Prettier",
    "version": "1.0.0",
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "permission_level": "public",
    "config": {
        "command": "eslint",
        "args": ["--fix", "--config", ".eslintrc.json"],
        "timeout_ms": 30000
    },
    "schema": {
        "type": "object",
        "properties": {
            "files": {
                "type": "array",
                "items": {"type": "string"},
                "description": "Files to lint"
            },
            "fix": {
                "type": "boolean",
                "default": true,
                "description": "Auto-fix issues"
            }
        },
        "required": ["files"]
    }
}
```

**成功响应**：

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "id": "cb0e8400-e29b-41d4-a716-446655440007",
        "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
        "type": "tool",
        "name": "Code Linter",
        "description": "Lint code files using ESLint and Prettier",
        "version": "1.0.0",
        "permission_level": "public",
        "status": "active",
        "config": {
            "command": "eslint",
            "args": ["--fix", "--config", ".eslintrc.json"],
            "timeout_ms": 30000
        },
        "created_at": "2026-03-26T10:00:00Z",
        "updated_at": "2026-03-26T10:00:00Z"
    }
}
```

**激活 Capability**：

```http
POST /api/v1/capabilities/:id/activate
Content-Type: application/json
```

**激活响应**：

```json
{
    "code": 0,
    "message": "Capability activated successfully",
    "data": {
        "id": "cb0e8400-e29b-41d4-a716-446655440007",
        "status": "active"
    }
}
```

**停用 Capability**：

```http
POST /api/v1/capabilities/:id/deactivate
Content-Type: application/json
```

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页数量，默认 10，最大 100 |
| type | string | 类型过滤：tool, skill, agent_runtime |
| status | string | 状态过滤：active, inactive |
| tenant_id | string | 租户ID过滤 |

**业务规则**：
- name 必填
- type: tool, skill, agent_runtime
- permission_level: public, restricted, admin_only
- tenant_id 为 NULL 表示全局 Capability
- 新创建的 Capability 默认为 active 状态，可通过 /deactivate 端点停用

#### 3.4.6 健康检查 (无需认证)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /health | 健康检查 |
| GET | /ready | 就绪检查 |

**健康检查请求**：

```http
GET /health
```

**健康检查响应**：

```json
{
    "status": "healthy",
    "service": "control-plane",
    "version": "0.1.0"
}
```

**就绪检查请求**：

```http
GET /ready
```

**就绪检查响应**：

```json
{
    "ready": true,
    "checks": {
        "database": "healthy",
        "redis": "healthy"
    }
}
```

**不健康响应示例**：

```json
{
    "ready": false,
    "checks": {
        "database": "healthy",
        "redis": "unhealthy: connection refused"
    }
}
```

### 3.5 Service 接口定义

```go
// TenantService 接口
type TenantService interface {
    CreateTenant(ctx context.Context, req *CreateTenantRequest) (*Tenant, error)
    GetTenant(ctx context.Context, tenantID string) (*Tenant, error)
    ListTenants(ctx context.Context, filter *TenantFilter) ([]*Tenant, int64, error)
    UpdateTenant(ctx context.Context, tenantID string, req *UpdateTenantRequest) error
    DeleteTenant(ctx context.Context, tenantID string) error
}

// TaskService 接口
type TaskService interface {
    CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error)
    GetTask(ctx context.Context, taskID string) (*Task, error)
    ListTasks(ctx context.Context, filter *TaskFilter) ([]*Task, int64, error)
    UpdateTask(ctx context.Context, taskID string, req *UpdateTaskRequest) error
    DeleteTask(ctx context.Context, taskID string) error
    UpdateTaskStatus(ctx context.Context, taskID string, status TaskStatus, reason string) error
    PauseTask(ctx context.Context, taskID string) error
    ResumeTask(ctx context.Context, taskID string) error
    CancelTask(ctx context.Context, taskID string, reason string) error
    InjectInstruction(ctx context.Context, taskID string, content string) error
    ValidateTaskParams(ctx context.Context, templateID string, params map[string]interface{}) error
    GetQueuePosition(ctx context.Context, taskID string) (int, error)
}

// TemplateService 接口
type TemplateService interface {
    CreateTemplate(ctx context.Context, req *CreateTemplateRequest) (*Template, error)
    GetTemplate(ctx context.Context, templateID string) (*Template, error)
    ListTemplates(ctx context.Context, filter *TemplateFilter) ([]*Template, int64, error)
    UpdateTemplate(ctx context.Context, templateID string, req *UpdateTemplateRequest) error
    DeleteTemplate(ctx context.Context, templateID string) error
    GetTemplateVersions(ctx context.Context, templateID string) ([]*TemplateVersion, error)
    PublishTemplate(ctx context.Context, templateID string) error
    DeprecateTemplate(ctx context.Context, templateID string) error
    ValidateTemplate(ctx context.Context, spec *TemplateSpec) error
    RenderTemplate(ctx context.Context, templateID string, params map[string]interface{}) (*ResolvedSpec, error)
}

// ProviderService 接口
type ProviderService interface {
    CreateProvider(ctx context.Context, req *CreateProviderRequest) (*Provider, error)
    GetProvider(ctx context.Context, providerID string) (*Provider, error)
    ListProviders(ctx context.Context, filter *ProviderFilter) ([]*Provider, int64, error)
    GetAvailableProviders(ctx context.Context, userID string) ([]*Provider, error)
    UpdateProvider(ctx context.Context, providerID string, req *UpdateProviderRequest) error
    DeleteProvider(ctx context.Context, providerID string) error
    TestProviderConnection(ctx context.Context, providerID string) (*ConnectionTestResult, error)
    SetDefaultProvider(ctx context.Context, userID string, providerID string) error
}

// CapabilityService 接口
type CapabilityService interface {
    CreateCapability(ctx context.Context, req *CreateCapabilityRequest) (*Capability, error)
    GetCapability(ctx context.Context, capabilityID string) (*Capability, error)
    ListCapabilities(ctx context.Context, filter *CapabilityFilter) ([]*Capability, int64, error)
    UpdateCapability(ctx context.Context, capabilityID string, req *UpdateCapabilityRequest) error
    DeleteCapability(ctx context.Context, capabilityID string) error
    ActivateCapability(ctx context.Context, capabilityID string) error
    DeactivateCapability(ctx context.Context, capabilityID string) error
}
```

## 4. Implementation Notes

### 4.1 关键实现要点

1. **请求校验**：使用 Gin 框架的 validator 进行参数校验
2. **错误处理**：统一错误码和错误消息格式
3. **日志记录**：请求入口/出口记录结构化日志
4. **响应格式**：统一 JSON 响应格式，包含 code/message/data
5. **软删除**：所有删除操作为软删除，保留 deleted_at 字段
6. **分页查询**：统一使用 page 和 page_size 参数

### 4.2 已知约束

1. MVP 阶段仅支持 API Key 认证，后续对接企业 IAM
2. 任务列表查询性能依赖数据库索引优化
3. 流式日志需要配合 WebSocket 实现
4. Provider 连接测试依赖外部服务可用性

### 4.3 技术决策

| 决策 | 选择 | 理由 |
|------|------|------|
| Web框架 | Gin | 轻量级、性能优异 |
| 参数校验 | validator v10 | Gin 内置支持、声明式规则 |
| 认证方式 | API Key | MVP 阶段简单可用 |
| 数据库 | PostgreSQL + GORM | 成熟稳定、功能完善 |
| 缓存 | Redis | 高性能、支持分布式 |

## 5. Change History

| Date | Version | Issue | PRD Ref | TRD Ref | Changes |
|------|---------|-------|---------|---------|---------|
| 2026-03-23 | v1.0 | - | §3.2, §4.1, §4.2 | §4.1, §7.1, §7.2 | 初始定义：任务/模板管理 API |
| 2026-03-26 | v2.0 | #8, #9, #10, #11, #12 | §3.2, §4.1, §4.2 | §4.1, §7.1, §7.2 | 新增：租户/Provider/Capability/健康检查 API；完善所有 API 的请求/响应示例和业务规则 |
