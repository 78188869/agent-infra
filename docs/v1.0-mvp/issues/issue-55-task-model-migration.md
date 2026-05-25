# Issue #55: Task 模型迁移 + 状态机清理

## 元数据

| 字段 | 值 |
|------|-----|
| **Issue** | #55 |
| **标题** | Task 模型迁移 + 状态机清理 |
| **状态** | in_progress |
| **分配** | — |
| **PR** | — |
| **分支** | feature/issue-55 |
| **创建** | 2026-05-25 |
| **关闭** | — |

## Scope

- [ ] Task 模型新增字段：`devops_subtask_id`（string, index）、`branch`（string）、`extra`（text）、`mr_url`（string）、`mr_status`（string）、`sandbox_status`（string）
- [ ] `description` 字段类型改为 `longtext`
- [ ] `waiting_approval` 和 `retrying` 状态标记 Deprecated
- [ ] `validStatusTransitions` 移除这两个状态的转换规则
- [ ] executor 和 intervention service 更新引用
- [ ] 单元测试更新

## Acceptance Criteria

- [ ] Task 模型新增字段通过 GORM AutoMigrate 自动添加（Nullable 或默认值，不破坏现有数据）
- [ ] `waiting_approval` 和 `retrying` 状态加 `// Deprecated: Phase 1 does not use this status` 标记
- [ ] `validStatusTransitions` map 中移除这两个状态的转换规则
- [ ] 统一为 7 个有效状态：pending / scheduled / running / paused / succeeded / failed / cancelled
- [ ] 现有测试通过，数据迁移无破坏性

## Additional Context

- **TRD 参考**: §8 数据模型变更, §8.3 状态机废弃
- **依赖**: 无（独立的数据层变更）
- **预估工期**: 2 天
