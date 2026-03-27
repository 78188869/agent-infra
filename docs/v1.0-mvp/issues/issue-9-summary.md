# Issue #9: MVP Phase 5 - Provider Management System

## Summary
实现 Provider 管理系统，支持三层作用域（system/tenant/user）、Provider 选择优先级链、连接测试功能。根据 TRD 第4.1.1节，实现多模型/Agent运行时切换（类似 cc switch）。

## Impact
- 新增 Provider CRUD API（7个端点）
- 实现 Provider 选择优先级：任务指定 > 用户默认 > 租户默认 > 系统默认
- 系统预置 Provider：Claude Code、智谱 GLM、DeepSeek
- API Key 安全存储（K8s Secret 引用）

## Status
In Progress

## Related
- **PRD**: US-D06 Provider 选择与配置, US-A04 Provider 管理
- **TRD**: §4.1.1 (Provider 机制), §6.2.9 (Provider 表), §7.1.6 (Provider API)
- **Knowledge**: docs/knowledge/provider.md
- **Plan**: docs/v1.0-mvp/plans/2026-03-24-issue-9-provider-management.md

## Dependencies
| Issue | 状态 | 说明 |
|-------|------|------|
| #5 Backend Core API | ✅ Completed | API 模式已建立 |
| #6 Database Models | ✅ Completed | Provider 模型已定义 |

## Acceptance Criteria
- [ ] 系统预置 Provider 正确加载
- [ ] 三层作用域正确隔离
- [ ] Provider 选择优先级正确
- [ ] API Key 安全存储（加密/Secret 引用）
- [ ] 连接测试功能正常
- [ ] 单元测试覆盖率 > 80%

## Tasks Overview
1. Provider Repository - 数据访问层
2. Provider Service - 业务逻辑层（含选择优先级链、连接测试）
3. Provider Handler - HTTP 处理层
4. Router Integration - 路由集成
5. System Preset Providers - 系统预置数据
6. Migration Update - 数据库迁移
7. Documentation Update - 文档更新

## Resolution
<!-- 完成后填写 -->

## Change History
| 日期 | 变更内容 |
|------|---------|
| 2026-03-24 | 创建 Issue Summary |
| 2026-03-24 | 创建执行计划 |
