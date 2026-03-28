# Agent Infra

A universal agent task execution platform that wraps Claude Code CLI to provide a managed execution environment for coding tasks.

---

## Features

- **Task Scheduling** — Queue management with rate limiting and priority-based dispatch
- **Sandboxed Execution** — Kubernetes-based isolated task execution with resource limits
- **Provider Management** — Multi-agent runtime configuration (Claude Code, MCP servers)
- **Capability Registry** — Tool registration and permission management
- **Human Intervention** — Checkpoint-based approval flow for critical operations
- **Monitoring** — Real-time metrics, logging, and alerting via SLS

## Architecture

```
                    ┌─────────────┐
                    │   Frontend  │  React + Ant Design
                    │   (Web UI)  │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │  API Layer  │  Go + Gin
                    │  (Handlers) │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
       ┌──────▼──┐  ┌─────▼─────┐  ┌──▼────────┐
       │Scheduler│  │ Executor  │  │Intervention│
       │ Engine  │  │  Engine   │  │  Manager   │
       └────┬────┘  └─────┬─────┘  └───────────┘
            │             │
       ┌────▼────┐  ┌─────▼─────┐
       │  Redis  │  │ Kubernetes │
       │ (Queue) │  │   (Pods)   │
       └─────────┘  └───────────┘
```

| Layer | Technology |
|-------|-----------|
| Frontend | React 18 + TypeScript 5 + Ant Design 5 + Vite 5 |
| Backend | Go 1.21 + Gin 1.9 + GORM 1.25 |
| Database | OceanBase (MySQL compatible) |
| Cache/Queue | Redis 6 |
| Container | Kubernetes (ACK) + Docker |
| Execution | Claude Code CLI |

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+
- Docker & Kubernetes cluster
- Redis 6+

### Run Backend

```bash
# Configure
cp configs/config.yaml.example configs/config.yaml

# Database migration
make db-migrate

# Start server
make run
```

### Run Frontend

```bash
cd web
npm install
npm run dev
```

### Development Commands

```bash
make test           # Run tests
make lint           # Lint code
make test-coverage  # Coverage report
```

## Documentation

| Document | Description |
|----------|------------|
| [agent.md](./agent.md) | AI coding agent guide (workflow, standards, rules) |
| [Business Requirements](./docs/BRD.md) | Business context and goals |
| [Product Requirements](./docs/current/PRD.md) | User stories and features |
| [Technical Design](./docs/current/TRD.md) | Architecture and API design |
| [Knowledge Base](./docs/knowledge/) | Module-specific documentation |

## Contributing

1. Pick or create an [issue](https://github.com/78188869/agent-infra/issues)
2. Create a feature branch: `feat/<scope>/<description>`
3. Develop with TDD — tests required, coverage > 80%
4. Commit with conventional format: `feat(scope): description`
5. Open a PR against `main`

> This project uses Git Worktrees for isolated development. See [agent.md](./agent.md) for the full workflow.

## License

This project is licensed under the **GNU Affero General Public License v3.0 (AGPL-3.0)**. See [LICENSE](LICENSE) for details.

- Free to use and modify
- Derivative works must be open source (share alike)
- Network users (SaaS) are entitled to source code

For commercial licensing options, please contact the maintainers.
