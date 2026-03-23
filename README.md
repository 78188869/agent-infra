# Agent Infra

Agentic Coding Platform - A universal agent task execution platform.

## License

This project is licensed under the **GNU Affero General Public License v3.0 (AGPL-3.0)**.

See [LICENSE](LICENSE) for the full license text.

**Key Points:**
- ✅ Free to use and modify
- ✅ Derivative works must be open source (share alike)
- ✅ Network users (SaaS) are entitled to source code
- ❌ Cannot be used in closed-source commercial products without sharing modifications

For commercial licensing options, please contact the maintainers.

## Development Workflow

This project uses **Git Worktrees** for isolated development. Each issue/feature gets its own isolated working directory.

### Creating a Worktree for```bash
# Create worktree formake worktree
# Or use the helper script:
./scripts/create-worktree.sh <issue-number>
```

### Workflow

1. **Create Worktree** - Isolated directory for   ```bash
   make worktree
   # or
   ./scripts/create-worktree.sh <issue-number>
   ```

2. **Develop** - Work in the isolated worktree
   ```bash
   cd /Users/yang/workspace/learning/agent-infra/worktrees/<issue-number>
   # Make changes...
   ```

3. **Test & Verify**
   ```bash
   make test
   make lint
   ```

4. **Commit & Push**
   ```bash
   git add .
   git commit -m "feat(scope): description"
   git push -u origin <branch-name>
   ```

5. **Create Pull Request**
   ```bash
   gh pr create --base main --title "feat: description"
   ```

6. **Merge** - After PR approval
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

### Useful Commands

```bash
# List all worktrees
git worktree list

# Check status
make k8s-status

# View logs
make k8s-logs
```
