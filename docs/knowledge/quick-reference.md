# Quick Reference

> 本文档包含开发中高频查询的参考数据。

---

## API Response Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 400xx | Request errors |
| 401xx | Auth errors |
| 403xx | Forbidden |
| 404xx | Not found |
| 500xx | Server errors |

---

## Task Status

```
Pending → Scheduled → Running → Succeeded
                   ↓          ↓
               Paused      Failed → Retrying
```

---

## Redis Keys

| Pattern | Description |
|---------|-------------|
| `scheduler:queue:tasks` | Priority queue (single sorted set with encoded score) |
| `scheduler:task:{id}:meta` | Task metadata |
| `scheduler:tenant:{id}:quota` | Tenant quota (concurrency) |
| `scheduler:tenant:{id}:daily:{date}` | Tenant daily task count (expires at midnight) |
| `scheduler:global:quota` | Global concurrency quota |
| `scheduler:task:{id}:state` | Preempted task state (JSON, 24h TTL) |
| `scheduler:preempted:tasks` | Set of preempted task IDs |
