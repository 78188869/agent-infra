---
name: issue-decomposer
description: Use when users ask to "break down work into issues", "split this into tasks", "create issues for X", or when facing a large goal that needs decomposition into independently developable units. Also triggers when users reference requirements documents (BRD/PRD/TRD) and want to plan development work, or when they describe a cross-cutting concern (like "support local dev", "add auth", "migrate to X") that touches multiple modules. Use even when the user just says "let's plan the work" or "how should we tackle this".
---

# Issue Decomposer

Break down development work into well-scoped, independently deliverable GitHub issues.

## Core Principle

**Issues describe WHAT needs to be true (observable outcomes), not HOW to achieve it.**

The process has three phases:
1. **Discovery** — understand the full scope by exploring code and requirements
2. **Decomposition** — split into independent issues sized for 3-5 days of work
3. **Review** — validate boundaries, catch gaps, sanitize content before publishing

## Input Sources

The skill handles two starting points:

| Source | Starting Point | Key Action |
|--------|---------------|------------|
| **Goal-driven** | User describes a cross-cutting goal (e.g. "support local dev", "add auth") | Explore codebase to find ALL blockers, not just the obvious ones |
| **Doc-driven** | User has BRD/PRD/TRD and wants to plan development | Read docs to extract requirements, then map to codebase impact |

Both paths converge on the same decomposition workflow.

---

## Phase 1: Discovery

### 1A. Clarify the scope

Before diving in, confirm with the user:
- What is the end goal? (e.g. "local dev environment that mirrors production behavior")
- What is already working? (avoid re-discovering non-issues)
- Any hard constraints? (resource limits, must-use technologies, timeline)

### 1B. Systematic exploration

Explore the codebase methodically. The goal is to find ALL friction points, not just the ones the user mentioned.

**For goal-driven tasks**, trace the user's goal through every layer:

```
Config → Database → Business Logic → External Services → Container Runtime
```

At each layer ask:
- Does this layer depend on something external? What happens when that dependency is absent?
- Are there hardcoded assumptions (env vars, URLs, file paths, API dialects)?
- Are there different code paths that could be activated by configuration?

**For doc-driven tasks**, read the requirements documents first:
- BRD → extract business goals and success criteria
- PRD → extract user stories and feature requirements
- TRD → extract technical decisions and module boundaries

Then map each requirement to affected code modules. Use the project's knowledge module table (in `agent.md` or equivalent) to identify which files/systems each requirement touches.

### 1C. Look beyond what the user mentioned

Users typically identify the obvious blockers. Your job is to find the hidden ones. Common blind spots:

- **Wiring gaps**: startup code that doesn't initialize all required components
- **Config gaps**: hardcoded values that should be switchable but aren't
- **Addressing assumptions**: code that assumes specific network topologies (Pod IP, cluster DNS, etc.)
- **Credential passing**: assumptions about how secrets reach the runtime (K8s Secrets vs env vars vs files)
- **Shared state assumptions**: volume mounts, network namespaces, file locks between containers
- **Template/tag coupling**: database model tags tied to a specific database dialect

When you find additional blockers, present them grouped and prioritized. Don't overwhelm — lead with "here's what you already identified" then "here's what else I found".

---

## Phase 2: Decomposition

### 2A. Sizing: target 3-5 days of human work

Each issue should represent roughly **3-5 days of work for a competent engineer** familiar with the tech stack. This is the sweet spot: large enough to be meaningful, small enough to estimate accurately.

**Sizing signals:**

| Signal | Likely too small | Right size | Likely too big |
|--------|-----------------|------------|----------------|
| Description | "Change this tag", "Add this import" | "Support X as an alternative to Y" | "Rewrite the entire module" |
| Scope | Single line/file change | Cross-cutting but bounded concern | Unbounded across many modules |
| Testability | Trivial, one assertion | Multiple scenarios, edge cases | Can't list all scenarios |
| Dependencies | None, trivial | May depend on 1 other issue | Blocks many others |

When an issue feels bigger than 5 days, look for a seam to split it:

```
"Implement Docker executor"  (too big, ~10 days)
    → "Abstract executor interface"  (3 days, unblocks both paths)
    → "Implement Docker executor"    (5 days, depends on interface)
```

When an issue feels smaller than 2 days, consider merging it with a related concern — but only if they're tightly coupled. Don't merge independent concerns just to fill time.

### 2B. INVEST quality check

Apply the [INVEST criteria](https://agilealliance.org/glossary/invest/) to each draft issue:

| Letter | Criterion | How to check |
|--------|-----------|-------------|
| **I** | Independent | Can this be developed without waiting on other issues? (If not, document the blocker explicitly) |
| **N** | Negotiable | Does the AC describe outcomes, not a fixed solution? (The assignee should have room to choose their approach) |
| **V** | Valuable | Does completing this issue alone deliver visible progress toward the goal? |
| **E** | Estimable | Can a developer read this and give a rough time estimate? (If not, it may be too vague or too broad) |
| **S** | Small | Does it fit in ~3-5 days? (If not, split further) |
| **T** | Testable | Is every acceptance criterion verifiable by observing behavior? (If not, rewrite the AC) |

If an issue fails any criterion, revise it before presenting to the user.

### 2C. Grouping and dependency mapping

Group work into issues following these rules:

**Each issue should be:**
- **Coherent**: all parts relate to the same concern
- **Independently verifiable**: has its own testable acceptance criteria

**Dependency handling:**
- If A must finish before B can start → A blocks B, note in both issues
- If A and B can proceed in parallel → independent issues
- If A and B are tightly coupled and always change together → merge into one issue

### 2D. Writing acceptance criteria

Acceptance criteria describe **observable behavior from the outside**, not implementation choices.

**Bad** (describes implementation):
```
- Extract LogWriter interface from SLSClient
- Create FileWriter that implements LogWriter
- Inject FileWriter via config
```

**Good** (describes outcomes):
```
- Task execution events are persisted to local storage without any external log service configured
- Persisted logs survive process restarts and are queryable with standard text tools
- Does not affect production log service behavior
```

Why this matters: the developer (or agent) who picks up the issue should have freedom to choose the best implementation. They may find a simpler approach that still satisfies the criteria.

**AC self-check for each issue:**
- Can someone verify each criterion by observing system behavior, without reading code?
- Does it avoid prescribing specific classes, files, or patterns?
- Could two different implementations both satisfy these criteria?

### 2E. Present to user

Present all issues as a list with the dependency graph. For each issue show:
- Title
- One-line problem statement
- Acceptance criteria count (not full text — too verbose in list form)
- Dependencies (if any)

```
Issue 1: Config system for local dev (5 AC, no deps)
Issue 2: SQLite compatibility (4 AC, no deps)
Issue 3: Local file logging (5 AC, no deps)
Issue 4: Executor abstraction (4 AC, no deps) → blocks Issue 5
Issue 5: Docker executor (6 AC, depends on #4)
Issue 6: Sandbox images (4 AC, no deps)

Dependency graph:
1 ─┐
2 ─┤
3 ─┼──→ 7 (integration)
4 ──→ 5 ─┤
6 ───────┘
```

Wait for user feedback before proceeding to creation. They may want to merge, split, or reprioritize.

---

## Phase 3: Review and Create

### 3A. Boundary review

Before creating issues, do a final review of the full set:

**Check for gaps:**
- Are there any discovered blockers not covered by any issue?
- Does every issue's completion contribute to the stated goal?
- Is there a clear "done" state — what does it look like when all issues are closed?

**Check for overlap:**
- Do any two issues have identical or contradictory acceptance criteria?
- Is any work described in two different issues? (If so, pick one and reference it from the other)
- Are boundary issues clearly scoped? (e.g. "who owns the integration test — the component issue or the integration issue?")

**Check the critical path:**
- What's the longest dependency chain? Is it necessary?
- Can any dependency be broken by narrowing scope?
- Is the integration issue realistic, or is it a catch-all that should be split?

### 3B. Content security review

Issue content may be public. Sanitize before creating:

| Risk | Bad | Good |
|------|-----|------|
| Credential handling | "GIT_TOKEN injected via sed into URL" | "Authentication is required for repository access" |
| Internal architecture | "The bug is in job_manager.go line 42" | "Job lifecycle management assumes a specific container runtime" |
| Supply chain | "Install via curl \| bash" | "Include the required CLI tool in the runtime image" |
| Production weaknesses | "SLS is just stdout" | "External log service configuration" |
| Internal file paths | "Change enum tags in provider.go" | "Data model has database-specific type constraints" |

### 3C. Use the project's issue template

Check for existing templates in `.github/ISSUE_TEMPLATE/`. If they exist, use them. Typical fields:

- **Task Description**: one-sentence summary
- **What problem does this task solve?**: the pain point or gap
- **Knowledge Module References**: which docs to read (helps the assignee orient)
- **Acceptance Criteria**: observable behavior checklist
- **Additional Context**: dependencies on other issues, relevant background

### 3D. Create issues

When the user approves:

1. Use `gh issue create` with `--body-file` (not heredoc) to avoid shell escaping issues
2. Write each issue body to a temp file, then pass via `--body-file`
3. Clean up temp files after creation
4. Verify no duplicates were created (check `gh issue list`)
5. Report the created issue numbers and links

---

## Anti-patterns to avoid

- **Don't decompose by layer** (one issue for all models, one for all APIs) — this creates coupling. Decompose by concern instead.
- **Don't pre-assign implementation** — if you find yourself writing "add a new file called X", you're over-specifying. Describe what the system should do.
- **Don't skip discovery** — the user will mention 3-4 blockers; your job is to find the other 3-4 they missed.
- **Don't create giant integration issues** — "wire everything together" is valid as a final issue, but it should reference specific components, not be a catch-all.
- **Don't assume the user's list is complete** — always do your own exploration. The user knows their goal; you know the codebase. Both perspectives are needed.
- **Don't skip the boundary review** — the most common decomposition failure is overlap or gaps between issues, not bad individual issues.
