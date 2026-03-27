---
name: prd-creator
description: Use when users ask to "create a PRD", "write product requirements document", "document user stories", "define features", "write acceptance criteria", or need help with user-facing specifications AFTER having a BRD. Also triggers when converting business requirements into product specs, mapping user journeys, defining UX flows, setting success metrics, or when stakeholders need alignment on what users will experience. Make sure to use this skill whenever the user mentions "user stories", "personas", "acceptance criteria", "success metrics", "HEART framework", or wants to translate business needs into concrete features, even if they don't explicitly ask for a "PRD". Always check if a BRD exists first — if not, recommend creating one.
---

# PRD Creator

Create comprehensive Product Requirements Documents (PRDs) that define user stories, feature specifications, UX flows, and success metrics — building on BRD's business context to answer WHAT users need.

## Core Principle

**PRD answers "WHAT" for users — not "HOW" to implement.**

- **WHAT**: User stories, feature specifications, UX flows, success metrics
- **NOT**: API design, database schema, code architecture, technology stack

```
BRD → PRD → TRD
WHY/WHO → WHAT (user-facing) → HOW (technical)
```

**Key insight:** PRD bridges BRD's business context to TRD's technical implementation. If you're writing technical details → STOP → That belongs in TRD.

## When to Use This Skill

**Use PRD when:**
- BRD exists and business context is clear
- Need to translate business needs into user-facing features
- Defining user stories and acceptance criteria
- Mapping user journeys and UX flows
- Setting measurable success metrics

**Skip PRD when:**
- No BRD exists (create BRD first using brd-creator skill)
- Just a small feature or bug fix
- Clear, well-understood product requirements

## Phase 1: Input Validation (BRD Check)

**CRITICAL: PRD requires BRD as input.**

Before proceeding, ask: "Do you have an existing BRD for this project?"

- **If YES** → Extract key information from BRD
- **If NO** → Recommend creating BRD first using brd-creator skill

### Extract from BRD

| BRD Section | PRD Input | Purpose |
|-------------|-----------|---------|
| Business Goals | Goals → Success Metrics | Ensure metrics measure business outcomes |
| Users & Roles | Personas | Build user stories for each persona |
| Functional Requirements | Feature specs | Translate to user-facing features |
| Constraints | Scope boundaries | Define in-scope vs out-of-scope |
| Stakeholders | Open Questions owners | Identify who to consult |

### BRD-PRD Traceability

Every PRD element should trace back to BRD. If you can't trace a story to a BRD requirement → ask: Is this truly needed for MVP?

## Phase 2: Discovery & Design Clarification

### JTBD Questions (Jobs To Be Done)
*See `references/jtbd-guide.md` for detailed framework explanation.*

1. **Situation**: When does this need arise for users? What triggers it?
2. **Motivation**: Why would users "hire" this product? What are they trying to achieve?
3. **Outcome**: What does success look like from the user's perspective?
4. **Current alternatives**: How do users solve this problem today? What frustrates them about current solutions?
5. **Emotional context**: How do users feel when facing this problem? (stressed, confused, frustrated?)

### User Journey Questions
1. What does a typical user session look like?
2. What's the first thing users see/do?
3. What are the most common tasks? (80/20 rule)
4. Where do users get confused or frustrated today?
5. What would make users say "wow, this is great"?

### Feature Prioritization Questions
1. If we could only ship ONE feature, what would it be?
2. What features are "nice to have" vs "must have"?
3. What features differentiate us from alternatives?

### UX Design Questions
1. What's the primary entry point for users?
2. How many steps should the core task take? (Fewer = better)
3. What feedback do users need during tasks?
4. How should errors be communicated?

### Design Trade-off Questions
1. Power vs Simplicity: Expert features or beginner-friendly?
2. Flexibility vs Consistency: Customizable or standardized?
3. Speed vs Accuracy: Fast with occasional errors or slow but correct?

## Phase 3: Generate PRD

**Read the template from `references/prd-template.md` to generate the PRD document.**

The template includes all sections: Executive Summary, Problem & Opportunity, User Personas & Scenarios, User Stories (Gherkin format), UX & Design Requirements, Success Metrics (HEART framework), Scope, Risks & Mitigation, Dependencies & Assumptions, Open Questions, and Revision History.

**Fill in the template with information gathered from:**
- BRD (business goals, users, requirements)
- Discovery questions (user journeys, priorities, trade-offs)
- User scenario mapping

## Phase 4: Logic & Completeness Review

After drafting the PRD, perform these checks BEFORE finalizing:

### 4.1 BRD-PRD Traceability
- Each user story traces to a BRD requirement?
- Each persona has relevant user stories?
- No orphan stories (no BRD connection)?

### 4.2 Metric-Goal Alignment
- Each business goal has at least one HEART metric?
- Each HEART metric supports a business goal?

### 4.3 No Technical Leakage
**RED FLAG TERMS (should NOT appear in PRD):**
- API, REST, GraphQL, endpoint, database, schema, SQL
- Microservice, Kubernetes, JWT, OAuth, React, Vue, Redis

If found → Replace with user-facing language (e.g., "Users must be authenticated" instead of "Use JWT")

### 4.4 Completeness Checklist
- [ ] Executive Summary: Clear 3-sentence overview
- [ ] Personas: Each has name, role, pain points
- [ ] User Stories: Gherkin format with acceptance criteria
- [ ] UX Flows: Entry points and paths shown
- [ ] Success Metrics: HEART framework with targets
- [ ] Scope: Clear in/out boundaries
- [ ] Open Questions: Owners and due dates assigned

## Phase 5: Iteration & Refinement

PRD is a living document. Iterate based on:
- User feedback from testing/research
- Stakeholder review comments
- Scope changes requested
- BRD updates (PRD must stay aligned)

### Sign-off Process
Before marking PRD as "approved":
- [ ] All stakeholders have reviewed
- [ ] All open questions resolved
- [ ] All Phase 4 checks pass
- [ ] Ready for TRD creation

## Critical Rules

### Rule 1: No Technical Implementation
```
❌ BAD: "Use REST API with JWT authentication"
✅ GOOD: "Users must be securely authenticated"

❌ BAD: "Store data in PostgreSQL with indexing"
✅ GOOD: "User data must be persisted and retrievable"

❌ BAD: "Implement using React with Redux"
✅ GOOD: "UI must respond to user input within 100ms"
```

### Rule 2: User-First Language
```
❌ BAD: "The system will..."
✅ GOOD: "Users can..."

❌ BAD: "Backend will process..."
✅ GOOD: "When users submit..."

❌ BAD: "API returns..."
✅ GOOD: "Users will see..."
```

### Rule 3: Every Feature Needs User Value
For each feature, ask: WHO uses this? WHAT do they do? WHY do they care?

If you can't answer all three → The feature may not be needed.

## Self-Review Checklist

Before finalizing, verify:

### BRD Alignment
- [ ] BRD exists: PRD is built on documented business requirements
- [ ] Traceability: Each story traces to BRD requirement
- [ ] Goal alignment: Metrics measure BRD goals

### User Focus
- [ ] Personas defined: Clear who we're building for
- [ ] User Stories complete: Gherkin format with acceptance criteria
- [ ] User value clear: Every feature has "So that..." clause

### Quality
- [ ] No technical leakage: Implementation details moved to TRD
- [ ] User-first language: No system-centric phrasing
- [ ] Measurable metrics: HEART framework with targets

## Common Anti-Patterns

| Anti-Pattern | Why It's Bad | Fix |
|--------------|--------------|-----|
| **Technical Leakage** | PRD contains API/DB details | Replace with user-facing language |
| **Missing User Value** | Features without "So that..." | Add user benefit or remove feature |
| **Vague Stories** | "As a user I want a button" | Specify: what button, why, what happens |
| **No BRD Reference** | Orphan requirements | Trace back to BRD or remove |
| **Unmeasurable Metrics** | "Improve user experience" | Use HEART: "Task completion rate ≥95%" |
| **Missing Error Paths** | Only happy path documented | Add error scenarios and recovery |

## Example: User Story Format

```gherkin
As a developer,
I want to create a new task from a template,
So that I don't have to configure the same settings every time.

Acceptance Criteria:
✓ GIVEN I'm on the task creation page WHEN I select a template THEN the form is pre-filled with template values
✓ GIVEN I've modified template values WHEN I submit THEN the task is created with my custom values
✓ GIVEN the template is invalid WHEN I submit THEN I see a clear error message explaining what's wrong
```

## Relationship to Other Documents

```
BRD (Business Requirements)
├── WHY — Business context, goals
├── WHO — Users, roles
└── High-level requirements

        ↓ PRD builds on BRD ↓

PRD (This Document)
├── User Stories — Who needs what and why
├── Feature Specs — User-facing functionality
├── UX Flows — How users interact
├── Success Metrics — Measurable outcomes
└── Design Decisions — UI/UX choices

        ↓ TRD builds on PRD ↓

TRD (Technical Requirements)
├── Architecture design
├── API specifications
├── Database schema
└── Technology stack
```

---

## References

- `references/prd-template.md` — Full PRD template with all sections
- `references/jtbd-guide.md` — Jobs To Be Done framework guide for understanding user motivations
- **brd-creator skill** — For creating the prerequisite BRD
