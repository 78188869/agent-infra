# agent.md 编写规范

> 本文档指导团队成员编写和维护 `agent.md`，确保 AI 编码代理高效、准确地理解项目规则。

---

## 1. 什么是 agent.md

`agent.md` 是 AI 编码代理（Claude Code、OpenAI Codex、Cursor 等）的操作手册。它告诉代理**怎么干活、遵守什么规则、不该做什么**。

### 与 README.md 的分工

| 文件 | 读者 | 核心定位 |
|------|------|---------|
| README.md | 人类开发者 | **项目名片** — 是什么、为什么、怎么跑起来 |
| agent.md | AI 编码代理 | **操作手册** — 怎么干活、遵守什么规则 |

**每块信息只在一个地方定义完整版，另一处用链接引用。**

---

## 2. 核心原则：规则 > 事实

```
放什么                              不放什么
─────────────────────────           ─────────────────────────
✓ 编码规范 + 项目特有规则            ✗ 外部链接/参考文档
✓ 架构分层规则 + 禁止事项            ✗ Changelog（git log 是权威）
✓ Tech Stack（技术选型锚点）         ✗ 详细目录树（子目录展开）
✓ Project Structure（第一层）        ✗ Issue/Plan 完整模板
✓ 开发工作流                        ✗ 快速参考表（Redis keys 等）
✓ 知识模块索引                      ✗ 安装步骤/环境搭建
✓ 常用命令                          ✗ Self-evident 的常识
```

判断标准：**agent 能否从代码中推断？** 能推断的不放，推断不了的放。

---

## 3. 放什么：逐项说明

### 必须放

| 内容 | 理由 | 来源 |
|------|------|------|
| **Tech Stack** | agent 生成代码需要匹配选型（Go 版本、框架、UI 库），从 go.mod 推断不够快且可能推断错 | Anthropic + OpenAI |
| **Project Structure** | agent 没有直觉，需要明确知道代码在哪、职责是什么。比人类更需要这份导航图 | OpenAI + Google |
| **Architecture Constraints** | 防止 agent 破坏分层边界，这是它无法从代码推断的约束 | Anthropic + Cursor |
| **Coding Standards** | 项目特有的编码规则（不是通用规范的重复） | 所有公司一致推荐 |
| **Development Workflow** | agent 的操作步骤（Issue → Plan → 开发 → 测试 → PR） | OpenAI |
| **Commands** | 构建、测试、部署命令 — agent 无法猜测的项目特定命令 | Anthropic + OpenAI |

### 不放

| 内容 | 理由 | 替代方案 |
|------|------|---------|
| 外部链接 | agent 不读外部链接，主流规范已内化在训练数据中 | 在编码规范中只写项目特有规则 |
| Changelog | git log 是权威来源，手动维护容易过时 | 不需要 |
| 详细目录树 | 会随代码膨胀，违反稳定性原则 | 只列第一层，深层用 `ls` 发现 |
| 完整模板 | 占空间且已在别处定义 | 引用 issues/README.md 等文件 |
| 参考数据 | 静态数据占空间但不是每次都用 | 外置到 knowledge/quick-reference.md |
| 安装步骤 | 人类操作用，agent 不需要 | 放 README.md |

---

## 4. Token Cache 稳定性原则

agent.md 在每次对话中全量加载到上下文窗口，因此：

### 为什么重要

- **每次对话都消耗 token** — 文件越大，每轮对话成本越高
- **注意力稀释** — Anthropic 官方警告：过长的指令导致 agent 忽略重要规则
- **Cache 失效** — 文件频繁变化会导致 prompt cache 失效，增加延迟和成本

### 稳定性策略

1. **不放会频繁变化的内容** — 目录只列第一层，新增文件在已有目录下不需要更新
2. **不放可推断的事实** — 让 agent 自己去 `find`、`grep`、`cat`
3. **只放稳定的规则和导航** — 架构约束、编码规范、模块索引

### 目标行数

| 来源 | 推荐行数 |
|------|---------|
| Anthropic 官方 | < 500 行 |
| 社区最佳实践 | 150-200 行 |
| **本项目建议** | **150-200 行** |

> 判断方法：对每一行问"去掉这行，agent 会不会犯错？"如果不会，就删掉。
> — Anthropic 官方建议

---

## 5. 目录结构的写法

### 规则：只到第一层

```
✗ 过于详细（会膨胀）
├── internal/
│   ├── api/
│   │   ├── handler/         # HTTP handlers
│   │   ├── middleware/      # Auth, rate limit
│   │   ├── response/        # Response format
│   │   └── router/          # Routes
│   ├── service/             # Business logic
│   └── ...

✓ 恰到好处（第一层 + 职责注释）
├── internal/
│   ├── api/                 # HTTP 层：handler、middleware、router、response
│   ├── service/             # 业务逻辑核心，事务边界
│   ├── repository/          # 数据访问层
│   └── model/               # GORM 模型定义
```

**为什么**：新增文件在已有目录下不需要更新 agent.md，只有新增顶层目录才需要。

---

## 6. 消除重复

同一信息在 agent.md 内部或与 README.md 之间不要出现两次。

| 常见重复 | 解决方式 |
|---------|---------|
| 目录规则 vs Architecture Constraints | 合并到 Architecture Constraints |
| docs 目录树 vs 知识模块索引 | 保留索引表，简化树 |
| README 的 Workflow vs agent.md 的 Workflow | agent.md 放完整版，README 只放链接 |
| 通用规范 vs 外部链接 | 只写项目特有规则，不说"参考某链接" |

---

## 7. 压缩手段

| 手段 | 效果 | 示例 |
|------|------|------|
| 表格替代 ASCII 流程图 | 信息密度更高 | 14 步工作流：ASCII → 表格 |
| 引用替代内联 | 避免重复 | 模板引用 issues/README.md |
| 一句话替代多行描述 | 去掉抽象描述 | 知识库特点 4 行 → 1 句话 |
| 外置参考数据 | 主文件不膨胀 | Quick Lookup → knowledge/ |
| 合并重复内容 | 消除冗余 | 目录规则 + Architecture 表合并 |

---

## 8. 推荐章节顺序

**先理解项目，再教怎么干活：**

```
1. Project Overview          # 建立上下文
2. Tech Stack                # 选型锚点
3. Project Structure         # 导航图（第一层 + 职责注释）
4. Architecture Constraints  # 分层规则 + 目录规则 + 禁止事项
5. Coding Standards          # 通用规范 + 项目补充规则
6. Documentation Structure   # 文档导航 + 阅读顺序
7. Issue Development Workflow # 操作步骤 + 模块知识选择 + 模块索引
8. Quick Lookup Tables       # 外置引用
9. Commands                  # 常用命令
```

---

## 9. 维护检查清单

每次修改 agent.md 时，逐项检查：

- [ ] **新增内容是规则还是事实？** 事实通常不该加
- [ ] **agent 能否从代码推断？** 能推断的不需要加
- [ ] **是否与 README.md 重复？** 只在一个地方保留完整版
- [ ] **是否与 agent.md 其他章节重复？** 合并而非新增
- [ ] **行数是否超过 200？** 超过则需要考虑压缩或外置
- [ ] **修改后是否需要同步更新其他文件？** 检查引用关系

### 定期审计（建议每季度）

1. 对比 agent.md 中的信息与代码实际状态
2. 删除 agent 已经能正确遵循的规则（不再需要的指令）
3. 添加 agent 反复犯错后总结的新规则
4. 验证技术栈版本与 go.mod / package.json 一致
5. 验证目录结构与实际代码一致

---

## 10. 参考资料

| 来源 | 链接 | 要点 |
|------|------|------|
| Anthropic 官方 | code.claude.com/docs/en/best-practices | 文件层级、大小建议、避免过度指定 |
| OpenAI Codex | developers.openai.com/codex/guides/agents-md | AGENTS.md 规范、32KB 限制 |
| Cursor | cursor.com/blog/agent-best-practices | 引用文件而非复制内容、按需添加规则 |
| Google Cloud | cloud.google.com/blog 上的 AI coding best practices | GEMINI.md 概念、跨会话连续性 |
| AGENTS.md 标准 | 社区跨工具标准，20,000+ 仓库采用 | symlink 策略实现跨工具兼容 |

> 关键共识（Anthropic + OpenAI + Cursor）：
> - 短小精悍比面面俱到更有效
> - 只在 agent 犯错后添加规则，不要预防性堆砌
> - 引用文件，不要复制内容
> - 定期清理，删除已经能正确遵循的指令
