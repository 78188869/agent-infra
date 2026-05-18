# Issue #53: Auth Middleware + API Key 管理

## 元数据

| 字段 | 值 |
|------|-----|
| **Issue** | #53 |
| **标题** | Auth Middleware + API Key 管理 |
| **状态** | in_progress |
| **分配** | — |
| **PR** | — |
| **分支** | feature/issue-53 |
| **创建** | 2026-05-18 |
| **关闭** | — |

## Scope

- [x] API Key 认证中间件（SHA256 hash 存储）
- [x] API Key CRUD 接口（创建/列出/撤销）
- [x] Authenticator 接口抽象（预留 SSO）
- [x] 路由保护（/api/v1/* 受保护，/health 和 /internal 不受保护）
- [x] 单元测试覆盖

## Acceptance Criteria

- [x] API Key 通过 SHA256(prefix + secret) 双部分存储
- [x] Auth Middleware 从 Header 提取 key，校验后注入 user_id 和 tenant_id
- [x] 所有 /api/v1/* 路由受保护
- [x] POST/GET/DELETE /api/v1/api-keys 完整 CRUD
- [x] Authenticator 接口预留 SSO 扩展
- [x] 单元测试覆盖

## 实现概要

| 组件 | 文件 | 说明 |
|------|------|------|
| APIKeyRepository | `internal/repository/api_key_repo.go` | CRUD + hash 查询 + 使用量追踪 |
| UserRepository | `internal/repository/user_repo.go` | 用户查询（middleware 上下文注入） |
| APIKeyService | `internal/service/api_key_service.go` | 创建/验证/撤销/列表业务逻辑 |
| UserService | `internal/service/user_service.go` | 用户查询（禁用检查） |
| Auth Middleware | `internal/api/middleware/auth.go` | Bearer token 提取 → 验证 → 上下文注入 |
| APIKeyHandler | `internal/api/handler/api_key.go` | POST/GET/DELETE /api/v1/api-keys |
| Router | `internal/api/router/router.go` | auth middleware 应用于 /api/v1/* |

## 测试统计

| 包 | 测试数 | 状态 |
|---|--------|------|
| repository | 17 | PASS |
| service | 17 | PASS |
| middleware | 11 | PASS |
| handler | 6 | PASS |
| router | 13 | PASS |

## 知识模块

- `knowledge/core-api.md`
- `knowledge/database.md`

## TRD 参考

- §4.1 Auth Middleware
- §7.1 api-keys 端点
- §10 安全设计
