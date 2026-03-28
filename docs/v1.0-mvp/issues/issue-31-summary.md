# Issue #31: [TASK] 配置体系支持本地开发环境

> **Status**: completed
> **Created**: 2026-03-28

## Summary
添加环境感知配置体系，支持 local/production 环境切换，让本地开发通过单个环境变量即可激活，无需逐项配置。

## Problem
1. 没有统一的环境概念（local/production），各模块默认值混用
2. config.yaml 中 `${VAR:default}` 语法未实现解析
3. 本地开发需要手动设置大量环境变量，启动门槛高
4. 没有环境特定的配置覆盖机制

## Scope
- [x] 统一配置加载器（支持环境变量展开 `${VAR:default}`）
- [x] 环境感知机制（通过 `APP_ENV` 或配置文件区分 local/production）
- [x] 本地开发配置文件 `config.local.yaml`
- [x] 各模块（数据库、Redis、执行器、日志）根据环境选择默认值
- [x] 启动时日志输出当前运行环境

## Knowledge References
- `knowledge/database.md`
- `knowledge/executor.md`
- `knowledge/monitoring.md`

## Key Decisions
1. 使用 `AppConfig` 统一结构体替代各模块分散的配置加载
2. `ExpandEnv()` 通过 regex 实现 `${VAR:default}` 展开，不引入 viper 等外部依赖
3. 环境检测优先级：`APP_ENV` 环境变量 > 配置文件 `env` 字段 > `"production"` 默认值
4. `RedisYAMLConfig` 桥接 YAML 的 host/port 到 RedisConfig 的 Addr 字段
5. `DatabaseConfig` 添加 YAML tags 保持向后兼容

## Execution Plan
详见 `plans/2026-03-28-env-aware-config.md`
