# Core API Knowledge

> **Last Updated**: 2026-03-23
> **PRD Version**: v0.7-draft
> **TRD Version**: v2.4

## 1. Overview

Core API 模块负责任务管理、模板管理、用户认证等核心 HTTP API 的设计与实现。

**模块职责**：
- 任务 CRUD 操作与状态管理
- 模板 CRUD 操作与版本管理
- API Key 认证与权限校验
- 请求参数校验与响应序列化

**核心概念**：
- **Task**: 一次 Agent 执行的抽象单元，是平台的核心业务对象
- **Template**: 可复用的任务配置，定义任务类型和执行策略
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

### 2.2 业务规则

1. **任务创建**：
   - 必须选择有效的模板
   - 参数必须通过模板定义的校验规则
   - 必须通过配额检查（租户级）

2. **任务状态转换**：
   - 只能从当前状态转换到合法的目标状态
   - 终态任务（succeeded/failed/cancelled）不可再变更

3. **模板发布**：
   - 只有 draft 状态的模板可以发布
   - 发布后的模板不可删除，只能废弃

## 3. Technical Design (from TRD)

### 3.1 架构设计

```
control-plane/
├── internal/
│   ├── api/
│   │   ├── handler/
│   │   │   ├── task.go        # 任务 HTTP 处理器
│   │   │   ├── template.go    # 模板 HTTP 处理器
│   │   │   ├── tenant.go      # 租户 HTTP 处理器
│   │   │   └── user.go        # 用户 HTTP 处理器
│   │   ├── middleware/
│   │   │   ├── auth.go        # API Key 认证
│   │   │   ├── ratelimit.go   # 限流
│   │   │   └── logger.go      # 日志
│   │   └── router.go          # 路由注册
│   └── service/
│       ├── task_service.go    # 任务业务逻辑
│       └── template_service.go # 模板业务逻辑
```

### 3.2 API 规范

#### 任务管理 `/api/v1/tasks`

| 方法 | 路径 | 说明 | 角色 |
|------|------|------|------|
| POST | /tasks | 创建任务 | developer |
| GET | /tasks | 任务列表 | developer |
| GET | /tasks/:id | 任务详情 | developer |
| POST | /tasks/:id/pause | 暂停任务 | developer, operator |
| POST | /tasks/:id/resume | 恢复任务 | developer, operator |
| POST | /tasks/:id/cancel | 取消任务 | developer, operator |
| POST | /tasks/:id/inject | 注入指令 | developer, operator |
| POST | /tasks/:id/retry | 重试任务 | developer, operator |
| GET | /tasks/:id/logs | 获取日志 | developer, operator |
| GET | /tasks/:id/queue-position | 查询排队位置 | developer |
| GET | /tasks/:id/metrics | 获取执行指标 | developer |

#### 模板管理 `/api/v1/templates`

| 方法 | 路径 | 说明 | 角色 |
|------|------|------|------|
| GET | /templates | 模板列表 | developer |
| GET | /templates/:id | 模板详情 | developer |
| POST | /templates | 创建模板 | admin |
| PUT | /templates/:id | 更新模板 | admin |
| DELETE | /templates/:id | 删除模板 | admin |
| POST | /templates/:id/publish | 发布模板 | admin |
| POST | /templates/:id/deprecate | 废弃模板 | admin |
| GET | /templates/:id/versions | 版本历史 | admin |
| POST | /templates/:id/validate | 校验模板 | admin |

### 3.3 Service 接口定义

```go
// TaskService 接口
type TaskService interface {
    CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error)
    GetTask(ctx context.Context, taskID string) (*Task, error)
    ListTasks(ctx context.Context, filter *TaskFilter) ([]*Task, int64, error)
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
```

### 3.4 认证机制

**API Key 认证流程**：
1. 请求头携带 `Authorization: Bearer <api_key>`
2. Auth 中间件提取并验证 API Key
3. 计算密钥 SHA256 哈希值，查询数据库匹配
4. 验证通过后注入用户上下文到请求

**API Key 存储规范**：
- 只存储 SHA256 哈希值，不存储明文
- 存储前 8 位前缀用于识别
- 支持过期时间和撤销操作

## 4. Implementation Notes

### 4.1 关键实现要点

1. **请求校验**：使用 Gin 框架的 validator 进行参数校验
2. **错误处理**：统一错误码和错误消息格式
3. **日志记录**：请求入口/出口记录结构化日志
4. **响应格式**：统一 JSON 响应格式，包含 code/message/data

### 4.2 已知约束

1. MVP 阶段仅支持 API Key 认证，后续对接企业 IAM
2. 任务列表查询性能依赖数据库索引优化
3. 流式日志需要配合 WebSocket 实现

### 4.3 技术决策

| 决策 | 选择 | 理由 |
|------|------|------|
| Web框架 | Gin | 轻量级、性能优异 |
| 参数校验 | validator v10 | Gin 内置支持、声明式规则 |
| 认证方式 | API Key | MVP 阶段简单可用 |

## 5. Change History

| Date | Version | Issue | PRD Ref | TRD Ref | Changes |
|------|---------|-------|---------|---------|---------|
| 2026-03-23 | v1.0 | - | §3.2, §4.1, §4.2 | §4.1, §7.1, §7.2 | 初始定义：任务/模板管理 API |
