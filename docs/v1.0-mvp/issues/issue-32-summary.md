# Issue #32: 数据库层兼容 SQLite

> **Status**: in_progress
> **Created**: 2026-03-28

## Summary
让应用支持 SQLite 作为数据库后端，消除本地开发对外部数据库服务的依赖。将 Model 层中 MySQL 专有语法（enum 类型、mediumtext、timestamp 精度）替换为跨数据库兼容的类型（varchar、text），通过配置字段 `DB_DRIVER` 切换 MySQL/SQLite。

## Scope
- [x] 添加 Driver 字段到 DatabaseConfig，支持 mysql/sqlite 切换
- [x] 替换所有 Model 中的 MySQL 专有 GORM tags
- [x] 更新 config.yaml 支持 DB_DRIVER 环境变量
- [x] SQLite 集成测试覆盖全部 Model
- [ ] 更新 knowledge/database.md 文档

## Knowledge References
- `knowledge/database.md`

## Key Decisions
1. 使用 `varchar(N)` 替代 `enum(...)` — SQLite 不支持 ENUM 类型，应用层通过常量保证值合法性
2. 使用 `text` 替代 `mediumtext` — SQLite 只有 TEXT 类型
3. 移除 `timestamp(3)` 精度指定 — GORM 对 time.Time 的默认处理跨 DB 兼容
4. 使用 `gorm.io/driver/sqlite` — 已在 go.mod 中，无需引入新依赖

## Execution Plan
详见 `plans/2026-03-28-sqlite-compat.md`
