---
name: TRD v3.0 Modification Design
description: TRD 修改方案：基于两份评审报告的 P1 问题修复设计
type: project
---

# TRD v3.0 修改方案

**日期**：2026-05-14
**范围**：基于 TRD-v2-review.html 和 review-trd-2026-05-14.html 两份评审的 P1 问题修复
**关联**：BRD v1.0 · PRD v2.0 · TRD v3.0 (TRD-v2.html)

---

## 1. 修改概览

### 1.1 变更清单

| 变更 | 类型 | 对应评审项 |
|------|------|-----------|
| 新增 §5 DevOps Integration Design | 新增章节 | P1-1 + P1-2 + P2-2 |
| 原 §5-§10 顺延为 §6-§11 | 重编号 | 结构调整 |
| 原 §4.3 补充子任务接口 | 修改 | P1-2 |
| 原 §7（现 §8）Task 模型新增字段 | 修改 | P1 字段映射 |
| 状态机标注废弃 | 修改 | P2-1 |
| MVP 范围说明 | 新增说明 | 多个 P2/P3 |

### 1.2 不在本次修改范围

- Agent SOP 设计（skill/CLAUDE.md 范畴，非 TRD 责任）
- Scheduler Dispatcher 健壮性改进（P2-3，后续 issue 处理）
- Credential 轮转策略（P3-2，MVP 简化可接受）
- 前端 UI 设计（后补）

---

## 2. 新增 §5 DevOps Integration Design

### 2.1 章节结构

```
5 · DevOps Integration Design
  5.1 DevOps 数据层级与抽象
  5.2 字段映射表
  5.3 任务创建交互流
  5.4 MCP Server 沙箱集成
  5.5 MVP 范围说明
```

### 2.2 §5.1 DevOps 数据层级与抽象

**内容**：

1. **DevOpsProvider 接口设计**（Go interface）：

```go
type DevOpsProvider interface {
    // 需求级操作
    ListRequirements(ctx context.Context, opts ListOptions) ([]Requirement, error)
    GetRequirement(ctx context.Context, id string) (*Requirement, error)

    // 子任务级操作
    ListSubtasks(ctx context.Context, requirementID string) ([]Subtask, error)
    GetSubtask(ctx context.Context, id string) (*Subtask, error)

    // 仓库信息
    GetRepoInfo(ctx context.Context, repoURL string) (*RepoInfo, error)

    // MR 操作（控制面调用）
    CreateMergeRequest(ctx context.Context, req CreateMRRequest) (*MergeRequest, error)
}
```

2. **统一数据结构**（不绑定具体平台字段名）：

```go
type Requirement struct {
    ID          string
    Title       string
    Description string   // 含验收标准
    Priority    string   // P1/P2/P3
    RepoURL     string
    Type        string   // requirement/bug/improvement
    Stage       string   // brd/prd/trd/dev/test/deploy
    ExternalURL string
    Artifacts   []Artifact
    Subtasks    []Subtask
}

type Subtask struct {
    ID          string
    Title       string
    Description string
    Type        string   // frontend/backend/testing/docs/other
    Assignee    string
    Status      string
}
```

3. **适配器模式说明**：
   - MVP 只实现 `JDXingyunProvider`（京东行云）
   - 接口设计基于 v5-app.html 中的 mockWorkItems 数据结构
   - 未来新增平台只需实现 `DevOpsProvider` 接口

### 2.3 §5.2 字段映射表

**内容**：完整的 DevOps → Task 字段映射表

| 来源层级 | DevOps 逻辑字段 | Task 模型字段 | 映射方式 | 可编辑 |
|---------|---------------|-------------|---------|-------|
| 需求 (Requirement) | `repo_url` | `repo_url` | 需求级关联，预填 | ✅ |
| 需求 (Requirement) | `priority` | `priority` | 需求级，预填 | ✅ |
| 子任务 (Subtask) | `title` | `name` | 直接映射，预填 | ✅ |
| 子任务 (Subtask) | `description` | `description` | 含验收标准，预填 | ✅ |
| 子任务 (Subtask) | `id` | `devops_subtask_id` | 唯一关联标识 | ❌ |
| 用户填写 | — | `branch` | 手动填写或选择 | ✅ 必填 |
| 用户填写 | — | `template_id` | 选择预置模板 | ✅ 必填 |
| 用户填写 | — | `capabilities[]` | 选择能力集 | ✅ 可选 |
| 用户填写 | — | `extra` | 附加指令 | ✅ 可选 |
| 系统 | — | `creator_id` | 当前登录用户 | 自动 |

**补充说明**：
- 所有从 DevOps 预填的字段均允许用户在创建时编辑覆盖
- 覆盖后的值直接存入 Task 模型，不影响 DevOps 原始数据
- Template 参数与 Issue 字段的优先级：用户显式编辑 > DevOps 预填 > Template 默认值

### 2.4 §5.3 任务创建交互流

**内容**：

1. **4 步用户操作流程**：

   Step 1: 浏览需求列表
   - 前端 → 后端 `GET /devops/requirements` → DevOps API
   - 展示需求卡片（标题、优先级、当前阶段、关联仓库）

   Step 2: 选择子任务
   - 用户点击需求 → 展开子任务列表
   - 前端 → 后端 `GET /devops/requirements/:id/subtasks` → DevOps API
   - 展示子任务（标题、描述、类型）

   Step 3: 选择模板 + 编辑参数
   - 选择预置模板（可基于子任务 type 推荐）
   - 所有预填字段（name、description、repo_url、priority）均可编辑
   - 填写 branch（必填）
   - 选择能力集（可选）、附加指令（可选）

   Step 4: 确认创建
   - 前端将所有字段（含用户编辑后的覆盖值）发送到后端
   - 后端创建 Task → 入队调度

2. **POST /tasks 请求体定义**：

```json
{
  "devops_subtask_id": "ST-001",      // 必填：关联的 DevOps 子任务 ID
  "template_id": "coding-task",        // 必填：预置模板 ID
  "branch": "main",                    // 必填：目标分支
  "name": "可选，覆盖预填的标题",
  "description": "可选，覆盖预填的描述",
  "repo_url": "可选，覆盖预填的仓库地址",
  "priority": "可选，覆盖预填的优先级",
  "provider_id": "可选，指定 Provider",
  "capabilities": ["可选，能力 ID 列表"],
  "extra": "可选，附加指令"
}
```

后端处理逻辑：
1. 通过 `devops_subtask_id` 调用 DevOps API 获取子任务详情和父级需求信息
2. 用 DevOps 返回值预填所有字段
3. 用户显式传递的字段覆盖预填值
4. Template Resolve → Credential Inject → Schedule

### 2.5 §5.4 MCP Server 沙箱集成

**内容**：

1. **环境变量注入**：

通过 CredentialManager.BuildSandboxEnv() 注入以下环境变量：

| 环境变量 | 来源 | 说明 |
|---------|------|------|
| `MCP_SERVER_URL` | 配置文件 | MCP Server 服务地址 |
| `DEVOPS_TOKEN` | Credential Manager | 用户 DevOps 凭证（AES-256-GCM 解密后注入） |
| `GIT_TOKEN` | Credential Manager | 用户 Git 凭证 |
| `TASK_ID` | 控制面 | 当前 Task ID，用于回调 |

2. **Skill / CLAUDE.md 配置机制**：

- Template 的 `spec.yaml` 中可引用 skill 文件列表
- 用户在创建任务时可编辑附加指令（extra 字段），内容会追加到 Agent 的初始 prompt
- 具体的 Agent SOP（编码→自检→Push→CI/CD→创建 MR）由 skill 或 CLAUDE.md 定义，不是平台的设计范畴

3. **架构说明**：

```
沙箱内:
  Agent (Claude Code CLI)
    ├── 读取 CLAUDE.md / skill 定义 SOP
    ├── 通过 MCP Server 调用 DevOps 工具（CI/CD 状态、创建 MR 等）
    └── 使用 GIT_TOKEN 进行 git 操作

沙箱外:
  MCP Server（独立服务）
    ├── 接收 Agent 的工具调用请求
    ├── 使用 DEVOPS_TOKEN 调用 DevOps API
    └── 返回结果给 Agent
```

### 2.6 §5.5 MVP 范围说明

**内容**：

- 阶段一只对接**一个** DevOps 平台（京东行云）
- DevOpsProvider 接口设计为可替换，但 MVP 不需要多平台适配
- Agent SOP 完全由 skill/CLAUDE.md 定义，平台不干预 Agent 行为
- MCP Server 作为外部独立服务部署，阶段一不内嵌到沙箱

---

## 3. 对现有章节的附带修改

### 3.1 原 §4.3 DevOps Integration Service — 重构接口

将原 `DevOpsClient` 重命名为 `DevOpsProvider`（与新 §5.1 对齐），新增子任务级操作：

```go
// 新增：子任务级操作
ListSubtasks(ctx context.Context, requirementID string) ([]Subtask, error)
GetSubtask(ctx context.Context, id string) (*Subtask, error)
```

新增 API 端点：

```
GET  /api/v1/devops/requirements           → 列表
GET  /api/v1/devops/requirements/:id       → 详情（含 repo_url、priority、artifacts）
GET  /api/v1/devops/requirements/:id/subtasks → 子任务列表
```

### 3.2 原 §7（现 §8）数据模型变更 — Task 模型新增字段

| 新字段 | 类型 | 说明 |
|-------|------|------|
| `devops_subtask_id` | string | 关联的 DevOps 子任务 ID |
| `description` | text | 任务描述（含验收标准），从 DevOps 预填 |
| `branch` | string | 目标分支，用户填写 |
| `extra` | text | 附加指令 |

### 3.3 状态机标注废弃

在 §7（现 §8）数据模型章节中增加说明：

- `waiting_approval` 和 `retrying` 状态在阶段一**废弃**
- 代码层面加 `// Deprecated: Phase 1 does not use this status` 标记
- TaskService 的 `validStatusTransitions` map 中移除这两个状态的转换规则
- BRD/PRD/TRD/code 四者统一为 7 个状态

### 3.4 章节重编号

| 原编号 | 新编号 | 章节名 |
|-------|-------|-------|
| §5 | §6 | 修改模块设计（Template Preset、Metrics、WS Auth） |
| §6 | §7 | API 设计 |
| §7 | §8 | 数据模型变更 |
| §8 | §9 | 部署架构 |
| §9 | §10 | 安全设计 |
| §10 | §11 | 测试策略 |

---

## 4. 设计原则

- **DevOps 抽象**：所有 DevOps 相关设计基于 `DevOpsProvider` 接口，可替换不同平台实现
- **预填可编辑**：所有从 DevOps 预填的字段均允许用户编辑覆盖
- **最小必填**：创建 Task 只需 `devops_subtask_id` + `template_id` + `branch`
- **SOP 外置**：Agent 的 SOP 由 skill/CLAUDE.md 定义，平台只负责注入基础设施
- **MVP 清晰**：明确标注阶段一的边界（单一平台、外部 MCP Server）
