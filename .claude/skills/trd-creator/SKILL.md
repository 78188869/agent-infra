---
name: trd-creator
description: Use when users ask to "create a TRD", "write technical requirements document", "design architecture", "define API specifications", "create database schema", or need help with technical implementation details AFTER having a PRD. Also triggers when converting product requirements into technical specs, designing system architecture, making technology stack decisions, or when developers need detailed implementation guidance. Make sure to use this skill whenever the user mentions "API design", "database schema", "architecture", "deployment", "technology stack", "system design", or wants to translate product features into concrete technical specifications, even if they don't explicitly ask for a "TRD". Always check if a PRD exists first — if not, recommend creating one.
---

# TRD Creator

Create comprehensive Technical Requirements Documents (TRDs) that define architecture design, API specifications, database schemas, and technology stack decisions — building on PRD's product specifications to answer HOW to implement.

## Core Principle

**TRD answers "HOW" — not "WHY" or "WHAT".**

- **HOW**: Architecture, APIs, database schemas, deployment, technology choices
- **NOT**: Business goals, user stories, UX flows, success metrics (those are BRD/PRD)

```
BRD → PRD → TRD
WHY/WHO → WHAT (user-facing) → HOW (technical)
```

**Key insight:** TRD bridges PRD's product features to implementation. If you're writing business goals or user stories → STOP → That belongs in BRD/PRD.

## When to Use This Skill

**Use TRD when:**
- PRD exists and product features are clear
- Need to translate features into technical implementation
- Designing system architecture
- Defining API specifications
- Creating database schemas
- Making technology stack decisions

**Skip TRD when:**
- No PRD exists (create PRD first using prd-creator skill)
- Simple feature or bug fix with obvious implementation
- Technical details already well-defined

## Phase 1: Input Validation (PRD Check)

**CRITICAL: TRD requires PRD as input.**

Before proceeding, ask: "Do you have an existing PRD for this project?"

- **If YES** → Extract key information from PRD
- **If NO** → Recommend creating PRD first using prd-creator skill

### Extract from PRD

| PRD Section | TRD Input | Purpose |
|-------------|-----------|---------|
| User Stories | API endpoints, data models | Each story needs technical implementation |
| Feature Specs | Components, services | Define what needs to be built |
| UX Flows | API sequences, state management | Map user actions to system behavior |
| Success Metrics | Performance requirements | Define SLAs, scaling targets |
| Scope | Implementation scope | What to build vs defer |

### PRD-TRD Traceability

Every TRD element should trace back to PRD. If you can't trace an API endpoint or data model to a PRD feature → ask: Is this truly needed for MVP?

## Phase 2: Technical Discovery

### Architecture Questions

1. **Deployment Context**: Where will this run? (cloud, on-premise, hybrid)
2. **Scaling Requirements**: Expected load? Concurrent users? Data volume?
3. **Integration Points**: External systems? APIs? Services?
4. **Constraints**: Budget, timeline, compliance, team expertise?

### Technology Questions

1. **Stack Preferences**: Existing standards? Team familiarity?
2. **Data Requirements**: Relational? Document? Time-series?
3. **Communication Patterns**: Sync vs async? REST vs GraphQL? Event-driven?
4. **Observability**: Logging, metrics, tracing requirements?

### Security Questions

1. **Authentication**: How do users authenticate?
2. **Authorization**: Role-based? Resource-based?
3. **Data Protection**: Encryption at rest? In transit?
4. **Compliance**: GDPR? SOC2? Industry-specific?

## Phase 3: Generate TRD

**Read the template from `references/trd-template.md` to generate the TRD document.**

The template includes 11 sections: Executive Summary, System Architecture, API Specifications, Data Models, Technology Stack, Deployment Architecture, Security Design, Performance Requirements, Trade-off Decisions, Open Questions, and Revision History.

**Fill in the template with information gathered from:**
- PRD (features, user stories, success metrics)
- Discovery questions (architecture, technology, security)
- Trade-off analysis for key decisions

## Phase 4: Trade-off Analysis (Critical)

For each significant technical decision, document:

### Format

| Decision | Options | Pros | Cons | Choice | Reason |
|----------|---------|------|------|--------|--------|
| [Topic] | [A/B/C] | [Benefits] | [Drawbacks] | [Selected] | [Why this option] |

### Example

| Decision | Options | Pros | Cons | Choice | Reason |
|----------|---------|------|------|--------|--------|
| Database | A) PostgreSQL / B) MongoDB | A: ACID, mature / B: Flexible schema | A: Schema migrations / B: No transactions | PostgreSQL | Financial data requires ACID guarantees |

### Key Decisions to Document

- Architecture pattern (monolith vs microservices)
- Database choice
- API style (REST vs GraphQL vs gRPC)
- Authentication method
- Deployment strategy
- Caching strategy

## Phase 5: Logic & Completeness Review

After drafting the TRD, perform these checks BEFORE finalizing:

### 5.1 PRD-TRD Traceability
- Each API endpoint traces to a PRD feature?
- Each data model supports a PRD requirement?
- No orphan technical components (no PRD connection)?

### 5.2 Architecture Consistency
- API layer doesn't bypass service layer
- Database schema matches data models
- Deployment matches scaling requirements

### 5.3 No Business Leakage
**RED FLAG TERMS (should NOT appear in TRD):**
- User personas, user stories, acceptance criteria
- Business goals, success metrics (KPI/NPS belong in PRD)
- UX flows, wireframes, mockups

If found → Replace with technical language (e.g., "User persona" → "Client type with specific access pattern")

### 5.4 Completeness Checklist
- [ ] Architecture diagram included
- [ ] All API endpoints documented
- [ ] Database schema defined
- [ ] Technology stack listed
- [ ] Deployment strategy clear
- [ ] Error handling specified
- [ ] Security considerations addressed

## Phase 6: Iteration & Refinement

TRD is a living document. Iterate based on:
- Developer feedback during implementation
- Technical feasibility findings
- PRD changes (TRD must stay aligned)
- New constraints discovered

### Sign-off Process
Before marking TRD as "approved":
- [ ] Tech lead has reviewed
- [ ] All trade-offs documented
- [ ] All Phase 5 checks pass
- [ ] Ready for implementation

## Critical Rules

### Rule 1: No Business/Product Content

```
❌ BAD: "As a user, I want to login quickly"
✅ GOOD: "POST /auth/login endpoint, <200ms response time"

❌ BAD: "Improve user satisfaction"
✅ GOOD: "API response time < 100ms p99"

❌ BAD: "Users should feel confident"
✅ GOOD: "TLS 1.3 encryption for all API traffic"
```

### Rule 2: Technical Precision

```
❌ BAD: "Use a database"
✅ GOOD: "PostgreSQL 15 with read replicas"

❌ BAD: "Make it scalable"
✅ GOOD: "Horizontal scaling to 10 pods, each 2vCPU/4GB"

❌ BAD: "Handle errors well"
✅ GOOD: "Return structured error codes: E001-E018"
```

### Rule 3: Every Decision Needs Trade-offs
For each major technical choice, document why NOT the alternatives. If you can't explain why you chose X over Y → The decision isn't justified.

## Self-Review Checklist

Before finalizing, verify:

### PRD Alignment
- [ ] PRD exists: TRD is built on documented product requirements
- [ ] Traceability: Each API/model traces to PRD feature
- [ ] Performance: Metrics match PRD success criteria

### Technical Quality
- [ ] Architecture complete: All layers defined
- [ ] API specifications: All endpoints documented
- [ ] Data models: Schema with relationships
- [ ] Trade-offs: Major decisions justified

### Documentation
- [ ] No business leakage: Product content moved to PRD
- [ ] Technical precision: Specific versions, numbers, formats
- [ ] Diagrams included: Architecture visualized

## Common Anti-Patterns

| Anti-Pattern | Why It's Bad | Fix |
|--------------|--------------|-----|
| **Business Leakage** | TRD contains user stories | Replace with API specs |
| **Vague Tech Stack** | "Use a database" | Specify: "PostgreSQL 15" |
| **No Trade-offs** | Decisions without alternatives | Add options analysis |
| **No PRD Reference** | Orphan technical specs | Trace back to PRD |
| **Missing Error Handling** | Only happy path | Add error codes and recovery |
| **Over-engineering** | Gold-plating beyond requirements | Cut to MVP scope |

## Relationship to Other Documents

```
BRD (Business Requirements)
├── WHY — Business context, goals
├── WHO — Users, roles
└── High-level requirements

        ↓ PRD builds on BRD ↓

PRD (Product Requirements)
├── User Stories — Who needs what and why
├── Feature Specs — User-facing functionality
├── UX Flows — How users interact
└── Success Metrics — Measurable outcomes

        ↓ TRD builds on PRD ↓

TRD (This Document)
├── System Architecture — Components, layers, communication
├── API Specifications — Endpoints, requests, responses
├── Data Models — Database schema, entities
├── Technology Stack — Frameworks, libraries, versions
├── Deployment Architecture — Infrastructure, scaling
└── Security Design — Auth, encryption, compliance
```

---

## References

- `references/trd-template.md` — Full TRD template with all sections
- **prd-creator skill** — For creating the prerequisite PRD
- **brd-creator skill** — For creating the prerequisite BRD
