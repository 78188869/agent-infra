# Issue Summaries

本目录存储 MVP 版本（v1.0）的 Issue 摘要文件。

## 文件命名规范

```
issue-{number}-summary.md
```

例如：`issue-5-summary.md`

## 模板结构

每个 Issue 摘要文件遵循以下结构：

```markdown
# Issue #N: {Issue Title}

> **Status**: pending | in_progress | completed | closed
> **Created**: YYYY-MM-DD
> **Closed**: YYYY-MM-DD (如果已完成)
> **PR**: #{PR Number} (如果已合并)

## Summary
简短描述 Issue 的目标和范围。

## Scope
- [x] 已完成项 1
- [x] 已完成项 2
- [ ] 待完成项 3

## Knowledge References
- `knowledge/{module}.md`
- `knowledge/{module}.md`

## Key Decisions
1. 关键决策 1
2. 关键决策 2

## Execution Plan
详见 `plans/issue-{N}-execution.md`（已归档）
```

## 生命周期

| 阶段 | 操作 |
|------|------|
| Issue 创建 | 创建 summary 文件，填写基本信息 |
| Issue 进行中 | 更新 scope 和 decisions |
| Issue 完成 | 标记 status，链接 PR |
| Issue 归档 | plans/ 可清理，summary 保留 |

## 现有 Issues

| Issue | Title | Status | Knowledge |
|-------|-------|--------|-----------|
| #5 | MVP Phase 1 - Backend Core API Development | in_progress | core-api, database |
| #6 | MVP Phase 2 - Database Models and Migrations | completed | database |
| #7 | MVP Phase 3 - Task Scheduler Engine | completed | scheduler |
| #8 | MVP Phase 4 - Task Executor Engine | completed | executor |
| #9 | MVP Phase 5 - Provider Management | completed | provider |
| #10 | MVP Phase 6 - Capability Management | completed | capability |
| #11 | MVP Phase 7 - Human Intervention Mechanism | in_progress | intervention |
| #28 | README.md Optimization | completed | agent-md-guide |

> 注：MVP 开发开始后，在此列出所有 Issue 摘要。
