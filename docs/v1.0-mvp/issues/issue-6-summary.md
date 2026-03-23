# Issue #6: MVP Phase 2 - Database Models and Migrations

> **Status**: completed
> **Created**: 2026-03-23
> **Closed**: 2026-03-23
> **PR**: #14

## Summary

根据 TRD 第6节数据模型设计，实现 GORM 模型定义和数据库迁移。

实现所有核心数据模型的定义和数据库迁移脚本，支持多租户、任务管理、模板配置、Provider 管理等核心功能。

## Scope

- [x] Tenant 模型 - 租户资源隔离与配额管理
- [x] User 模型 - 用户管理与角色权限
- [x] APIKey 模型 - API 密钥认证（SHA256 哈希存储）
- [x] Template 模型 - 任务模板与版本控制
- [x] Task 模型 - 任务执行状态机（9 种状态）
- [x] ExecutionLog 模型 - 执行日志索引（10 种事件类型）
- [x] Intervention 模型 - 人工干预记录（5 种操作）
- [x] Capability 模型 - 能力注册（工具/技能/运行时）
- [x] Provider 模型 - Agent 运行时配置（cc switch 兼容）
- [x] UserProviderDefault 模型 - 用户默认 Provider 设置
- [x] 迁移管理器 - AutoMigrate 支持
- [x] 单元测试覆盖

## Knowledge References

- `docs/knowledge/database.md` - 数据库模型与存储
- `docs/v1.0-mvp/TRD.md` §6 - 数据模型设计
- `docs/v1.0-mvp/TRD.md` §6.3 - 任务状态转换规则

## Key Decisions

1. **UUID 主键**: 使用 `github.com/google/uuid` 生成 UUID
2. **软删除**: 所有核心实体使用 `deleted_at` 字段实现软删除
3. **JSON 字段**: 使用 `gorm.io/datatypes` 处理灵活配置
4. **字符串类型 ID**: 模型使用 `string` 类型 ID（非 uuid.UUID）以便于 JSON 序列化
5. **Provider 作用域**: 支持 system/tenant/user 三层作用域
6. **任务状态机**: 提供 `CanPause()`、`CanResume()`、`CanCancel()` 等状态校验方法

## Execution Plan

详见 `docs/v1.0-mvp/plans/2026-03-23-issue-6-database-models.md`

## Verification Criteria

| Criteria | Verification Method |
|----------|---------------------|
| 所有模型定义完成 | `go build ./...` |
| 单元测试通过 | `go test ./internal/model/... -v` |
| 模型与 TRD 一致 | Code review |
| 迁移管理器可用 | `go test ./internal/migration/...` |

## Change History

| 日期 | 变更内容 |
|------|---------|
| 2026-03-23 | 创建 Issue Summary |
| 2026-03-23 | 完成 PR #14 合并 |
