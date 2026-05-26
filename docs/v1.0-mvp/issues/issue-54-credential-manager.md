# Issue #54: Credential Manager 加密存储与沙箱注入

> **Status**: closed
> **Created**: 2026-05-26
> **Closed**: 2026-05-26
> **PR**: #66

## Summary

实现用户第三方凭证（Git Token、DevOps Token）的 AES-256-GCM 加密存储、CRUD API 和沙箱环境变量注入。

## Scope

- [x] Credential 模型定义（user_id + type + encrypted）
- [x] AES-256-GCM 加密/解密工具（Encryptor）
- [x] CredentialRepository（CRUD + upsert）
- [x] CredentialService（Store / Get / Delete / BuildSandboxEnv）
- [x] CredentialHandler（POST / GET / DELETE /credentials）
- [x] 路由注册（api/v1/credentials，需认证）
- [x] 配置文件添加 ENCRYPTION_KEY
- [x] 单元测试覆盖所有层

## Knowledge References

- `knowledge/database.md` - 数据模型、GORM 配置
- `knowledge/executor.md` - 沙箱生命周期管理、环境变量注入
- `knowledge/core-api.md` - 凭证管理 API

## Key Decisions

1. 使用 AES-256-GCM 对称加密，密钥来自配置文件 ENCRYPTION_KEY（32 字节）
2. 加密格式：base64(nonce + ciphertext + tag)
3. 同一用户同一类型凭证覆盖更新（upsert）
4. 凭证不在任何日志中输出
5. ENCRYPTION_KEY 为必填配置，缺失时启动失败

## Execution Plan

详见 `plans/2026-05-26-issue-54-credential-manager.md`
