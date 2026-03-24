# Agent Infra

Agentic Coding Platform - A universal agent task execution platform.

## License

This project is licensed under the **GNU Affero General Public License v3.0 (AGPL-3.0)**.

See [LICENSE](LICENSE) for the full license text.

**Key Points:**
- Free to use and modify
- Derivative works must be open source (share alike)
- Network users (SaaS) are entitled to source code
- Cannot be used in closed-source commercial products without sharing modifications

For commercial licensing options, please contact the maintainers.

---

## Documentation Structure

```
docs/
├── knowledge/                 # 独立知识库（持续演进）
│   ├── core-api.md           # 任务/模板管理 API
│   ├── database.md           # 数据模型与存储
│   ├── scheduler.md          # 任务调度引擎
│   ├── executor.md           # 沙箱执行引擎
│   ├── provider.md           # Agent 运行时配置
│   ├── capability.md         # 能力注册管理
│   ├── intervention.md       # 人工干预机制
│   └── monitoring.md         # 监控告警设计
│
├── v{version}/               # 版本目录
│   ├── BRD.md                # 业务需求文档
│   ├── PRD.md                # 产品需求文档
│   ├── TRD.md                # 技术设计文档
│   ├── decisions/            # 架构决策记录（ADR）
│   ├── issues/               # Issue 摘要
│   └── plans/                # 执行计划
│
└── current -> v{version}/    # 当前活跃版本
```

---

## Issue Development Workflow

当开始开发或规划一个 Issue 时，按以下流程操作：

### 1. 创建 Issue Summary

在 `docs/current/issues/` 创建文件：

```bash
# 文件命名：issue-{number}-{short-title}.md
# 例如：issue-001-auth-timeout.md
```

### 2. 获取背景知识

根据工作类型，读取相关文档：

| 工作类型 | 知识模块 | 决策参考 |
|---------|---------|---------|
| API 开发 | core-api, database | TRD §5, decisions/ |
| 调度逻辑 | scheduler, database | TRD §3.2 |
| 执行管理 | executor, provider | TRD §3.2 |
| 能力管理 | capability, provider | TRD §3.2 |

### 3. 生成执行计划

在 `docs/current/plans/` 创建文件：

```bash
# 文件命名：{YYYY-MM-DD}-{short-title}.md
# 例如：2026-03-23-api-implementation.md
```

### 4. 执行与更新

- 按 Plan 执行任务
- 更新 knowledge 模块的 Change History
- 如有架构决策，创建新的 ADR

---

## Development Workflow

This project uses **Git Worktrees** for isolated development. Each issue/feature gets its own isolated working directory.

### Creating a Worktree

```bash
# Create worktree for an issue
make worktree
# Or use the helper script:
./scripts/create-worktree.sh <issue-number>
```

### Complete Workflow

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Issue Development Workflow                         │
└─────────────────────────────────────────────────────────────────────┘

Step 0: Pull Latest Code
    │
    ├── git pull origin main
    │
    ▼
Step 1: Create Worktree
    │
    ├── git worktree add -b feature/issue-{number} .claude/worktrees/issue-{number} main
    │
    ▼
Step 2: Create Issue Summary
    │
    ├── 在 docs/current/issues/ 创建 issue-{number}-{title}.md
    │
    ▼
Step 3: Gather Knowledge
    │
    ├── 读取 docs/current/TRD.md
    ├── 读取 docs/current/decisions/
    ├── 读取 docs/knowledge/{modules}.md
    │
    ▼
Step 4: Create Plan
    │
    ├── 在 docs/current/plans/ 创建 {date}-{title}.md
    │
    ▼
Step 5: Develop (TDD)
    │
    ├── 按照计划逐步实现
    ├── 编写测试用例
    ├── 确保测试覆盖率 > 80%
    │
    ▼
Step 6: Test & Verify
    │
    ├── make test
    ├── make lint
    ├── go test -cover ./internal/...
    │
    ▼
Step 7: Code Review (Test Cases)
    │
    ├── 使用 test-cases.md 进行代码审查
    ├── 验证所有测试用例通过
    ├── 修复发现的问题
    │
    ▼
Step 8: Commit & Push
    │
    ├── git add .
    ├── git commit -m "feat(scope): description"
    ├── git push -u origin <branch-name>
    │
    ▼
Step 9: Create Pull Request
    │
    ├── gh pr create --base main --title "feat: description"
    │
    ▼
Step 10: Wait for PR Merge (⏳ Human Required)
    │
    ├── 等待人工审核并合并 PR
    ├── 人工确认: 输入 "merged" 或 PR 编号
    │
    ▼
Step 11: Pull Merged Changes
    │
    ├── git checkout main
    ├── git pull origin main
    │
    ▼
Step 12: Close Issue
    │
    ├── gh issue close {number} --repo {repo}
    ├── 更新 Issue Summary 状态
    │
    ▼
Step 13: Cleanup Environment
    │
    ├── git worktree remove .claude/worktrees/issue-{number}
    ├── git branch -d feature/issue-{number} (可选)
    │
    ▼
✅ Complete
```

### Quick Workflow Commands

```bash
# 1. Pull latest code
git pull origin main

# 2. Create worktree for issue #10
git worktree add -b feature/issue-10-capability-management .claude/worktrees/issue-10-capability-management main
cd .claude/worktrees/issue-10-capability-management

# 3. After development - Run tests
make test
make lint

# 4. Check coverage
go test ./internal/... -coverprofile=coverage.out
go tool cover -func=coverage.out

# 5. Push and create PR
git push -u origin feature/issue-10-capability-management
gh pr create --base main --title "feat(capability): implement Capability Management System"

# 6. After PR merge (human confirms)
git checkout main
git pull origin main

# 7. Close issue
gh issue close 10 --repo 78188869/agent-infra

# 8. Cleanup worktree
git worktree remove .claude/worktrees/issue-10-capability-management
```

### Branch Naming Convention

| Branch Type | Pattern | Example |
|------------|---------|---------|
| Feature | `feat/<scope>/<description>` | `feat/api/add-tasks-endpoint` |
| Fix | `fix/<scope>/<description>` | `fix/scheduler/race-condition` |
| Docs | `docs/<description>` | `docs/api-reference` |
| Chore | `chore/<description>` | `chore/update-dependencies` |

---

## Quick Reference

### Commands

```bash
# Backend
make run test lint

# Frontend
cd web && npm run dev

# Deploy
make docker-build-all k8s-apply k8s-status

# Worktrees
git worktree list
make worktree
make clean-worktree
```

### Useful Links

| Document | Path |
|----------|------|
| Agent Guide | [agent.md](./agent.md) |
| Business Requirements | [docs/BRD.md](./docs/BRD.md) |
| Current Version | [docs/current/](./docs/current/) |
| Knowledge Base | [docs/knowledge/](./docs/knowledge/) |

---

## For Coding Agents

If you are a coding agent (like Claude Code), please read [agent.md](./agent.md) first for detailed workflow instructions.

**Quick Start:**
1. Read `agent.md` - Project overview and workflow
2. Check `docs/current/` - Current version documents
3. Read relevant `docs/knowledge/{module}.md` - Module details
4. Create Issue Summary in `docs/current/issues/`
5. Create Plan in `docs/current/plans/`
6. Execute and update knowledge
