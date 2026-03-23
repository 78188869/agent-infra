# Provider Knowledge

> **Last Updated**: 2026-03-23
> **PRD Version**: v0.7-draft
> **TRD Version**: v2.4

## 1. Overview

Provider 模块负责 Agent 运行时配置管理，支持多种模型后端（如 Claude Code、智谱 GLM、DeepSeek 等）。

**模块职责**：
- Provider 注册与配置
- 三层作用域管理（系统/租户/用户）
- 模型映射与配置兼容
- 默认 Provider 选择链

**核心概念**：
- **Provider**: Agent 运行时配置抽象
- **Scope**: 作用域（system/tenant/user）
- **Model Mapping**: 模型名称映射（opus→glm-5 等）

## 2. Product Requirements (from PRD)

### 2.1 用户故事

| 故事ID | 描述 | 验收标准 |
|--------|------|---------|
| US-D06 | Provider 选择与配置 | 创建任务时可下拉选择 Provider，可配置个人 Provider，可设置个人默认 |
| US-A04 | Provider 管理（管理员） | 可配置系统预置 Provider，可设置租户默认 Provider，可测试连接 |

### 2.2 Provider 作用域

| scope | 说明 | 配置者 | 可见范围 |
|-------|------|--------|---------|
| **system** | 系统预置 Provider | admin | 所有用户 |
| **tenant** | 租户级 Provider | admin | 该租户所有用户 |
| **user** | 用户个人 Provider | developer | 仅该用户 |

### 2.3 Provider 选择优先级

```
任务创建时指定 provider_id?
    │
    ├── 是 ──▶ 使用指定的 Provider
    │
    └── 否 ──▶ 用户有默认 Provider?
                    │
                    ├── 是 ──▶ 使用用户默认 Provider
                    │
                    └── 否 ──▶ 租户有默认 Provider?
                                    │
                                    ├── 是 ──▶ 使用租户默认 Provider
                                    │
                                    └── 否 ──▶ 使用系统默认 Provider (Claude Code)
```

**优先级**：`任务指定 > 用户默认 > 租户默认 > 系统默认`

## 3. Technical Design (from TRD)

### 3.1 Provider 类型

| Type | 说明 | 示例 |
|------|------|------|
| claude_code | 官方 Claude Code CLI | Anthropic 官方 |
| anthropic_compatible | Anthropic 兼容 API | 智谱 GLM、DeepSeek |
| openai_compatible | OpenAI 兼容 API | 其他兼容 API |
| custom | 自定义 | 用户自定义配置 |

### 3.2 Provider 配置结构

```go
type Provider struct {
    ID          string            `json:"id"`
    Scope       string            `json:"scope"`        // system, tenant, user
    TenantID    *string           `json:"tenant_id"`
    UserID      *string           `json:"user_id"`
    Name        string            `json:"name"`
    Type        string            `json:"type"`
    Description string            `json:"description"`
    Config      *ProviderConfig   `json:"config"`
    Status      string            `json:"status"`       // active, inactive, deprecated
}

type ProviderConfig struct {
    // API 配置
    APIEndpoint    string            `json:"api_endpoint"`    // ANTHROPIC_BASE_URL
    APIKeyRef      string            `json:"api_key_ref"`     // K8s Secret 引用

    // 模型映射 (cc switch 兼容)
    ModelMapping   *ModelMapping     `json:"model_mapping"`

    // 运行时配置
    RuntimeType    string            `json:"runtime_type"`    // cli, api, sdk
    RuntimeImage   string            `json:"runtime_image"`   // Docker 镜像
    RuntimeCommand []string          `json:"runtime_command"` // 启动命令

    // 环境变量 (cc switch env)
    Env            map[string]string `json:"env"`

    // 权限配置 (cc switch permissions)
    Permissions    *Permissions      `json:"permissions"`

    // 插件配置 (cc switch enabledPlugins)
    EnabledPlugins map[string]bool   `json:"enabled_plugins"`
}

type ModelMapping struct {
    DefaultModel   string `json:"default_model"`
    OpusModel      string `json:"opus_model"`
    SonnetModel    string `json:"sonnet_model"`
    HaikuModel     string `json:"haiku_model"`
    ReasoningModel string `json:"reasoning_model"`
}

type Permissions struct {
    Allow []string `json:"allow"` // 如 "Bash(node:*)"
    Deny  []string `json:"deny"`
}
```

### 3.3 数据库表结构

```sql
CREATE TABLE providers (
    id              VARCHAR(36) PRIMARY KEY,
    scope           ENUM('system', 'tenant', 'user') NOT NULL DEFAULT 'system',
    tenant_id       VARCHAR(36),
    user_id         VARCHAR(36),
    name            VARCHAR(64) NOT NULL,
    type            ENUM('claude_code', 'anthropic_compatible', 'openai_compatible', 'custom') NOT NULL,
    description     TEXT,
    api_endpoint    VARCHAR(512),
    api_key_ref     VARCHAR(256),
    model_mapping   JSON,
    runtime_type    ENUM('cli', 'api', 'sdk') DEFAULT 'cli',
    runtime_image   VARCHAR(256),
    runtime_command JSON,
    env_vars        JSON,
    permissions     JSON,
    enabled_plugins JSON,
    extra_params    JSON,
    status          ENUM('active', 'inactive', 'deprecated') DEFAULT 'active',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMP NULL,

    UNIQUE KEY uk_scope_name (scope, tenant_id, user_id, name),
    INDEX idx_scope_tenant (scope, tenant_id),
    INDEX idx_scope_user (scope, user_id)
);
```

### 3.4 预置 Provider 示例

**智谱 GLM Provider 配置示例**：

```json
{
    "name": "zhipu-glm",
    "type": "anthropic_compatible",
    "api_endpoint": "https://open.bigmodel.cn/api/anthropic",
    "api_key_ref": "zhipu-api-key",
    "model_mapping": {
        "default": "glm-5",
        "opus": "glm-5",
        "sonnet": "glm-4.7",
        "haiku": "glm-4.5-air",
        "reasoning": "glm-5"
    },
    "env_vars": {
        "API_TIMEOUT_MS": "3000000",
        "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1"
    },
    "permissions": {
        "allow": ["Bash(node:*)", "Bash(npm:*)"]
    },
    "enabled_plugins": {
        "context7@claude-plugins-official": true,
        "episodic-memory@superpowers-marketplace": true
    }
}
```

## 4. Implementation Notes

### 4.1 关键实现要点

1. **密钥安全**: API Key 存储在 K8s Secret，通过引用方式使用
2. **配置兼容**: 配置格式兼容 `cc switch` 命令
3. **作用域隔离**: 不同作用域的 Provider 配置相互独立
4. **连接测试**: 提供测试接口验证 Provider 配置是否正确

### 4.2 API 接口

| 方法 | 路径 | 说明 | 角色 |
|------|------|------|------|
| GET | /providers | Provider 列表 | developer, admin |
| GET | /providers/:id | Provider 详情 | developer, admin |
| POST | /providers | 创建 Provider | admin, developer |
| PUT | /providers/:id | 更新 Provider | admin, developer(自己的) |
| DELETE | /providers/:id | 删除 Provider | admin, developer(自己的) |
| POST | /providers/:id/test | 测试连接 | admin, developer |
| PUT | /providers/:id/set-default | 设置默认 | admin(租户), developer(个人) |

### 4.3 技术决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 密钥存储 | K8s Secret | 安全性高，与 K8s 集成 |
| 配置格式 | cc switch 兼容 | 复用现有生态 |
| 作用域设计 | 三层（系统/租户/用户） | 灵活性与隔离性兼顾 |

## 5. Change History

| Date | Version | Issue | PRD Ref | TRD Ref | Changes |
|------|---------|-------|---------|---------|---------|
| 2026-03-23 | v1.0 | - | §4.11 | §4.1.1, §6.2.9 | 初始定义：Provider 管理机制 |
