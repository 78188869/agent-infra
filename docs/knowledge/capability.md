# Capability Knowledge

> **Last Updated**: 2026-03-23
> **PRD Version**: v0.7-draft
> **TRD Version**: v2.4

## 1. Overview

Capability 模块负责工具、Skills、Agent 运行时等能力的注册与管理。

**模块职责**：
- 能力注册与配置
- 权限级别控制
- 能力授权管理
- 使用统计

**核心概念**：
- **Tool**: 可执行程序或 CLI 工具
- **Skill**: 预定义的技能包
- **Agent Runtime**: Agent 运行时环境
- **Permission Level**: 权限级别（public/restricted/admin_only）

## 2. Product Requirements (from PRD)

### 2.1 用户故事

| 故事ID | 描述 | 验收标准 |
|--------|------|---------|
| US-A03 | 能力注册管理 | 可注册新工具和 Skills，可配置权限和限制，可启用/禁用能力 |

### 2.2 能力类型

| 类型 | 说明 | 管理方式 |
|---------|------|---------|
| 工具 | 可执行程序、CLI 工具 | 注册 + 权限配置 |
| Skills | 预定义的技能包 | 注册 + 参数配置 |
| Agent Runtime | Agent 运行时 | 注册 + 配额管理 |

### 2.3 权限级别

| 级别 | 说明 |
|------|------|
| public | 所有用户可用 |
| restricted | 需要特殊授权 |
| admin_only | 仅管理员可用 |

## 3. Technical Design (from TRD)

### 3.1 数据库表结构

```sql
CREATE TABLE capabilities (
    id              VARCHAR(36) PRIMARY KEY COMMENT '能力ID',
    tenant_id       VARCHAR(36) COMMENT '租户ID(NULL表示全局能力)',

    -- 能力信息
    type            ENUM('tool', 'skill', 'agent_runtime')
                    NOT NULL COMMENT '能力类型',
    name            VARCHAR(64) NOT NULL COMMENT '能力名称',
    description     TEXT COMMENT '能力描述',
    version         VARCHAR(32) DEFAULT '1.0.0' COMMENT '版本',

    -- 配置
    config          JSON COMMENT '能力配置',
    schema          JSON COMMENT '参数Schema',

    -- 权限
    permission_level ENUM('public', 'restricted', 'admin_only')
                    DEFAULT 'public' COMMENT '权限级别',

    -- 状态
    status          ENUM('active', 'inactive') DEFAULT 'active' COMMENT '状态',

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    UNIQUE KEY uk_tenant_type_name (tenant_id, type, name),
    INDEX idx_type_status (type, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='能力注册表';
```

### 3.2 Service 接口

```go
type CapabilityService interface {
    // 能力注册
    RegisterCapability(ctx context.Context, req *RegisterCapabilityRequest) (*Capability, error)
    GetCapability(ctx context.Context, capabilityID string) (*Capability, error)
    ListCapabilities(ctx context.Context, filter *CapabilityFilter) ([]*Capability, error)

    // 能力状态
    ActivateCapability(ctx context.Context, capabilityID string) error
    DeactivateCapability(ctx context.Context, capabilityID string) error

    // 授权管理
    AuthorizeCapability(ctx context.Context, tenantID string, capabilityIDs []string) error
    RevokeCapability(ctx context.Context, tenantID string, capabilityIDs []string) error
    GetAuthorizedCapabilities(ctx context.Context, tenantID string) ([]*Capability, error)

    // 能力校验
    ValidateCapabilityConfig(ctx context.Context, capabilityType string, config map[string]interface{}) error
}
```

### 3.3 API 接口

| 方法 | 路径 | 说明 | 角色 |
|------|------|------|------|
| GET | /capabilities | 能力列表 | developer |
| GET | /capabilities/:id | 能力详情 | developer |
| POST | /capabilities | 注册能力 | admin |
| PUT | /capabilities/:id | 更新能力 | admin |
| DELETE | /capabilities/:id | 删除能力 | admin |

### 3.4 能力配置示例

**工具配置示例**：

```json
{
    "type": "tool",
    "name": "git",
    "description": "Git 版本控制工具",
    "config": {
        "command": "/usr/bin/git",
        "allowed_actions": ["clone", "pull", "push", "commit", "checkout"]
    },
    "permission_level": "public"
}
```

**Skill 配置示例**：

```json
{
    "type": "skill",
    "name": "code-review",
    "description": "代码审查技能",
    "config": {
        "prompt_template": "review-prompt.md",
        "output_format": "markdown"
    },
    "schema": {
        "language": {"type": "string", "required": true},
        "focus_areas": {"type": "array", "items": "string"}
    },
    "permission_level": "restricted"
}
```

## 4. Implementation Notes

### 4.1 关键实现要点

1. **全局能力**: `tenant_id` 为 NULL 表示全局能力，所有租户可用
2. **能力校验**: 注册时验证配置是否符合 Schema
3. **授权检查**: 使用能力前检查租户是否有授权
4. **版本管理**: 同名能力支持多版本共存

### 4.2 能力白名单机制

在任务执行时，通过能力白名单控制 Agent 可使用的工具：

```yaml
# 模板配置中的能力白名单
execution:
  capabilities:
    tools:
      - git
      - node
      - npm
    skills:
      - code-review
```

### 4.3 技术决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 配置存储 | JSON | 灵活、可扩展 |
| 权限控制 | 三级（public/restricted/admin_only） | 平衡安全与易用 |
| 全局能力 | tenant_id=NULL | 简化共享能力管理 |

### 4.4 实际实现架构

> **Implemented in Issue #10** - See source files for details

```
internal/
├── repository/
│   └── capability_repo.go      # CapabilityRepository (GORM)
├── service/
│   └── capability_service.go   # CapabilityService (业务逻辑)
├── api/handler/
│   └── capability.go           # CapabilityHandler (HTTP)
└── model/
    └── capability.go           # Capability model (已存在)
```

### 4.5 API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/capabilities | 注册能力 |
| GET | /api/v1/capabilities | 能力列表 |
| GET | /api/v1/capabilities/:id | 能力详情 |
| PUT | /api/v1/capabilities/:id | 更新能力 |
| DELETE | /api/v1/capabilities/:id | 删除能力 |
| POST | /api/v1/capabilities/:id/activate | 激活能力 |
| POST | /api/v1/capabilities/:id/deactivate | 停用能力 |

## 5. Change History

| Date | Version | Issue | PRD Ref | TRD Ref | Changes |
|------|---------|-------|---------|---------|---------|
| 2026-03-24 | v1.1 | #10 | §4.3 | §4.1, §6.2.8 | 实现能力管理 CRUD API |
| 2026-03-23 | v1.0 | - | §4.3 | §4.1, §6.2.8 | 初始定义：能力注册与管理 |
