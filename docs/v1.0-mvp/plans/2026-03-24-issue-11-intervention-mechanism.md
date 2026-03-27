# Plan: Human Intervention Mechanism (Issue #11)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement human intervention mechanism for task execution, supporting pause/resume/cancel/inject operations with full audit trail.

**Architecture:** Layered architecture following existing patterns (Repository → Service → Handler). Intervention service coordinates with Task service for state transitions and maintains intervention records for audit purposes.

**Tech Stack:** Go 1.22 + Gin 1.9 + GORM 1.25

---

## Context

### Issue Summary
- **Issue**: #11 - MVP Phase 7 - Human Intervention Mechanism
- **PRD Reference**: US-D03 (人工干预任务), US-O02 (异常处理)
- **TRD Reference**: §7.2 人工干预机制
- **Knowledge**: docs/knowledge/intervention.md

### Dependencies
| Issue | Title | Status |
|-------|-------|--------|
| #5 | Backend Core API Development | ✅ CLOSED |
| #8 | Task Executor Engine | ✅ CLOSED |

### Existing Infrastructure
- Task model has `paused` status and `validStatusTransitions` map
- Task service has `isValidStatusTransition()` function
- Scheduler supports task preemption (future: inject command)

---

## Objectives

1. Implement Intervention model with audit trail
2. Implement InterventionRepository for data persistence
3. Implement InterventionService with business logic
4. Implement InterventionHandler with 6 API endpoints
5. Integrate with Task service for state transitions
6. Add database migration for interventions table
7. Update Issue Summary documentation

8. Run tests and verify coverage > 80%

---

## Knowledge Required

- [x] docs/knowledge/intervention.md
- [x] docs/v1.0-mvp/TRD.md §7.2
- [x] internal/model/task.go (existing status constants)
- [x] internal/service/task_service.go (status transitions)

---

## File Structure

### New Files
| File | Purpose |
|------|---------|
| `internal/model/intervention.go` | Intervention model with audit fields |
| `internal/repository/intervention_repo.go` | Repository interface and implementation |
| `internal/repository/intervention_repo_test.go` | Repository unit tests |
| `internal/service/intervention_service.go` | Service interface and implementation |
| `internal/service/intervention_service_test.go` | Service unit tests |
| `internal/api/handler/intervention.go` | HTTP handlers for intervention endpoints |
| `internal/api/handler/intervention_test.go` | Handler unit tests |
| `internal/migration/20260324000003_create_interventions_table.go` | Database migration |
| `docs/v1.0-mvp/issues/issue-11-summary.md` | Issue summary |

### Modified Files
| File | Changes |
|------|---------|
| `internal/api/router/router.go` | Add intervention routes |
| `internal/api/router/router_test.go` | Update mock services |
| `cmd/control-plane/main.go` | Add InterventionService initialization |
| `docs/v1.0-mvp/issues/README.md` | Add issue #11 entry |

---

## Tasks

### Phase 1: Data Layer

#### Task 1: Intervention Model

**Files:**
- Create: `internal/model/intervention.go`

**Implementation Notes:**
- Define InterventionAction constants (pause, resume, cancel, inject, modify)
- Define InterventionStatus constants (pending, applied, failed)
- Intervention struct with fields: ID, TaskID, OperatorID, Action, Content (JSON), Reason, Result (JSON), Status, CreatedAt
- Add foreign key relationships to Task and User models
- Implement TableName() method

**Testing:**
- Verify model can be created with GORM
- Verify field types and constraints

---

#### Task 2: Intervention Repository

**Files:**
- Create: `internal/repository/intervention_repo.go`
- Create: `internal/repository/intervention_repo_test.go`

**Implementation Notes:**
- InterventionRepository interface with methods: Create, GetByID, ListByTask, ListByOperator
- Implement using GORM with proper error handling
- Use project's error types (NotFoundError, InternalError)
- Add filtering support (by task_id, by operator_id, by status)

**Testing:**
- Test Create intervention
- Test GetByID (found and not found cases)
- Test ListByTask with pagination
- Test ListByOperator with pagination
- Use mock repository for unit tests or SQLite for integration tests

---

### Phase 2: Business Logic Layer

#### Task 3: Intervention Service

**Files:**
- Create: `internal/service/intervention_service.go`
- Create: `internal/service/intervention_service_test.go`

**Implementation Notes:**
- InterventionService interface with methods: Pause, Resume, Cancel, Inject, ListInterventions
- InjectInterventionRequest struct for inject action
- Service depends on TaskService and InterventionRepository
- Validate task state before intervention (use state transition matrix)
- Update task status via TaskService
- Create intervention record with audit trail
- Return intervention status for API response

**State Transition Matrix:**
| Current Status | pause | resume | cancel | inject |
|---------------|-------|--------|--------|--------|
| pending | ✗ | ✗ | ✓ | ✗ |
| scheduled | ✗ | ✗ | ✓ | ✗ |
| running | ✓ | ✗ | ✓ | ✓ |
| paused | ✗ | ✓ | ✓ | ✗ |
| waiting_approval | ✗ | ✗ | ✓ | ✓ |
| retrying | ✗ | ✗ | ✓ | ✗ |
| succeeded | ✗ | ✗ | ✗ | ✗ |
| failed | ✗ | ✗ | ✗ | ✗ |
| cancelled | ✗ | ✗ | ✗ | ✗ |

**Testing:**
- Test Pause with valid/invalid task states
- Test Resume with valid/invalid task states
- Test Cancel with valid/invalid task states
- Test Inject with valid/invalid task states
- Test ListInterventions with pagination
- Test error handling for non-existent tasks
- Use mock TaskService and InterventionRepository

---

### Phase 3: API Layer

#### Task 4: Intervention Handler

**Files:**
- Create: `internal/api/handler/intervention.go`
- Create: `internal/api/handler/intervention_test.go`

**Implementation Notes:**
- InterventionHandler struct with service dependency
- Handlers: Pause, Resume, Cancel, Inject, ListInterventions
- Extract user_id from context for operator_id (add TODO for auth middleware)
- Use response helpers for success/error responses
- Add proper HTTP status codes (200, 201, 400, 404, 500)

**Testing:**
- Test each endpoint with valid/invalid requests
- Test authorization (user_id from context)
- Test error responses for various scenarios
- Use mock InterventionService

---

#### Task 5: Router Integration

**Files:**
- Modify: `internal/api/router/router.go`
- Modify: `internal/api/router/router_test.go`
- Modify: `cmd/control-plane/main.go`

**Implementation Notes:**
- Add InterventionService parameter to router.Setup()
- Register intervention routes under `/api/v1/tasks/:id/`:
  - POST `/tasks/:id/pause`
  - POST `/tasks/:id/resume`
  - POST `/tasks/:id/cancel`
  - POST `/tasks/:id/inject`
  - GET `/tasks/:id/interventions`
- Update mock services in router tests
- Initialize InterventionService in main.go

---

### Phase 4: Database Migration

#### Task 6: Database Migration

**Files:**
- Create: `internal/migration/20260324000003_create_interventions_table.go`

**Implementation Notes:**
- Follow TRD §6.2.7 interventions table definition
- Use GORM AutoMigrate compatible approach
- Add foreign key constraints to tasks and users tables
- Add indexes on task_id, operator_id, created_at

---

### Phase 5: Documentation

#### Task 7: Issue Summary

**Files:**
- Create: `docs/v1.0-mvp/issues/issue-11-summary.md`
- Modify: `docs/v1.0-mvp/issues/README.md`

**Implementation Notes:**
- Create issue summary following template
- Link to PRD/TRD references
- List scope items
- Update README.md with issue #11 entry

- Mark status as In Progress when starting

---

## Dependencies

### Internal Dependencies
- TaskService: For status updates and task retrieval
- TaskRepository: For task queries (via TaskService)
- InterventionRepository: For intervention persistence

### External Dependencies
- None for MVP (WebSocket real-time push deferred to post-MVP)

---

## Risks

| Risk | Mitigation |
|------|------------|
| Inject command requires executor integration | MVP scope: record intervention only, actual injection to executor is separate concern (Issue #8 follow-up) |
| Auth middleware not yet implemented | Use placeholder for user_id from context, add TODO comment |
| WebSocket push not in MVP | Deferred to post-MVP, API returns intervention status |

---

## Out of Scope (Post-MVP)

- WebSocket real-time push for intervention events
- Checkpoint/approval workflow (US-D04 MR审核)
- Modify action (parameter modification during pause)
- Retry action (can be handled via task update)

---

## Status

Not Started

---

## Change History

| Date | Change |
|------|--------|
| 2026-03-24 | Created plan |
