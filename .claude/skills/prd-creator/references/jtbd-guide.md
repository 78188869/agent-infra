# Jobs To Be Done (JTBD) Framework Guide

> A user-centered approach to understanding why users "hire" products to accomplish their goals.

---

## What is JTBD?

JTBD (Jobs To Be Done) is a framework that shifts focus from **user demographics** to **user motivations**. Instead of asking "Who is the user?", JTBD asks "What job is the user trying to get done?"

### Core Insight

> "People don't buy products; they hire them to do a job." — Clayton Christensen

Users have **jobs** (tasks, problems, goals) they need to accomplish. Your product is a **candidate** they might hire to do those jobs.

---

## JTBD vs Traditional Personas

| Traditional Persona | JTBD Approach |
|--------------------|---------------|
| "Alex is a 28-year-old developer" | "Alex needs to quickly spin up dev environments" |
| Demographics-focused | Motivation-focused |
| Who they are | What they're trying to achieve |
| Static attributes | Situational context |

**Key shift:** From "Who is this person?" to "Why would they use this product in this moment?"

---

## Jobs Statement Format

A well-formed Jobs Statement follows this structure:

```
When [situation],
I want to [motivation],
So that I can [expected outcome].
```

### Example

```
When I'm starting a new project and need consistent configuration,
I want to create tasks from pre-defined templates,
So that I can save time and avoid configuration errors.
```

### Components

| Component | Question | Example |
|-----------|----------|---------|
| **Situation** | When does this job arise? | "When I'm under deadline pressure..." |
| **Motivation** | What drives the user? | "I want to quickly reproduce a bug..." |
| **Outcome** | What does success look like? | "So I can fix it before the release" |

---

## JTBD Interview Questions

Use these during discovery to uncover true motivations:

### Situation Questions
- "When did you last [perform related task]?"
- "What was happening that made you decide to [take action]?"
- "Walk me through the last time you encountered [problem]."

### Motivation Questions
- "What were you trying to accomplish?"
- "Why was that important to you?"
- "What would have happened if you couldn't do that?"

### Outcome Questions
- "How did you know you were successful?"
- "What did success look like?"
- "How would you explain the value to a colleague?"

### Struggle Questions (most revealing)
- "What was the hardest part?"
- "What frustrated you most?"
- "What workarounds did you try?"

---

## Integrating JTBD with User Stories

### Standard User Story
```gherkin
As a developer,
I want to create tasks from templates,
So that I save time.
```

### JTBD-Enhanced User Story
```gherkin
As a developer under deadline pressure,
When I need to quickly spin up a new task,
I want to create tasks from templates,
So that I can avoid configuration errors and meet my deadline.

Acceptance Criteria:
✓ GIVEN I'm on the task creation page with time pressure
  WHEN I select a template
  THEN the form is pre-filled and I can submit in under 30 seconds
✓ GIVEN I select a template
  WHEN I review the pre-filled values
  THEN I see clear validation of any configuration issues
```

### Enhancement Checklist

| Element | Standard Story | JTBD-Enhanced |
|---------|---------------|---------------|
| **Context** | Implicit | Explicit situation |
| **Urgency** | Missing | Clear time/pressure indicator |
| **Emotional state** | Missing | Frustration, anxiety, or relief |
| **Outcome measure** | Vague | Specific success criteria |

---

## JTBD Types

### 1. Functional Jobs
Getting a practical task done.

```
When I need to deploy code to production,
I want a one-click deployment button,
So I can ship features faster.
```

### 2. Emotional Jobs
Achieving a feeling or emotional state.

```
When I'm presenting to executives,
I want confidence that my data is accurate,
So I feel prepared and professional.
```

### 3. Social Jobs
How users want to be perceived by others.

```
When I share my work with the team,
I want it to look polished and professional,
So my colleagues respect my attention to detail.
```

---

## Job Map Template

Use this to map the full job lifecycle:

| Phase | Questions to Ask | Example |
|-------|------------------|---------|
| **Define** | How do users decide they need to do this? | "I notice my code isn't working as expected" |
| **Locate** | Where do they look for solutions? | "I search internal docs or Stack Overflow" |
| **Select** | How do they choose a solution? | "I compare options based on speed and reliability" |
| **Prepare** | What setup is needed? | "I need to configure my environment first" |
| **Execute** | What's the core action? | "I run the debugging tool" |
| **Monitor** | How do they track progress? | "I watch the console output" |
| **Modify** | What adjustments might be needed? | "I tweak parameters if it doesn't work" |
| **Conclude** | How do they know it's done? | "I see the bug fixed and tests passing" |

---

## Competitors in JTBD

Your competitors aren't just similar products—they're **any alternative** users hire to do the job.

### For "Debug code quickly" job:

| Competitor Type | Example |
|-----------------|---------|
| **Direct** | Other debugging tools |
| **Indirect** | Stack Overflow, documentation |
| **Non-consumption** | Ignoring the bug, restarting |
| **Workaround** | Console.log debugging |

**Key insight:** If users are "hiring" console.log, your fancy debugger is competing against a free, familiar alternative.

---

## JTBD Discovery Checklist

Before finalizing PRD, ensure you can answer:

- [ ] **What job** is the user trying to get done?
- [ ] **When** does this job arise? (situation)
- [ ] **Why** is this job important to the user? (motivation)
- [ ] **What outcome** defines success? (expected result)
- [ ] **What alternatives** are they currently hiring?
- [ ] **What struggles** do they face with current solutions?
- [ ] **What would make** them switch to your solution?

---

## Quick Reference: JTBD One-Pager

```
┌─────────────────────────────────────────────────────────────┐
│                    JOBS TO BE DONE                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   WHEN [situation/trigger]                                  │
│   │                                                         │
│   │   "When I'm [doing something] and [problem occurs]..."  │
│   │                                                         │
│   ▼                                                         │
│   I WANT TO [motivation/action]                             │
│   │                                                         │
│   │   "I want to [do something about it]..."                │
│   │                                                         │
│   ▼                                                         │
│   SO THAT I CAN [expected outcome]                          │
│                                                             │
│       "So I can [achieve desired result]"                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Example: Developer Platform

### Job Statement
```
When I'm on-call and receive an alert at 2am,
I want to quickly identify the root cause,
So I can fix the issue and go back to sleep.
```

### Derived User Stories

**Story 1: Quick Access**
```gherkin
As an on-call developer,
When I receive an alert at 2am,
I want one-click access to relevant logs,
So I can start investigating within 30 seconds.
```

**Story 2: Context Preservation**
```gherkin
As an on-call developer,
When I'm debugging an incident,
I want to see related recent changes,
So I can quickly identify potential causes.
```

**Story 3: Guided Resolution**
```gherkin
As an on-call developer,
When I identify the issue,
I want suggested remediation steps,
So I can resolve it even when half-asleep.
```

---

## When to Use JTBD

| Use JTBD When | Skip JTBD When |
|---------------|----------------|
| Exploring new product ideas | Simple feature additions |
| User research phase | Well-understood requirements |
| Competitive analysis | Maintenance work |
| Defining MVP scope | Bug fixes |
| Writing positioning/messaging | Technical debt |

---

## References

- "Competing Against Luck" by Clayton Christensen
- "When Coffee and Kale Compete" by Alan Klement
- JTBD.org framework documentation
