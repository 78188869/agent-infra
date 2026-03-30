# Issue 37: 本地开发一键启动

> **Status**: closed
> **Created**: 2026-03-29
> **Closed**: 2026-03-30
> **PR**: #52

## Summary
将所有本地化改造集成，提供一键启动的本地开发体验。开发者只需 `make local` 即可启动完整应用（SQLite + miniredis + Docker Runtime），无需安装 MySQL、Redis、K8s。

## Scope
- [x] 更新 config.local.yaml 使用 SQLite 替代 MySQL
- [x] main.go 支持 APP_ENV 自动选择配置文件
- [x] 集成 miniredis 作为本地内存 Redis
- [x] 串联 Scheduler/Executor 初始化到 main.go
- [x] 调用 SeedProviders 填充初始数据
- [x] 添加优雅关停（graceful shutdown）
- [x] 添加 `make local` 命令
- [x] 集成测试验证一键启动
- [x] 文档说明本地开发步骤

## Knowledge References
- `knowledge/executor.md`
- `knowledge/scheduler.md`
- `knowledge/monitoring.md`

## Key Decisions
1. 使用 miniredis（已有测试依赖）替代真实 Redis，零安装成本
2. 使用 SQLite（已有 driver 支持）替代 MySQL
3. Docker Runtime 在本地可选——Docker 不可用时 API 仍可用，任务执行会失败
4. mock service 保留作为数据库完全不可用时的兜底

## Acceptance Criteria
- [x] `make local` 单个命令即可启动完整应用
- [x] 启动后 API 可完成完整的任务生命周期操作
- [x] 无需预先安装 MySQL、Redis、K8s
- [x] 有文档说明本地开发的启动步骤和依赖要求

## Execution Plan
详见 `docs/superpowers/plans/2026-03-29-issue-37-local-one-click-start.md`

## Notes
- 依赖前置 issue 全部完成（配置、SQLite、文件日志、接口抽象、Docker 引擎、镜像构建）
- miniredis 已是 go.mod 的测试依赖（alicebob/miniredis/v2）
