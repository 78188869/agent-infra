# Execution Plan: Task 模型迁移 + 状态机清理

> **Issue**: #55
> **Date**: 2026-05-25
> **Status**: In Progress

## Overview

扩展 Task 数据模型，新增 DevOps 集成和任务创建所需的字段，清理废弃的状态机状态。

## Implementation Steps

### 1. Task 模型新增字段 (internal/model/task.go)

**新增字段：**
| 字段 | 类型 | GORM Tag | JSON Tag |
|------|------|----------|----------|
| DevopsSubtaskID | string | `gorm:"type:varchar(128);index"` | `devops_subtask_id,omitempty"` |
| Branch | string | `gorm:"type:varchar(256)"` | `branch,omitempty"` |
| Extra | string | `gorm:"type:text"` | `extra,omitempty"` |
| MRUrl | string | `gorm:"type:varchar(512)"` | `mr_url,omitempty"` |
| MRStatus | string | `gorm:"type:varchar(32)"` | `mr_status,omitempty"` |
| SandboxStatus | string | `gorm:"type:varchar(32)"` | `sandbox_status,omitempty"` |

**修改字段：**
- `Description`: `gorm:"type:text"` → `gorm:"type:longtext"`

**注意：** TemplateID 已存在（*string, index），无需重复添加。

### 2. 标记废弃状态常量 (internal/model/task.go)

- `TaskStatusWaitingApproval`: 添加 Deprecated 注释
- `TaskStatusRetrying`: 添加 Deprecated 注释

```go
// Deprecated: Phase 1 does not use this status
TaskStatusWaitingApproval = "waiting_approval"
// Deprecated: Phase 1 does not use this status
TaskStatusRetrying = "retrying"
```

### 3. 清理状态机转换 (internal/service/task_service.go)

- 从 `validStatusTransitions` map 中删除 `TaskStatusWaitingApproval` 和 `TaskStatusRetrying` 的转换规则
- 保留 7 个有效状态的转换

### 4. 更新 executor (internal/executor/task_executor.go)

- `canExecute()`: 移除 `TaskStatusRetrying` 判断
- `HandleTaskEvent()`: 从 validStatuses map 中移除 `TaskStatusWaitingApproval` 和 `TaskStatusRetrying`

### 5. 更新 intervention service (internal/service/intervention_service.go)

- `canPause()`: 移除 `TaskStatusWaitingApproval` 和 `TaskStatusRetrying`
- `canInject()`: 移除 `TaskStatusWaitingApproval`

### 6. 更新测试

- `internal/model/task_test.go`: 更新状态常量测试
- `internal/service/task_service_test.go`: 移除废弃状态的转换测试用例
- `internal/executor/task_executor_test.go`: 移除 `TaskStatusRetrying` 相关测试
- 新增字段测试用例

### 7. 更新 API 文档

- `docs/api/openapi.yaml`: 更新 status enum，移除 `waiting_approval` 和 `retrying`

## Test Plan

1. `go test ./internal/model/...` - 模型测试
2. `go test ./internal/service/...` - 状态转换测试
3. `go test ./internal/executor/...` - 执行器测试
4. `make test && make lint` - 全量验证
5. `go test -cover ./internal/...` - 覆盖率报告
