# Intervention Knowledge

> **Last Updated**: 2026-03-23
> **PRD Version**: v0.7-draft
> **TRD Version**: v2.4

## 1. Overview

Intervention 模块负责任务执行过程中的人工干预机制。

**模块职责**：
- 暂停/恢复/取消任务
- 指令注入
- 审批断点（v1.1+）
- 干预记录管理

**核心概念**：
- **Intervention**: 人工干预记录
- **Action**: 干预动作类型（pause/resume/cancel/inject/modify）
- **Checkpoint**: 审批断点（v1.1+）

## 2. Product Requirements (from PRD)

### 2.1 用户故事

| 故事ID | 描述 | 验收标准 |
|--------|------|---------|
| US-D03 | 人工干预任务 | 支持暂停/恢复/取消任务，支持注入指令，支持修改任务参数，记录干预历史 |
| US-O02 | 异常处理 | 支持暂停、恢复、取消任务，支持注入指令继续执行，支持审批断点流程 |

### 2.2 干预类型

| 类型 | 触发条件 | 用户操作 | 后续动作 | 场景追溯 |
|------|----------|----------|----------|---------|
| 审批断点 | 到达预设断点 | 批准/拒绝 | 继续/终止任务 | 2.4 交付验收 |
| 实时接管 | 用户主动暂停 | 注入指令/修改配置 | 继续执行 | 2.3 反馈闭环 |
| 异常介入 | 失败/超时/异常 | 诊断/修复/放弃 | 重试/恢复/终止 | 2.3 反馈闭环 |
| MR 审核 | Agent 提交 MR | 审核代码/批准/拒绝 | 合并/驳回 | 2.4 交付验收 |

### 2.3 干预操作

| 操作 | 说明 |
|------|------|
| 暂停 | 暂停正在执行的任务 |
| 恢复 | 恢复暂停的任务继续执行 |
| 取消 | 终止任务执行 |
| 重试 | 重新执行失败的任务 |
| 注入指令 | 向运行中的任务注入新指令 |
| 修改配置 | 动态调整任务参数 |

## 3. Technical Design (from TRD)

### 3.1 数据库表结构

```sql
CREATE TABLE interventions (
    id              VARCHAR(36) PRIMARY KEY COMMENT '干预ID',
    task_id         VARCHAR(36) NOT NULL COMMENT '任务ID',
    operator_id     VARCHAR(36) NOT NULL COMMENT '操作者ID',

    -- 干预信息
    action          ENUM('pause', 'resume', 'cancel', 'inject', 'modify')
                    NOT NULL COMMENT '干预动作',
    content         JSON COMMENT '干预内容',
    reason          VARCHAR(512) COMMENT '干预原因',

    -- 结果
    result          JSON COMMENT '执行结果',
    status          ENUM('pending', 'applied', 'failed')
                    DEFAULT 'pending' COMMENT '状态',

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',

    INDEX idx_task (task_id),
    INDEX idx_operator (operator_id),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (task_id) REFERENCES tasks(id),
    FOREIGN KEY (operator_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='人工干预记录表';
```

### 3.2 干预流程

```
┌─────────────────────────────────────────────────────────────────────┐
│                        人工干预数据流                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  1. 干预请求                                                        │
│     用户面板 ──▶ POST /api/v1/tasks/:id/inject                       │
│             ──▶ MSE网关 ──▶ 控制面                                  │
│                                                                      │
│  2. 干预处理                                                        │
│     TaskSvc ──▶ 验证任务状态                                        │
│             ──▶ 记录干预到数据库                                    │
│             ──▶ 获取沙箱Job Pod地址                                │
│             ──▶ 转发干预指令到沙箱                                  │
│                                                                      │
│  3. 沙箱执行                                                        │
│     Wrapper ──▶ 接收干预指令                                        │
│             ──▶ 注入到CLI                                           │
│             ──▶ 继续执行                                            │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.3 API 接口

| 方法 | 路径 | 说明 | 角色 |
|------|------|------|------|
| POST | /tasks/:id/pause | 暂停任务 | developer, operator |
| POST | /tasks/:id/resume | 恢复任务 | developer, operator |
| POST | /tasks/:id/cancel | 取消任务 | developer, operator |
| POST | /tasks/:id/inject | 注入指令 | developer, operator |
| POST | /tasks/:id/retry | 重试任务 | developer, operator |
| GET | /tasks/:id/interventions | 干预历史 | developer, operator |

### 3.4 注入机制（MVP）

MVP 阶段采用 **HTTP 轮询方式** 实现指令注入：

```
┌─────────────────────────────────────────────────────────────────────┐
│                        指令注入机制                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  1. wrapper 通过 HTTP 接收干预指令                                   │
│  2. wrapper 将指令写入 inject.json                                   │
│  3. cli-runner 的启动脚本轮询读取 inject.json (每 1 秒)              │
│  4. 发现新指令后注入到 CLI 的 stdin                                  │
│                                                                      │
│  后续版本可考虑使用 inotify 实现实时监听                             │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.5 进程信号通信

通过共享 PID Namespace 实现进程控制：

| 信号 | 发送方 | 接收方 | 说明 |
|------|--------|--------|------|
| SIGSTOP | wrapper | cli-runner | 暂停 CLI 执行 |
| SIGCONT | wrapper | cli-runner | 恢复 CLI 执行 |
| SIGTERM | wrapper | cli-runner | 优雅终止 CLI |

## 4. Implementation Notes

### 4.1 关键实现要点

1. **状态校验**: 干预前必须校验任务当前状态是否允许该操作
2. **幂等性**: 同一干预操作可重复调用，结果一致
3. **超时处理**: 干预指令发送后设置超时，避免无限等待
4. **审计日志**: 所有干预操作记录到审计日志

### 4.2 状态转换规则

| 当前状态 | pause | resume | cancel | inject |
|---------|-------|--------|--------|--------|
| pending | ✗ | ✗ | ✓ | ✗ |
| scheduled | ✗ | ✗ | ✓ | ✗ |
| running | ✓ | ✗ | ✓ | ✓ |
| paused | ✗ | ✓ | ✓ | ✗ |

### 4.3 技术决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 注入方式 | HTTP 轮询 | 实现简单，MVP 阶段够用 |
| 进程控制 | 信号 + 共享 PID | K8s 原生支持 |
| 记录存储 | 数据库 + SLS | 持久化 + 日志分析 |

## 5. Change History

| Date | Version | Issue | PRD Ref | TRD Ref | Changes |
|------|---------|-------|---------|---------|---------|
| 2026-03-23 | v1.0 | - | §4.5 | §4.1, §6.2.7 | 初始定义：人工干预机制 |
