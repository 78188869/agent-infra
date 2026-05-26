# Execution Plan: Issue #54 - Credential Manager

> **Date**: 2026-05-26
> **Issue**: #54
> **Developer**: Claude + yuanyang

## Overview

实现 AES-256-GCM 加密存储的凭证管理系统，包括 Model、Encryptor、Repository、Service、Handler 和路由注册。

## Implementation Steps

### Step 5.1: Model 层 - Credential 模型

**Files:**
- `internal/model/credential.go` (new)
- `internal/model/credential_test.go` (new)
- `internal/model/base.go` (update AllModels)

**Details:**
- 定义 Credential 结构体：ID (BaseModel), UserID, Type, Encrypted
- Type 取值：`git_token`, `devops_token`
- 唯一索引：user_id + type（upsert 基础）
- 测试：字段定义验证

### Step 5.2: Encryptor 工具 - AES-256-GCM

**Files:**
- `internal/config/encryptor.go` (new)
- `internal/config/encryptor_test.go` (new)

**Details:**
- AESEncryptor struct 持有 32 字节密钥
- Encrypt(plaintext string) → base64(nonce + ciphertext + tag)
- Decrypt(ciphertext string) → plaintext
- NewAESEncryptor(key string) 验证密钥长度
- 测试：加密/解密正确性、密钥缺失处理、密钥长度校验

### Step 5.3: Repository 层 - CredentialRepository

**Files:**
- `internal/repository/credential_repo.go` (new)
- `internal/repository/credential_repo_test.go` (new)

**Details:**
- Interface: Store / GetByUserAndType / ListByUser / Delete
- Upsert 逻辑：先查后更或新建
- 测试：CRUD 操作

### Step 5.4: Service 层 - CredentialService

**Files:**
- `internal/service/credential_service.go` (new)
- `internal/service/credential_service_test.go` (new)

**Details:**
- Interface: Store / Get / Delete / List / BuildSandboxEnv
- Store：加密后入库，upsert 语义
- Get：解密后返回明文
- List：返回凭证类型列表（不含密文）
- Delete：软删除
- BuildSandboxEnv：查询用户所有凭证，解密后返回 map[string]string
  - git_token → GIT_TOKEN
  - devops_token → DEVOPS_TOKEN
- 测试：所有方法 + 密钥缺失处理

### Step 5.5: Handler + Router 层

**Files:**
- `internal/api/handler/credential.go` (new)
- `internal/api/handler/credential_test.go` (new)
- `internal/api/router/router.go` (update)

**Details:**
- POST /api/v1/credentials：存储凭证 {type, value}
- GET /api/v1/credentials：列出已有凭证类型（不含密文）
- DELETE /api/v1/credentials/:type：删除凭证
- 所有路由需 API Key 认证，从 context 获取 userID
- 测试：HTTP 请求验证

### Step 5.6: 配置集成

**Files:**
- `internal/config/config.go` (update: 添加 EncryptionKey 字段)
- `configs/config.yaml` (update: 添加 encryption_key)

### Step 5.7: 注册到 AllModels + Wire

**Files:**
- `internal/model/base.go` (update)
- `cmd/main.go` or wire setup (update: 初始化 encryptor + credentialSvc + credentialHandler)

## Test Strategy

- **单元测试**: 每层独立测试，使用 SQLite 内存数据库
- **加密测试**: 确保加密/解密往返一致
- **边界情况**: 密钥缺失、空值、无效类型
- **覆盖率目标**: > 80%

## Dependencies

- #1 Auth Middleware（已完成，PR #64）
- 无其他外部依赖
