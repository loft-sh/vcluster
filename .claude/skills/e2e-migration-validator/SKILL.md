---
name: e2e-migration-validator
description: >
  Use this skill whenever a user wants to validate, review, or audit e2e tests that have been
  migrated from an old framework (typically in a `e2e-next/` directory) to the new `e2e-next/`
  framework. Triggers include: "review my migrated test", "validate this e2e test", "check
  my test migration", "did I migrate this correctly", "review e2e PR", "check test coverage
  after migration", or any request involving comparing old vs new e2e test files across the
  vcluster, vcluster-pro, or loft-enterprise repos. Always use this skill when an e2e test
  file or PR from one of these repos is provided for review — even if the user just says
  "take a look at this test".
---

# E2E Test Migration Validator

A skill for exhaustively reviewing e2e tests migrated from the old framework (`next/` directory)
to the new framework (`e2e-next/` directory) across the `vcluster`, `vcluster-pro`, and
`loft-enterprise` repositories.

---

## Inputs

The user will provide one or more of:
- **Uploaded file(s)**: The new migrated test file, and optionally the original test file
- **GitHub PR link**: A pull request URL containing the migrated test(s)
- **Pasted code**: Snippets of old and/or new test code inline in the chat

If only the new test is provided (no original), you must locate the original yourself using
the local repo tools (see [Repo Access](#repo-access)).

---

## Repo Access

Use the local repo MCP tools to inspect implementation details and locate source files.
All three repos may be relevant depending on the test subject:

| Repo | Use when |
|------|----------|
| `vcluster` | Core vcluster functionality, open-source features |
| `vcluster-pro` | Pro/enterprise vcluster features |
| `loft-enterprise` | Loft platform, multi-tenancy, UI-level features |

When reviewing a test, **always look up the actual implementation** being tested in the
relevant repo before drawing conclusions about correctness or coverage. Do not rely solely
on what the test itself asserts.

---

## Review Workflow

Follow these steps in order for every migration review:

### Step 1 — Locate the Original Test

If the original (`next/`) test was not provided:
1. Search the appropriate repo(s) for the original test file by name or describe pattern
2. Read it fully before proceeding
3. Note the framework, helper utilities, and assertion style used

### Step 2 — Locate the Implementation Under Test

Using the repo MCP tools:
1. Find the source code that the test is exercising (controllers, handlers, CLI commands, API routes, UI components, etc.)
2. Read the implementation thoroughly — **line by line** if needed
3. Build a mental model of:
   - All code paths (happy path, error paths, edge cases)
   - All configurable inputs / flags / options
   - Side effects and state mutations
   - Interactions with other subsystems

### Step 3 — Framework Migration Check

Verify the structural correctness of the migration:

- [ ] Test file is in the correct `e2e-next/` directory structure
- [ ] Imports use the new framework's helpers/fixtures (not the old ones)
- [ ] Test lifecycle hooks (`beforeAll`, `afterAll`, `beforeEach`, `afterEach`) are correctly
      translated — no leftover old-framework teardown patterns
- [ ] Setup and teardown create/delete resources correctly and completely
- [ ] Cluster/namespace/resource provisioning uses the new framework's APIs
- [ ] Any shared utilities (custom matchers, wait helpers, retry logic) are migrated or
      replaced with new equivalents — not just copied verbatim
- [ ] Test isolation is maintained (tests do not share mutable state across runs)
- [ ] Framework-specific APIs (e.g. `page`, `context`, `request`) are used correctly

### Step 4 — Correctness Check

For each test case in the migrated file:

1. **Does the test actually test what it claims to test?**
   - Read the test name / description and compare against what is asserted
   - Flag any tests where the name and assertions are misaligned
2. **Are the assertions meaningful?**
   - Look for trivially-passing assertions (e.g. `expect(x).toBeDefined()` with no deeper check)
   - Look for assertions that would pass even if the feature is broken
   - Look for missing negative assertions (e.g. testing that a resource *was* created but not
     that an invalid input *was rejected*)
3. **Is the test deterministic?**
   - Flag race conditions, missing `await`, missing retries on flaky operations
   - Flag hardcoded timeouts that may not hold in CI
4. **Is the test environment realistic?**
   - Does the test reflect how the feature is actually used in production?
   - Are there unrealistic shortcuts (e.g. bypassing auth, using superuser where a normal user
     should be tested)?

### Step 5 — Exhaustive Coverage Analysis

This is the most important step. Compare the implementation (Step 2) against the test suite.

For **every distinct behavior, code path, and configurable option** in the implementation:

1. Check whether it is covered by at least one test
2. If not covered, classify the gap:
   - 🔴 **Critical gap** — a regression here would not be caught at all
   - 🟡 **Important gap** — partial coverage, but a significant edge case is missed
   - 🟢 **Minor gap** — low-risk or already covered by unit tests

For each gap, suggest a concrete additional test case including:
- What scenario it covers
- What inputs/state are needed
- What assertion(s) should be made

**Common gap categories to check explicitly:**
- Error handling and invalid inputs
- Permission / RBAC boundary conditions
- Resource limit enforcement
- Concurrent operations
- Feature flags / configuration variants
- Upgrade/downgrade paths (if applicable)
- Interaction with other features (cross-feature integration)
- Cleanup/teardown correctness (does the feature leave no orphaned resources?)

### Step 6 — Code Quality Review

- Naming: are test and variable names descriptive and consistent with the new framework's conventions?
- DRY: is there duplicated setup that should be extracted into a shared fixture or helper?
- Readability: would a new engineer understand what is being tested and why?
- Comments: are non-obvious steps explained?
- Dead code: are there commented-out blocks, unused imports, or leftover debug statements?

---

## Output Format

Produce **all three** of the following output sections:

---

### 1. Inline Code Analysis

Go through the migrated test file and annotate specific lines or blocks with numbered
comments. Format as a fenced code block with inline `// [N] ...` annotations, followed by
a numbered legend below it explaining each annotation.

Example:
```typescript
// [1] Missing: should also assert that the old resource was deleted
await expect(newResource).toBeVisible();

// [2] Race condition: no retry/wait before asserting pod readiness
expect(pod.status).toBe('Running');
```

**Legend:**
1. 🔴 Critical gap — deletion is not verified; the old resource could persist undetected
2. 🟡 Flakiness risk — pod status may not be `Running` immediately; use `waitForCondition`

---

### 2. Checklist Report

```
## Migration Checklist — <test file name>

### Framework Migration
- [x] Correct directory structure
- [x] New framework imports
- [ ] ⚠️  Teardown uses old-framework pattern on line 42

### Correctness
- [x] Assertions are meaningful
- [ ] ⚠️  Test "should reject invalid config" passes even with valid config (line 87)

### Coverage Gaps
- 🔴 No test for RBAC: non-admin user attempting privileged operation
- 🟡 No test for concurrent resource creation
- 🟢 No test for audit log output (low risk)

### Code Quality
- [ ] ⚠️  Duplicate setup block (lines 12–25 and 61–74) — extract to beforeEach
- [x] Naming is clear and consistent
```

---

### 3. PR Review Comment

A copy-paste-ready GitHub PR review comment in Markdown. Structure:

```
## E2E Migration Review

**Overall:** ✅ Approved with suggestions / ⚠️ Changes requested / ❌ Major issues found

### Summary
<2–3 sentence high-level assessment>

### Issues

#### 🔴 Must Fix
- ...

#### 🟡 Should Fix
- ...

#### 🟢 Nice to Have
- ...

### Suggested Additional Tests
- **Test:** `<descriptive test name>`
  **Scenario:** <what to set up>
  **Assert:** <what to check>

<repeat for each suggested test>

### Nits
- ...
```

---

## Framework Notes

> **For the reviewer using this skill:** If you know the specific old and new frameworks
> involved in this migration, add them here so Claude can apply framework-specific checks.
>
> Example entries:
> - Old: `@vcluster/e2e` custom test runner → New: Playwright + `@vcluster/e2e-next` fixtures
> - Old: Ginkgo/Gomega (Go) → New: Playwright (TypeScript)
> - Old: Custom shell-script harness → New: Playwright with kubectl helpers
>
> Until this section is filled in, Claude will infer the frameworks from the provided files.

---

## Tips for High-Quality Reviews

- **Always read the implementation first.** A test that looks complete may miss half the
  code paths if you haven't read the source.
- **Think adversarially.** For each assertion, ask: "How could this pass even if the feature
  is broken?"
- **Don't skip cleanup verification.** Many e2e bugs are leaked resources, not failed
  assertions.
- **Flag flakiness proactively.** A flaky test is worse than no test — it erodes trust in
  the suite.
- **Be specific in suggestions.** Don't say "add more tests." Say exactly what scenario,
  what inputs, and what assertion.
