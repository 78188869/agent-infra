# Local Development Guide

## Prerequisites

- Go 1.21+
- Docker (optional, for task execution)

**No MySQL, Redis, or Kubernetes required.**

## Quick Start

```bash
make local
```

This single command:
1. Creates `data/agent_infra.db` (SQLite)
2. Starts in-memory Redis (miniredis)
3. Runs database migrations
4. Seeds system providers (Claude Code, Zhipu GLM, DeepSeek)
5. Starts the HTTP server on `:8080`

## Architecture (Local Mode)

| Component | Production | Local |
|-----------|-----------|-------|
| Database | OceanBase (MySQL) | SQLite (`data/agent_infra.db`) |
| Cache/Queue | Redis 6 | miniredis (in-memory) |
| Container Runtime | K8s Jobs | Docker Compose |
| Logging | Aliyun SLS | File (`logs/`) + stdout |

## API Endpoints

After startup, the API is available at `http://localhost:8080`:

```bash
# Health check
curl http://localhost:8080/health

# Create a tenant
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name": "test-tenant"}'

# List providers (seeded)
curl http://localhost:8080/api/v1/providers
```

## Configuration

Local config is at `configs/config.local.yaml`. Key settings:

- `database.driver: sqlite` — file-based database
- `database.name: data/agent_infra.db` — database file path
- `log.level: debug` — verbose logging
- `log.outputs: both` — stdout + file logging

Seed data is idempotent — running `make local` multiple times won't create duplicate providers.

## Without Docker

If Docker is not installed, the app still starts successfully. Task creation and management via API works, but task execution will fail with a Docker error. This is expected.

## Cleanup

```bash
rm -rf data/ logs/
```
