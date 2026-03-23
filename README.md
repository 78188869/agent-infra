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

### Workflow

1. **Create Worktree** - Isolated directory for development
   ```bash
   make worktree
   # or
   ./scripts/create-worktree.sh <issue-number>
   ```

2. **Create Issue Summary** - Document the task
   ```bash
   # Create docs/current/issues/issue-{number}-{title}.md
   ```

3. **Gather Knowledge** - Read relevant docs
   ```bash
   # Read docs/current/TRD.md
   # Read docs/current/decisions/
   # Read docs/knowledge/{modules}.md
   ```

4. **Create Plan** - Document the approach
   ```bash
   # Create docs/current/plans/{date}-{title}.md
   ```

5. **Develop** - Work in the isolated worktree
   ```bash
   cd /Users/yang/workspace/learning/agent-infra/worktrees/<issue-number>
   # Make changes...
   ```

6. **Test & Verify**
   ```bash
   make test
   make lint
   ```

7. **Commit & Push**
   ```bash
   git add .
   git commit -m "feat(scope): description"
   git push -u origin <branch-name>
   ```

8. **Create Pull Request**
   ```bash
   gh pr create --base main --title "feat: description"
   ```

9. **Merge** - After PR approval
   ```bash
   gh pr merge --merge
   make clean-worktree  # optional cleanup
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
