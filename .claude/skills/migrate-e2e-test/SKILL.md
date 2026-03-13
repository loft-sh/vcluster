---
name: migrate-e2e-test
description: Plan migration of an old e2e test suite to the e2e-next framework. Use when given a Linear ticket, file path, or test suite name that needs migrating from test/e2e* to e2e-next/. Triggers on requests like "migrate this test", "port test_e2e to e2e-next", "plan test migration for ENG-1234". Produces a migration plan that an implementing agent can follow.
---

# Migrate E2E Test

## Overview

Produce a migration plan that translates an old `test/e2e*` test to the `e2e-next/` framework. The plan is a concise document an implementing agent follows — it defines *what* to migrate and *why* certain decisions were made, never *how* at the code level.

**Core principle**: The behavior is already implemented and tested. This is a translation, not TDD. The cycle is: translate -> run green -> verify quality.

## Arguments

```
/migrate-e2e-test [--auto] <source>
```

- `source`: Linear ticket ID (e.g., `ENG-9832`), old test file path, or `test/e2e*` directory name
- If no source given, ask the user
- `--auto`: Skip user approval checkpoints and write the plan directly. When set, do not pause for confirmation after the problem summary (Phase 1) or after the decomposition (Phase 3) — proceed straight through to writing the output file.

## Workflow

### Phase 1: Gather Context

1. **If Linear ticket**: Read issue via `mcp__linear__get_issue`. Extract old file path and target location.
2. **If file path or directory**: Use directly.
3. Read the old test file(s) — this is the source of truth for behavior.
4. Read the target location in `e2e-next/` — check for existing files in the same package.
5. Spawn an **Explore** subagent to inventory available infrastructure:
   - Scan `e2e-next/clusters/` for cluster definitions and `cluster.KubeClientFrom` / `cluster.CurrentKubeClientFrom` accessors
   - Which labels in `e2e-next/labels/labels.go` apply
   - Which client accessors are needed — use this table:
     - `cluster.KubeClientFrom(ctx, clusterName)` — host cluster typed client
     - `cluster.CurrentKubeClientFrom(ctx)` — current vcluster typed client
     - `cluster.CurrentClusterClientFrom(ctx)` — current cluster CR client
   - Whether the old test uses any helpers that have no e2e-next equivalent
   - **Inline helper scan**: Search existing `e2e-next/` test files for inline helper functions (file-level funcs that are not in `setup/`) that perform the same operations the old test needs. Record function name, file location, and what it does. This feeds into the Helper Consolidation step in Phase 3.
   - **Same-label setup scan**: Look at `e2e-next/test_core/` and `e2e-next/test_deploy/` patterns. Find all `e2e-next/test_*/**/*.go` files whose `Describe` carries any of the same labels as the target test. Read each one **completely**. Extract the full setup chain — `BeforeEach`, `BeforeAll`, `BeforeSuite`, cluster creation, any non-trivial environment configuration. Record this verbatim in the plan under `## Design Decisions` as "Existing setup pattern for label X." If the new test's setup will differ, explain why in the same section.
   - **External dependency scan (simplified)**: If the old test installs Helm charts or external services (CSI, ingress-nginx, metrics-server, etc.), flag each as an `[infra]` sub-problem. Check how they're provisioned in `.github/workflows/e2e.yaml` and the old test's `values.yaml`.
   - **Code-under-test scan**: Read the syncer/controller code the test exercises (e.g., `pkg/controllers/syncer/`, `pkg/util/translate/`). Understanding the code path prevents debugging issues that are really API misuse or misunderstood behavior.
   - Check `Justfile` recipes and `.github/workflows/e2e.yaml` matrix for available test infrastructure
6. **PR label inference**: Check whether the old test's directory appears in the matrix in `.github/workflows/e2e.yaml`. If it does, the migrated test must carry `labels.PR` — it was running on every PR before and must continue to do so after migration. Record the result (PR-required or not) in the plan's translation table.
7. Summarize the migration in one paragraph: what the old test covers, how many `It` blocks, what behaviors. **Unless `--auto`**, present to user for confirmation before proceeding.

### Phase 2: Build Translation Table

Map every old-framework pattern used in the test to its e2e-next equivalent. Only include patterns actually used — not a generic reference table.

Format:
```markdown
| Old Pattern | New Pattern |
|---|---|
| `f.HostClient.CoreV1()...` | `cluster.KubeClientFrom(ctx, constants.GetHostClusterName()).CoreV1()...` |
| `f.VClusterClient.CoreV1()...` | `cluster.CurrentKubeClientFrom(ctx).CoreV1()...` |
```

Flag any old patterns that have **no direct equivalent** — these become infra sub-problems.

### Phase 3: Decompose

Each old `It()` block maps to one `[migrate]` sub-problem. Apply these rules:

**Decision tree per old It():**
```
Does the old It() test exactly one behavior?
+- YES -> One [migrate] sub-problem
+- NO, tests multiple things -> Split into multiple It() blocks in the new test
+- Multiple old It() blocks test the same thing differently -> Merge into one
```

**Sub-problem types:**
- `[infra]` — new builders, labels, or helpers that don't exist yet
- `[consolidate]` — extract inline helpers from existing tests into shared `setup/` packages (see Helper Consolidation below)
- `[migrate]` — rewrite one old `It()` block in the new framework
- `[cleanup]` — remove the specific old `It()` blocks (and surrounding `Context` if now empty) that were migrated in this plan. **Never delete tests that were migrated in a different PR** — even if they live in the same file or directory. If removing the migrated tests leaves the old file empty (no remaining `It()` blocks), delete the file. If deleting all files in a directory leaves it empty, delete the directory and its CI matrix entry. This keeps each migration PR's deletion traceable to exactly the tests it migrated.

**Ordering:**
- `[infra]` first (unblocks everything)
- `[consolidate]` second (ensures helpers are shared before specs use them)
- `[migrate]` in any order (they're independent)
- `[cleanup]` last

Each sub-problem entry:

```markdown
### SP-{N}: [{type}] {title}

**Old**: `It("{old test name}", ...)` (or "N/A" for infra/cleanup)
**Acceptance**: {one sentence — what must be true when done}
**Steps**: {numbered list of logical steps, 3-5 max}
```

**Setup Factoring** — before deciding on `BeforeEach` vs inline setup, build a setup matrix:

```markdown
| Setup Step              | SP-0 | SP-1 | SP-2 | ... |
|-------------------------|------|------|------|-----|
| Create namespace        | Y    | Y    | Y    |     |
| Create configmap        |      | Y    | Y    |     |
```

Rules:
1. Start from the convention default: **use `BeforeEach`**.
2. Any step present in **all** specs -> `BeforeEach` (shared setup).
3. Any step present in **some** specs -> inline in the `It` block (spec-specific setup).
4. Only choose "no shared `BeforeEach`" if the intersection is empty.

Include the matrix and the resulting split in the plan's `## Design Decisions` section.

**Helper Consolidation** — after identifying `[infra]` sub-problems, check whether any inline helpers in existing `e2e-next/` test files duplicate logic the new test needs. Use the inline helper scan from Phase 1 step 5.

Decision rules:
1. If an inline helper in another test file performs the **same operation** the new test needs -> create a `[consolidate]` sub-problem to extract it into `setup/`.
2. If the new test needs a helper that already exists **inline in one other file** -> `[consolidate]` extracts it to `setup/` and updates both the existing file and the new test to use the shared version.
3. If the new test needs a helper that already exists **inline in multiple files** -> `[consolidate]` extracts to `setup/` and updates all call sites.
4. If the new test needs a helper with **no existing inline equivalent** and it's only used by this one test -> keep it inline, no `[consolidate]` needed.
5. If the same operation appears in **2+ `It` blocks within the new test file itself** (e.g., identical cleanup sequences, identical assertion chains), extract it into a file-local helper function. If the operation is generic enough to benefit other tests (cleanup patterns like annotation-stripping + delete, or status polling), promote it to `setup/` instead.

Each `[consolidate]` sub-problem entry:

```markdown
### SP-{N}: [consolidate] Extract {helper name} to setup/{package}

**Found in**: `{file path}:{function name}` (and any other locations)
**Operation**: {what the helper does}
**Target**: `e2e-next/setup/{package}/{file}.go`
**Steps**:
1. Create the shared helper in `setup/` following the `setup.Func` / functional-options pattern
2. Update the existing test file(s) to use the shared helper
3. Run the affected tests to confirm they still pass (see **Affected tests** below)
4. The new `[migrate]` sub-problems will use the shared helper directly

**Affected tests**: {list the `just -f Justfile.agent test-focus {label} "{focus}"` commands to run for each modified test file}
```

Include a `## Helper Consolidation` section in the plan output that lists the scan results: which inline helpers were found, which are being consolidated, and which the new test will keep inline.

**Unless `--auto`**, present the decomposition to the user and wait for confirmation before proceeding to Phase 4.

### Phase 4: Output

The output path is `.agent-scratchpad/migration-{identifier}.md` where `{identifier}` is the ticket ID or a slugified description.

**Before writing**, check if the file already exists. If it does:
1. Read the existing file
2. **If `--auto`**: overwrite without asking
3. **Otherwise**: Show the user a brief summary of what's already there and ask: **overwrite** or **abort**

The plan document structure:

```markdown
# Migration: {ticket} — {old path} -> e2e-next

## Problem Summary
{one paragraph: what the old test covers, how many behaviors, target file}

## Bootstrap Requirements
- **Standard bootstrap works**: yes / no
- If **no**: provide a **complete self-contained provisioning plan**. External services must be installed in `BeforeAll` using Helm or equivalent and torn down in `AfterAll`. Never defer this to the implementing agent — the provisioning design must be in the plan.

{Examples:
- "yes — `bootstrap 'core'` is sufficient."
- "no — tests require CSI volume snapshots. Provisioning plan: `[infra]` sub-problem to add `just setup-csi-volume-snapshots` to bootstrap recipe."}

## Old -> New Translation
{table from Phase 2 — only patterns actually used}

## Sub-Problems
{SP-0..N from Phase 3}

## Helper Consolidation
{Results of the inline helper scan from Phase 1.
 For each inline helper found in existing e2e-next/ test files:
 - File, function name, what it does
 - Decision: consolidate to setup/ (-> [consolidate] SP) or keep inline (with reason)
 If no inline helpers overlap with the new test's needs, state "No consolidation needed."}

## Structure
{Describe/Context/It tree outline — names and labels only, no code}
{For each Ordered container: name the specific spec-to-spec side-effect dependency that requires it.
 If you cannot name one, use BeforeEach instead. See the Ordered vs BeforeEach table in e2e-conventions.md.}

## Design Decisions
{numbered list: why you chose separate file vs same file, what old helpers were dropped and why, etc.}

## Allowed Directories
{List every directory the implementation will touch — one relative path per line, narrowest applicable directory.}
- {e.g. e2e-next/test_core}
- {e.g. e2e/test_sync}

## Validation
{the exact `just -f Justfile.agent test` or `just -f Justfile.agent test-focus` command to run}
Then verify against `.claude/rules/e2e-quality-checklist.md` (auto-loaded).
```

Present a summary to the user with the file path.

## What belongs in the plan

- Problem summary (the north star for the implementing agent)
- Translation table (old -> new pattern mappings actually used)
- Sub-problems with acceptance criteria and steps
- Helper consolidation scan results and decisions
- Structural outline (Describe/Context/It tree with labels)
- Design decisions (choices that aren't obvious from the old test)
- Validation command and quality gate reference

## What does NOT belong in the plan

- **Code.** No code skeletons, no full implementations, no import lists. The implementing agent reads the old test, the translation table, and the framework references — it writes the code.
- **Pre-checked quality checklists.** Quality is verified against the implementation, not against imagined code. The plan points to the quality reference; the implementing agent checks it as a final gate.
- **Step-by-step implementation instructions.** The plan says "create file X with these It blocks." The e2e-tdd-workflow and quality reference handle the how.

## Guardrails

- **Always confirm** the problem summary before decomposing (skip with `--auto`)
- **Always confirm** the decomposition before writing the plan (skip with `--auto`)
- **Never write code** in the plan — not even "helpful" skeletons
- **Never pre-check the quality checklist** — it's a post-implementation verification gate
- **Never skip reading the old test** — it's the source of truth for behavior
- If an old helper has no e2e-next equivalent, flag it as an `[infra]` sub-problem
- If the old test uses hardcoded resource names, note the switch to random suffixes in the translation table
- If the target file already exists with other tests, decide whether to add a new `Context` or create a separate file — document the decision
- **Never default to `Ordered`** — the old test using `BeforeEach`/`AfterEach` does NOT mean the new test needs `Ordered`. Apply the decision table from `e2e-conventions.md`: only use `Ordered` when a spec depends on a prior spec's side effect. Shared expensive setup (cluster, namespace) is NOT a justification — `BeforeEach` pays the cost per spec but enables parallelization.
- **Never write `Bootstrap Requirements: no`** without a complete self-contained provisioning design in the same section. Labels alone are not a provisioning plan — they only control when tests run, not how the environment is set up.
