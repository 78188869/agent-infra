# TRD Template Reference

> **When to use:** Read this file when you're ready to generate the TRD document after completing discovery.

---

# TRD: [System Name] Technical Requirements Document

> **Version:** v0.1-draft | **Date:** YYYY-MM-DD | **Status:** Draft
> **PRD Reference:** [Link to PRD document]

---

## 1. Executive Summary

[2-3 sentence overview of the technical approach]

**Key Technical Decisions:**
- [e.g., Monolithic architecture for MVP simplicity]
- [e.g., PostgreSQL for ACID compliance]
- [e.g., Kubernetes deployment for scalability]

---

## 2. System Architecture

### Architecture Overview

[1-2 paragraphs describing the high-level architecture: monolith/microservices, layers, key patterns]

### Component Responsibilities

| Component | Responsibility | Technology | PRD Reference |
|-----------|---------------|------------|---------------|
| API Server | Handle HTTP requests, business logic | Go 1.21+ | PRD §X.X |
| Frontend | User interface | React 18.x | PRD §X.X |
| Database | Persistent storage | PostgreSQL 15.x | PRD §X.X |

### Communication Patterns

| From | To | Protocol | Purpose |
|------|-----|----------|---------|
| Frontend | API Server | HTTPS/REST | User requests |
| API Server | Database | SQL/TCP | Data persistence |
| API Server | Cache | Redis Protocol | Session/cache |

---

## 3. API Specifications

### Authentication

| Method | Header | Format |
|--------|--------|--------|
| API Key | `X-API-Key` | `key_xxx` |
| Bearer Token | `Authorization` | `Bearer <jwt>` |

### API Endpoints

| Method | Path | Description | PRD Reference |
|--------|-----|-------------|---------------|
| GET | `/api/v1/resources` | List resources (paginated) | PRD §X.X |
| POST | `/api/v1/resources` | Create resource | PRD §X.X |
| GET | `/api/v1/resources/{id}` | Get resource by ID | PRD §X.X |
| PUT | `/api/v1/resources/{id}` | Update resource | PRD §X.X |
| DELETE | `/api/v1/resources/{id}` | Delete resource | PRD §X.X |

### Response Format

```json
{
  "code": 0,
  "message": "success",
  "data": { ... },
  "request_id": "req-xxx"
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| 0 | 200 | Success |
| 1001 | 400 | Invalid request parameters |
| 1002 | 401 | Unauthorized |
| 1003 | 403 | Forbidden |
| 1004 | 404 | Resource not found |
| 1005 | 409 | Conflict (e.g., duplicate) |
| 1006 | 500 | Internal server error |

---

## 4. Data Models

### Entity Relationships

[Describe key entities and their relationships: User has many Tasks, Task belongs to Template, etc.]

### Database Schema

```sql
-- Users Table
CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tasks Table
CREATE TABLE tasks (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id),
    title VARCHAR(500) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_status (status)
);
```

---

## 5. Technology Stack

| Layer | Technology | Version | Purpose |
|-------|------------|---------|---------|
| Frontend | React | 18.x | UI framework |
| Backend | Go | 1.21+ | API server |
| Database | PostgreSQL | 15.x | Primary storage |
| Cache | Redis | 7.x | Session/cache |
| Container | Docker | 24.x | Containerization |
| Orchestration | Kubernetes | 1.28+ | Deployment |

### Key Libraries

| Library | Version | Purpose |
|---------|---------|---------|
| gin | 1.9+ | HTTP framework |
| gorm | 1.25+ | ORM |
| jwt-go | 5.x | JWT handling |

---

## 6. Deployment Architecture

### Infrastructure

[1-2 sentences: e.g., Deployed on AWS EKS with 3 availability zones, using ALB for load balancing]

### Resource Requirements

| Component | CPU | Memory | Replicas |
|-----------|-----|--------|----------|
| API Server | 2 cores | 4GB | 3 |
| PostgreSQL | 4 cores | 16GB | 2 (HA) |
| Redis | 2 cores | 8GB | 3 |

### Scaling Strategy

| Metric | Threshold | Action |
|--------|-----------|--------|
| CPU | > 70% | Add replica (max 10) |
| Memory | > 80% | Add replica (max 10) |
| Requests/sec | > 1000 | Scale horizontally |

---

## 7. Security Design

| Area | Measure | Implementation |
|------|---------|----------------|
| Transport | TLS 1.3 | All connections encrypted |
| Authentication | API Key / JWT | Header-based validation |
| Authorization | RBAC | Role-based access control |
| Data | AES-256 encryption | Sensitive fields encrypted at rest |
| Secrets | Vault / Secrets Manager | No hardcoded credentials |

---

## 8. Performance Requirements

| Operation | Target | Max Acceptable |
|-----------|--------|----------------|
| API Response (p95) | < 100ms | < 500ms |
| Database Query | < 50ms | < 200ms |
| Cache Hit | < 5ms | < 20ms |

### Throughput

| Metric | Target | Peak Capacity |
|--------|--------|---------------|
| Requests/sec | 100 | 500 |
| Concurrent Users | 1,000 | 5,000 |
| Data Storage | 100GB | 1TB |

---

## 9. Trade-off Decisions

| Decision | Options | Pros | Cons | Choice | Reason |
|----------|---------|------|------|--------|--------|
| Architecture | A) Monolith / B) Microservices | A: Simple deploy / B: Independent scaling | A: Single point of failure / B: Complexity | Monolith | MVP phase, small team |
| Database | A) PostgreSQL / B) MongoDB | A: ACID, mature / B: Flexible schema | A: Schema migrations / B: No transactions | PostgreSQL | Data requires ACID |
| API Style | A) REST / B) GraphQL | A: Simple, cacheable / B: Flexible queries | A: Over-fetching / B: Complexity | REST | Standard CRUD operations |

---

## 10. Open Technical Questions

| # | Question | Owner | Priority | Status |
|---|----------|-------|----------|--------|
| 1 | [Technical question requiring decision] | [Name] | P0 | Open |

---

## 11. Revision History

| Version | Date | Changes | Author | Status |
|---------|------|---------|--------|--------|
| v0.1-draft | YYYY-MM-DD | Initial draft | - | Draft |
| v1.0 | YYYY-MM-DD | Approved | - | Approved |
