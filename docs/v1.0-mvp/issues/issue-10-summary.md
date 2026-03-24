# Issue #10: MVP Phase 6 - Capability Management System

> **Status**: 🔄 In Progress
> **Created**: 2026-03-24
> **Assignee**: @claude
> **Branch**: `feature/issue-10-capability-management`

## Summary

根据 TRD 第7.1.5节，实现能力注册管理系统，管理工具（tool）、技能（skill）和 Agent 运行时（agent_runtime）三类能力。

## Impact

**涉及模块**:
- `internal/repository/` - 新增 CapabilityRepository
- `internal/service/` - 新增 CapabilityService
- `internal/api/handler/` - 新增 CapabilityHandler
- `internal/api/router/` - 添加能力路由
- `cmd/control-plane/` - 集成能力服务

**影响范围**:
- 能力注册与配置
- 权限级别控制
- 能力状态管理

## Scope

- [x] 能力类型支持 (tool/skill/agent_runtime)
- [ ] CapabilityRepository (数据访问层)
- [ ] CapabilityService (业务逻辑层)
- [ ] CapabilityHandler (HTTP 处理层)
- [ ] API 路由注册
- [ ] 单元测试覆盖 > 80%

## API Endpoints

| 方法 | 路径 | 说明 | 角色 |
|------|------|------|------|
| GET | /api/v1/capabilities | 能力列表 | developer |
| GET | /api/v1/capabilities/:id | 能力详情 | developer |
| POST | /api/v1/capabilities | 注册能力 | admin |
| PUT | /api/v1/capabilities/:id | 更新能力 | admin |
| DELETE | /api/v1/capabilities/:id | 删除能力 | admin |
| POST | /api/v1/capabilities/:id/activate | 激活能力 | admin |
| POST | /api/v1/capabilities/:id/deactivate | 停用能力 | admin |

## Related

- **PRD**: §4.3 能力管理
- **TRD**: `docs/v1.0-mvp/TRD.md` §7.1.5 能力注册表
- **Knowledge**: `docs/knowledge/capability.md`
- **Dependencies**:
  - Issue #5 (Backend Core API) - ✅ Completed
  - Issue #6 (Database Models) - ✅ Completed
- **Plan**: `docs/v1.0-mvp/plans/2026-03-24-issue-10-capability-management.md`

## Key Decisions

1. **Existing Model**: 使用已有的 `internal/model/capability.go` 模型
2. **Permission Levels**: public/restricted/admin_only 三级权限
3. **Global Capabilities**: tenant_id 为 NULL 表示全局能力
4. **Soft Delete**: 使用 GORM 软删除机制

## Verification Criteria

| Criteria | Verification Method |
|----------|---------------------|
| 能力 CRUD API 正常工作 | Unit tests for Create/Get/List/Update/Delete |
| 能力类型正确区分 | Unit tests for tool/skill/agent_runtime types |
| 参数 Schema 验证有效 | Unit tests for JSON schema validation |
| 权限控制正确 | Unit tests for permission level validation |
| 单元测试覆盖率 > 80% | `go test -cover ./internal/...` |

## Resolution

<!-- 解决方案（完成后填写） -->

## Change History

| 日期 | 变更内容 |
|------|---------|
| 2026-03-24 | 创建 Issue Summary 和执行计划 |
| 2026-03-24 | 创建 worktree `feature/issue-10-capability-management` |
