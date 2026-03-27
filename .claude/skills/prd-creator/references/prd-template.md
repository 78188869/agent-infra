# PRD Template Reference

> **When to use:** Read this file when you're ready to generate the PRD document after completing discovery.

---

# 📄 PRD: [Product/Feature Name]

> **Version:** v0.1-draft | **Date:** YYYY-MM-DD | **Status:** 🟡 Draft
> **BRD Reference:** [Link to BRD document]

---

## 📋 Table of Contents

1. [Executive Summary](#executive-summary)
2. [Problem & Opportunity](#problem--opportunity)
3. [User Personas & Scenarios](#user-personas--scenarios)
4. [User Stories](#user-stories)
5. [UX & Design Requirements](#ux--design-requirements)
6. [Success Metrics (HEART)](#-success-metrics-heart)
7. [Scope](#-scope)
8. [Risks & Mitigation](#-risks--mitigation)
9. [Dependencies & Assumptions](#-dependencies--assumptions)
10. [Open Questions](#-open-questions)
11. [Revision History](#-revision-history)

---

## 📋 Executive Summary

> [2-3 sentence overview of what we're building and why]

**Key Points:**
- [Point 1: What problem we're solving]
- [Point 2: Who we're solving it for]
- [Point 3: What success looks like]

**MVP Target:** [Date or phase]

---

## 🎯 Problem & Opportunity

### Current State
[Describe the current situation/process from user perspective]

### Problem Statement
[Clear articulation of the user problem - from BRD]

### Opportunity
[Why now? What opportunity does this address?]

### Goals
| Goal | Priority | BRD Reference |
|------|----------|---------------|
| [Goal 1] | P0 | BRD §1.2 |
| [Goal 2] | P1 | BRD §1.2 |
| [Goal 3] | P2 | BRD §1.2 |

---

## 👥 User Personas & Scenarios

### Personas

| Persona | Role | Description | Key Pain Points | MVP? |
|---------|------|-------------|-----------------|------|
| [Name] | [Job Title] | [Who they are, context] | [What frustrates them] | ✓ |
| [Name] | [Job Title] | [Who they are, context] | [What frustrates them] | ✓ |
| [Name] | [Job Title] | [Who they are, context] | [What frustrates them] | ✗ v1.1 |

### User Scenarios

**Scenario 1: [Scenario Name]**
- **Actor:** [Persona]
- **Trigger:** [What starts this scenario]
- **Preconditions:** [What must be true before starting]
- **Steps:**
  1. [Step 1]
  2. [Step 2]
  3. [Step 3]
- **Outcome:** [Expected result]
- **Error Paths:** [What could go wrong and how to recover]

**Scenario 2: [Scenario Name]**
- **Actor:** [Persona]
- **Trigger:** [What starts this scenario]
- **Preconditions:** [What must be true before starting]
- **Steps:**
  1. [Step 1]
  2. [Step 2]
- **Outcome:** [Expected result]
- **Error Paths:** [What could go wrong and how to recover]

---

## 📖 User Stories

### 🎯 Epic 1: [Epic Name]
[One-line description of this epic - what user capability does it enable?]

**Story 1.1: [Story Title]**
```gherkin
As a [user type],
I want to [action],
So that [benefit/value].

Acceptance Criteria:
✓ GIVEN [context] WHEN [action] THEN [outcome]
✓ GIVEN [context] WHEN [action] THEN [outcome]
✓ GIVEN [context] WHEN [action] THEN [outcome]
```
**Priority:** P0 | **Effort:** S/M/L | **Dependencies:** None | **BRD Ref:** §3.1

---

**Story 1.2: [Story Title]**
```gherkin
As a [user type],
I want to [action],
So that [benefit/value].

Acceptance Criteria:
✓ GIVEN [context] WHEN [action] THEN [outcome]
✓ GIVEN [context] WHEN [action] THEN [outcome]
```
**Priority:** P1 | **Effort:** M | **Dependencies:** Story 1.1 | **BRD Ref:** §3.2

---

### 🎯 Epic 2: [Epic Name]
[One-line description]

**Story 2.1: [Story Title]**
...

---

## 🎨 UX & Design Requirements

### User Flow Overview
```
[Entry Point] → [Step 1] → [Step 2] → [Decision] → [Outcome]
                                    ↓
                              [Alternative Path]
```

### Interaction Requirements
| Interaction | Behavior | Priority |
|------------|----------|----------|
| [Interaction 1] | [How it should behave from user's perspective] | P0 |
| [Interaction 2] | [How it should behave from user's perspective] | P1 |
| [Interaction 3] | [How it should behave from user's perspective] | P2 |

### Response Time Requirements
| Action | Expected Response | Max Acceptable |
|--------|-------------------|----------------|
| [Page load] | < 1 second | < 3 seconds |
| [Form submission] | < 500ms | < 2 seconds |
| [Search/query] | < 200ms | < 1 second |

### Accessibility Requirements
- [ ] Keyboard navigation support
- [ ] Screen reader compatible
- [ ] Color contrast WCAG AA
- [ ] [Other requirements specific to product]

### Error Handling
| Error Scenario | User Message | Recovery Action |
|---------------|--------------|-----------------|
| [Error 1] | [What user sees - friendly message] | [How to recover - user action] |
| [Error 2] | [What user sees - friendly message] | [How to recover - user action] |

---

## 📊 Success Metrics (HEART)

> Using HEART Framework - optimized for internal tools & platforms

### H - Happiness (满意度)
| Metric | Current | Target | Measurement |
|--------|---------|--------|-------------|
| User Satisfaction Score | - | ≥4.0/5.0 | Quarterly survey |
| Support Tickets | [Current] | ↓30% | Ticket system |
| NPS (if applicable) | - | ≥40 | Survey |

### E - Engagement (参与度)
| Metric | Current | Target | Measurement |
|--------|---------|--------|-------------|
| Daily Active Users | - | [Target] | Analytics |
| Feature Usage Rate | - | ≥70% | Analytics |
| Session Duration | - | [Target] | Analytics |

### A - Adoption (采用率)
| Metric | Current | Target | Measurement |
|--------|---------|--------|-------------|
| Team Adoption | 0% | 100% by [Date] | Onboarding tracking |
| New Feature Adoption | - | ≥80% within 30 days | Feature flags |
| Onboarding Completion | - | ≥90% | Funnel analysis |

### R - Retention (留存率)
| Metric | Current | Target | Measurement |
|--------|---------|--------|-------------|
| Weekly Return Rate | - | ≥85% | Cohort analysis |
| Churn to Old Process | - | <5% | Process audit |
| Feature Stickiness | - | ≥60% | DAU/MAU ratio |

### T - Task Success (任务成功率) ⭐ Core for Internal Tools
| Metric | Current | Target | Measurement |
|--------|---------|--------|-------------|
| Task Completion Rate | [Current]% | ≥95% | Funnel analysis |
| Time to Complete | [Current] min | ↓50% | Time tracking |
| Error Rate | [Current]% | <2% | Error logging |
| First-Time Success Rate | - | ≥80% | User testing |

---

## 🔍 Scope

### ✅ In Scope (MVP)
| Feature | Priority | User Value | BRD Reference |
|---------|----------|------------|---------------|
| [Feature 1] | P0 | [Why users need this] | BRD §3.1 |
| [Feature 2] | P0 | [Why users need this] | BRD §3.2 |
| [Feature 3] | P1 | [Why users need this] | BRD §3.3 |

### ❌ Out of Scope
| Item | Reason | Future Consideration |
|------|--------|---------------------|
| [Item 1] | [Why not now - business reason, not technical] | [When to reconsider] |
| [Item 2] | [Why not now - business reason, not technical] | [When to reconsider] |

### 🔮 Future Considerations (Backlog)
| Feature | Priority | Dependencies |
|---------|----------|--------------|
| [Feature X] | P2 | [What needs to happen first] |
| [Feature Y] | P3 | [What needs to happen first] |

---

## ⚠️ Risks & Mitigation

| Risk | Likelihood | Impact | Mitigation | Owner |
|------|------------|--------|------------|-------|
| [Risk 1] | 🔴 High | 🔴 High | [How to address - user/product perspective] | [Name] |
| [Risk 2] | 🟡 Medium | 🟡 Medium | [How to address - user/product perspective] | [Name] |
| [Risk 3] | 🟢 Low | 🟡 Medium | [How to address - user/product perspective] | [Name] |

---

## 📎 Dependencies & Assumptions

### Dependencies
| Dependency | Type | Status | Impact if Blocked |
|------------|------|--------|-------------------|
| [Dependency 1] | External/Internal | ⏳ Pending | [User impact] |
| [Dependency 2] | External/Internal | ✅ Available | [User impact] |

### Assumptions
- [ ] [Assumption 1 - what we assume about users/behavior]
- [ ] [Assumption 2 - what we assume about data/content]
- [ ] [Assumption 3 - what we assume about timeline/resources]
- [ ] [Assumption 4 - what we assume about training/adoption]

---

## ❓ Open Questions

| # | Question | Owner | Due Date | Status |
|---|----------|-------|----------|--------|
| 1 | [Question about user needs/behavior] | [Name] | [Date] | 🔴 Open |
| 2 | [Question about priority/trade-offs] | [Name] | [Date] | 🟡 In Progress |
| 3 | [Question about scope] | [Name] | [Date] | 🟢 Resolved |

---

## 📝 Revision History

| Version | Date | Changes | Author | Status |
|---------|------|---------|--------|--------|
| v0.1-draft | YYYY-MM-DD | Initial draft | - | 🟡 Draft |
| v0.2-draft | YYYY-MM-DD | [Changes] | - | 🟡 Draft |
| v1.0 | YYYY-MM-DD | Approved | - | 🟢 Approved |
