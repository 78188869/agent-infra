# ADR-001: Docker Compose 作为本地容器运行时

## Status

Accepted

## Context

生产环境使用 K8s Job 创建隔离 Pod 执行任务。本地开发没有 K8s 集群，需要一种轻量级容器隔离方案，使本地能完整走通"创建任务 → 容器执行 → 状态监控 → 人工干预 → 任务完成"的全流程。

**核心需求**：
- 任务在容器内隔离执行
- 支持 pause/resume/inject/cancel 干预操作
- 任务间共享工作区（代码检出、执行产物互访）
- 对上层调用方（TaskExecutor）透明，与 K8s 行为一致

**已有抽象**：代码中已定义 `ContainerRuntime` 接口（Create/GetStatus/Delete/GetAddress），`K8sRuntime` 是当前唯一实现。

## Decision

采用 **Docker Compose** 作为本地容器运行时方案：

1. 新增 `DockerRuntime` 实现 `ContainerRuntime` 接口
2. 新增 `ComposeManager` 生成并管理 per-task 的 `docker-compose.yml`
3. 每个任务对应一个 compose stack（cli-runner + wrapper），复用 K8s 的 Sidecar 模式
4. 工作区通过宿主机 bind mount 共享
5. 日志使用文件输出，不启动 log-agent
6. 通过配置 `executor.runtime_type` 选择运行时（docker/k8s）

**不纳入本次范围**：
- interrupt（SIGINT + --resume 机制）—— 留给后续 issue
- Agent SDK 集成重构 wrapper —— 留给后续 issue

## Consequences

### Positive

- 最小改动量：只需实现 `ContainerRuntime` 接口，TaskExecutor 和上层代码不变
- 调试友好：开发者可直接 `docker compose logs` 查看容器输出
- 与 K8s Sidecar 模式一致：cli-runner + wrapper 双容器结构
- compose 文件可版本控制，方便排查问题

### Negative

- 依赖宿主机 Docker CLI 和 Docker Compose
- 资源开销约 1-2GB/任务（相比 K8s Pod 无显著差异）
- inject 机制仍依赖文件轮询（inject.json），不如 Agent SDK 方案优雅

### Neutral

- Docker 和 K8s 运行时通过配置切换，需要分别维护测试

## Alternatives Considered

| 方案 | 优点 | 缺点 | 为何未选择 |
|------|------|------|-----------|
| Docker Go SDK 直接管理容器 | 不依赖外部 CLI，可作为 Go 包集成 | 多容器编排逻辑需自行实现（网络、卷、启动顺序），代码复杂度高，调试困难 | compose 更贴近 K8s Pod 语义 |
| testcontainers-go 等编排库 | 社区维护 | 主要面向测试场景，定制化受限，与 Sidecar 模式不完全匹配 | 过度工程化 |
| minikube / kind | 与生产完全一致 | 资源开销大（4GB+），启动慢，本机资源紧张时不适用 | 资源开销过高 |

## References

- [TRD §5 沙箱执行引擎](../TRD.md)
- [Executor Knowledge](../../knowledge/executor.md)
- [Issue #35: Docker 容器执行引擎](https://github.com/78188869/agent-infra/issues/35)
