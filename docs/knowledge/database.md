# Database Knowledge

> **Last Updated**: 2026-03-23
> **PRD Version**: v0.7-draft
> **TRD Version**: v2.4

## 1. Overview

Database 模块负责数据模型定义、数据库连接管理和数据迁移。

**模块职责**：
- 核心业务实体模型定义
- GORM 配置与连接池管理
- 数据库迁移脚本管理
- 软删除与审计字段支持

**核心概念**：
- **Tenant**: 租户，资源隔离的基本单位
- **User**: 用户，归属于租户
- **Task**: 任务，一次 Agent 执行的抽象单元
- **Template**: 模板，可复用的任务配置
- **Provider**: Agent 运行时配置

## 2. Product Requirements (from PRD)

### 2.1 数据实体关系

```
Tenant（租户）
    │
    ├── User（用户）
    │       │
    │       ├── Task（任务）───┬── ExecutionLog（执行日志）
    │       │                 ├── Output（输出产物）
    │       │                 └── Intervention（干预记录）
    │       │
    │       └── Template（模板）
    │
    ├── Capability（能力）
    │
    └── Provider（运行时配置）
```

### 2.2 数据保留策略

| 数据类型 | 保留策略 | 说明 |
|----------|---------|------|
| 任务元数据 | 永久保留 | 软删除 |
| 执行日志 | SLS 永久，DB 索引 30 天 | 关键事件索引 |
| 输出产物 | OSS 按配置 | 支持用户自定义 |

## 3. Technical Design (from TRD)

### 3.1 技术选型

| 组件 | 选型 | 版本 | 说明 |
|------|------|------|------|
| 数据库 | OceanBase | 4.x | MySQL 兼容、分布式能力 |
| ORM | GORM | 1.25.x | Go 主流 ORM 库 |
| 连接池 | GORM 内置 | - | 配置化连接池参数 |

### 3.2 核心表结构

#### 3.2.1 tenants（租户表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(36) | 租户 ID (UUID) |
| name | VARCHAR(128) | 租户名称 |
| quota_cpu | INT | CPU 核心数上限 |
| quota_memory | BIGINT | 内存上限 (GB) |
| quota_concurrency | INT | 最大并发任务数 |
| quota_daily_tasks | INT | 每日任务数上限 |
| status | ENUM | active/suspended |

#### 3.2.2 users（用户表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(36) | 用户 ID (UUID) |
| tenant_id | VARCHAR(36) | 租户 ID (FK) |
| username | VARCHAR(64) | 用户名 |
| role | ENUM | developer/admin/operator/reviewer |
| status | ENUM | active/disabled |
| deleted_at | TIMESTAMP | 软删除时间 |

#### 3.2.3 tasks（任务表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(36) | 任务 ID (UUID) |
| tenant_id | VARCHAR(36) | 租户 ID (FK) |
| template_id | VARCHAR(36) | 模板 ID (FK) |
| creator_id | VARCHAR(36) | 创建者 ID (FK) |
| provider_id | VARCHAR(36) | Provider ID |
| status | ENUM | pending/scheduled/running/paused/waiting_approval/retrying/succeeded/failed/cancelled |
| priority | ENUM | high/normal/low |
| params | JSON | 运行时参数 |
| resolved_spec | MEDIUMTEXT | 参数替换后的完整配置 |
| pod_name | VARCHAR(128) | K8s Pod 名称 |
| result | JSON | 执行结果 |
| error_message | TEXT | 错误信息 |
| error_code | VARCHAR(32) | 错误码 |
| metrics | JSON | 执行指标 |
| deleted_at | TIMESTAMP | 软删除时间 |

#### 3.2.4 templates（模板表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(36) | 模板 ID (UUID) |
| tenant_id | VARCHAR(36) | 租户 ID (FK) |
| name | VARCHAR(128) | 模板名称 |
| version | VARCHAR(32) | 版本号 |
| spec | MEDIUMTEXT | 模板 YAML 定义 |
| scene_type | ENUM | coding/ops/analysis/content/custom |
| status | ENUM | draft/published/deprecated |
| provider_id | VARCHAR(36) | 指定 Provider |
| deleted_at | TIMESTAMP | 软删除时间 |

#### 3.2.5 providers（运行时配置表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(36) | Provider ID |
| scope | ENUM | system/tenant/user |
| tenant_id | VARCHAR(36) | 租户 ID (scope=tenant/user) |
| user_id | VARCHAR(36) | 用户 ID (scope=user) |
| name | VARCHAR(64) | Provider 名称 |
| type | ENUM | claude_code/anthropic_compatible/openai_compatible/custom |
| api_endpoint | VARCHAR(512) | API 端点 |
| api_key_ref | VARCHAR(256) | API 密钥引用 (K8s Secret) |
| model_mapping | JSON | 模型映射配置 |
| runtime_type | ENUM | cli/api/sdk |
| runtime_image | VARCHAR(256) | Docker 镜像 |
| env_vars | JSON | 环境变量 |
| permissions | JSON | 权限配置 |
| enabled_plugins | JSON | 启用的插件 |

#### 3.2.6 execution_logs（执行日志表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT AUTO | 日志 ID |
| task_id | VARCHAR(36) | 任务 ID (FK) |
| event_type | ENUM | status_change/tool_call/tool_result/llm_input/llm_output/error/heartbeat/intervention/metric/checkpoint |
| event_name | VARCHAR(64) | 事件名称 |
| content | JSON | 事件内容 |
| timestamp | TIMESTAMP(3) | 事件时间 |

#### 3.2.7 interventions（人工干预表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(36) | 干预 ID |
| task_id | VARCHAR(36) | 任务 ID (FK) |
| operator_id | VARCHAR(36) | 操作者 ID (FK) |
| action | ENUM | pause/resume/cancel/inject/modify |
| content | JSON | 干预内容 |
| result | JSON | 执行结果 |
| status | ENUM | pending/applied/failed |

### 3.3 GORM 模型定义

```go
// internal/model/task.go
type Task struct {
    ID            string          `gorm:"primaryKey;type:varchar(36)"`
    TenantID      string          `gorm:"type:varchar(36);index:idx_tenant_status"`
    TemplateID    *string         `gorm:"type:varchar(36);index"`
    CreatorID     string          `gorm:"type:varchar(36);index"`
    ParentTaskID  *string         `gorm:"type:varchar(36)"`
    ProviderID    string          `gorm:"type:varchar(36);not null"`

    Name          string          `gorm:"type:varchar(256)"`
    Description   string          `gorm:"type:text"`

    Status        TaskStatus      `gorm:"type:enum('pending','scheduled','running','paused','waiting_approval','retrying','succeeded','failed','cancelled');default:'pending';index:idx_tenant_status"`
    Progress      int             `gorm:"default:0"`
    CurrentStage  string          `gorm:"type:varchar(64)"`

    Params        datatypes.JSON  `gorm:"type:json"`
    ResolvedSpec  string          `gorm:"type:mediumtext"`
    Priority      Priority        `gorm:"type:enum('high','normal','low');default:'normal'"`

    PodName       string          `gorm:"type:varchar(128)"`
    SandboxID     string          `gorm:"type:varchar(64)"`
    RetryCount    int             `gorm:"default:0"`
    MaxRetries    int             `gorm:"default:3"`

    Result        datatypes.JSON  `gorm:"type:json"`
    ErrorMessage  string          `gorm:"type:text"`
    ErrorCode     string          `gorm:"type:varchar(32)"`
    Metrics       datatypes.JSON  `gorm:"type:json"`

    ScheduledAt   *time.Time      `gorm:"index"`
    StartedAt     *time.Time
    FinishedAt    *time.Time
    CreatedAt     time.Time       `gorm:"autoCreateTime"`
    UpdatedAt     time.Time       `gorm:"autoUpdateTime"`
    DeletedAt     gorm.DeletedAt  `gorm:"index"`

    // 关联
    Tenant        *Tenant         `gorm:"foreignKey:TenantID"`
    Template      *Template       `gorm:"foreignKey:TemplateID"`
    Creator       *User           `gorm:"foreignKey:CreatorID"`
}
```

### 3.4 索引设计

| 表 | 索引名 | 字段 | 说明 |
|------|--------|------|------|
| tasks | idx_tenant_status | tenant_id, status | 租户任务查询 |
| tasks | idx_status_created | status, created_at | 状态+时间查询 |
| tasks | idx_scheduled_at | scheduled_at | 调度时间查询 |
| execution_logs | idx_task_time | task_id, timestamp | 日志时间查询 |
| execution_logs | idx_task_event | task_id, event_type | 事件类型查询 |

## 4. Implementation Notes

### 4.1 关键实现要点

1. **软删除**：所有核心实体使用 `deleted_at` 字段实现软删除
2. **UUID 生成**：使用 `github.com/google/uuid` 生成 UUID
3. **JSON 字段**：使用 `gorm.io/datatypes` 处理 JSON 字段
4. **时间精度**：执行日志使用毫秒级时间戳 `TIMESTAMP(3)`

### 4.2 数据库连接配置

```yaml
database:
  driver: mysql
  dsn: "user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600s
```

### 4.3 迁移策略

- 使用 GORM AutoMigrate 进行开发环境迁移
- 生产环境使用独立的迁移脚本
- 向后兼容的 Schema 变更

## 5. Change History

| Date | Version | Issue | PRD Ref | TRD Ref | Changes |
|------|---------|-------|---------|---------|---------|
| 2026-03-23 | v1.0 | - | §5.1 | §6.1, §6.2, §6.3 | 初始定义：数据模型与表结构 |
