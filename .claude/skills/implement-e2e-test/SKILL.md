---
name: implement-e2e-test
description: Implement an e2e test from an existing test plan. Use when asked to "implement the plan", "execute the test plan", given a plan file path, or a ticket ID with an existing plan in .agent-scratchpad/. Takes a plan produced by /migrate-e2e-test or /decompose-and-plan and writes the actual test code.
---

# Implement E2E Test

## Overview

Take a test plan from `.agent-scratchpad/` and implement it — write test code, verify against the running vcluster, and run a quality gate. This skill writes code; the planning skills (`/migrate-e2e-test`, `/decompose-and-plan`) produce the plans it consumes.

## Arguments

```
/implement-e2e-test <plan-file-or-ticket-id>
```

- `plan-file-or-ticket-id`: path to a `.agent-scratchpad/*.md` file, or a ticket ID (e.g., `ENG-9832`) to match against existing plan filenames
- If omitted, list `.agent-scratchpad/*.md` and ask the user to pick
- `--auto`: Skip user approval checkpoints. Proceed without confirmation after the execution plan summary (Phase 0 step 7) and before `[infra]` sub-problems (Phase 1). Safety gates (failure escalation, scope guard, quality checklist) remain active.

## Phase 0: Setup

1. **Resolve plan file**
   - If a path: use directly
   - If a ticket ID: glob `.agent-scratchpad/*{ticket-id}*` (case-insensitive)
   - If omitted: list `.agent-scratchpad/*.md`, present to user, wait for selection

2. **Parse plan** — extract:
   - Plan type: `migration` (file starts with `# Migration:`) or `decomposition` (file starts with `# Decomposition:`)
   - Sub-problems list (SP-0, SP-1, ...) with types (`[infra]`, `[migrate]`, `[test]`, `[impl]`, `[cleanup]`)
   - Validation command (from the `## Validation` section)
   - For migrations: translation table (`## Old -> New Translation`) and old test path (`## Problem Summary`)
   - For decompositions: affected source files

3. **Read source material**
   - Migration: read the old test file (source of truth for behavior)
   - Decomposition: read affected source files listed in the plan

4. **Read conventions**
   - `.claude/e2e-tdd-workflow.md` — dev loop mechanics
   - `.claude/rules/e2e-conventions.md` — e2e-next conventions and patterns

5. **Preflight check**
   All `just -f Justfile.agent` commands require `AGENT_SESSION` to be set. Derive it from the plan filename (e.g., `migration-test-sync` -> `AGENT_SESSION=sync`). Export it and use it for all subsequent commands in this session. Also activate the scope guard by writing the colon-separated paths from `## Allowed Directories` to `.agent-scratchpad/.scope-guard` — the `e2e-scope-guard` hook reads this file to enforce boundaries.
   ```bash
   export AGENT_SESSION="<derived-from-plan>"
   echo "<colon-separated directories from ## Allowed Directories>" > .agent-scratchpad/.scope-guard-"$AGENT_SESSION"
   just -f Justfile.agent preflight
   ```
   If the cluster is not ready, auto-bootstrap using the label from the plan's `## Validation` section:
   ```bash
   just -f Justfile.agent bootstrap "<label>"
   ```
   Bootstrap creates a Kind cluster, builds the vcluster image, loads it into Kind, and creates vcluster instances. Inform the user it's running, then proceed once complete.

6. **Initialize state file** — after preflight succeeds, before presenting the execution plan:
   - Write `.agent-scratchpad/state-<AGENT_SESSION>.md` with:
     - Session identity (plan file, AGENT_SESSION, label filter, start timestamp)
     - All SPs listed as PENDING under `## Progress`
     - Current kubecontext from `kubectl config current-context` in `## Cluster State`
     - Empty `## Cluster Deviations from Baseline` and `## Known Infrastructure Gaps` sections
     - `## Next Steps` pointing to SP-0
   - The `require-agent-session.sh` hook writes the `session_id=AGENT_SESSION` mapping to `.agent-scratchpad/.active-sessions` automatically on the first `just -f Justfile.agent` call. No manual step needed here.

7. **Present execution plan** — summarize to user:
   - Plan type and source
   - Number of sub-problems by type
   - Estimated workflow (e.g., "3 migrations then 1 cleanup")
   - **Unless `--auto`**, wait for user confirmation before proceeding

## Phase 1: Serial Sub-Problem Execution

Process sub-problems **in dependency order** (infra -> migrate/test/impl -> cleanup), **one at a time**.

### `[infra]` sub-problems

1. **Unless `--auto`**, present the required work to the user (new builder, label, helper) and wait for user confirmation. With `--auto`, log the infra change being made and proceed.
2. Implement the infrastructure code
4. Compile-check: `just -f Justfile.agent compile-check`
5. If syncer code changed: `just -f Justfile.agent rebuild`

### `[migrate]` sub-problems

1. Write the new test using:
   - The translation table from the plan
   - The old test as source of truth for behavior
   - Conventions from `.claude/rules/e2e-conventions.md`
2. Run the test: `just -f Justfile.agent test "<label-filter>"`
3. If it fails:
   - Investigate using the failure ladder from `.claude/e2e-tdd-workflow.md` section 5
   - Fix and re-run (max 3 retries)
4. Confirm green

### `[test]` sub-problems (decomposition — RED phase)

1. Write the failing test — it should assert the desired behavior that doesn't exist yet
2. Run: `just -f Justfile.agent test "<label-filter>"`
3. Confirm it **fails** with the expected error
4. If it passes: the test isn't testing the right thing — fix the assertion and re-run
5. Do NOT proceed to implementation until the test fails as expected

### `[impl]` sub-problems (decomposition — GREEN phase)

1. Write the minimum implementation to make the failing test pass
2. Rebuild: `just -f Justfile.agent rebuild`
3. Run: `just -f Justfile.agent test "<label-filter>"`
4. If it fails:
   - Investigate using the failure ladder
   - Fix and re-run (max 3 retries)
5. Confirm green

### `[cleanup]` sub-problems

1. Verify all prior sub-problems passed
2. Delete old test files (migration) or temporary scaffolding
3. Compile-check: `just -f Justfile.agent compile-check`

### After each sub-problem

- **Update state file**: mark the completed SP as PASSED or FAILED, update `## Progress -> Current SP` and `## Next Steps`
- **Cluster deviations**: any time a cluster change is made (helm upgrade, resource deletion, config patch, manual label/annotation, etc.), append a timestamped entry to `## Cluster Deviations from Baseline`:
  ```
  - [<ISO timestamp>] <what changed> (<persistent | ephemeral — reverted by <mechanism>>)
  ```
- **Infrastructure gaps**: when a workaround is discovered and applied, also add an entry to `## Known Infrastructure Gaps`:
  ```
  - <gap description>
    Workaround: <what to do each time>
  ```
- Report status: which SP completed, pass/fail, brief summary

**On failure — follow this protocol in order:**

1. Run `just -f Justfile.agent report`. Read the `ErrorAt` file and line. State the root cause in one sentence before attempting any fix.

2. **Retry 1**: Use the failure ladder (section 5 of `.claude/e2e-tdd-workflow.md`) to diagnose. Apply a fix targeting the specific root cause. Re-run.
   - Check syncer pod logs: `kubectl logs -n <vcluster-namespace> -l app=vcluster --tail=100`
   - Check vcluster events: `kubectl get events -n <vcluster-namespace> --sort-by=.lastTimestamp`

3. **Before Retry 2**: Explicitly state:
   - "Root cause of retry 1 failure: ..."
   - "Is this the same root cause as the original failure? yes/no"
   - If **yes** (same root cause, different fix): **stop and escalate immediately** — repeating the same diagnosis wastes cycles.
   - If **no** (a different root cause emerged): proceed with retry 2.

4. **After Retry 2**: Stop and escalate regardless. Do not attempt a 3rd fix.

**Escalation report must include:**
- Original failure (ErrorAt + message)
- Retry 1: root cause and what was changed
- Retry 2: root cause (if different) and what was changed
- Suggested next steps (what you would try with more context)

## Phase 2: Quality Gate

1. **Run the full test suite** (all labels, not just the plan's label) to verify no regressions:
   ```bash
   just -f Justfile.agent test-all
   ```
   All specs must pass. This catches regressions in tests unrelated to the current migration.

2. **Run lint** to catch formatting and static analysis violations:
   ```bash
   just -f Justfile.agent lint ./e2e-next/...
   ```

3. **Quality checklist** — walk through all items from `.claude/rules/e2e-quality-checklist.md` (auto-loaded) against the code you wrote. For each item, cite the specific code that passes or explain why it's N/A — do not rubber-stamp.

4. **Fix violations** — if any item fails, fix the code, re-run lint, and re-run the full test suite

## Phase 3: Summary

1. **Close state file**:
   - Add `## Status: COMPLETED` at the top of `.agent-scratchpad/state-<AGENT_SESSION>.md`
   - Remove the session entry from `.agent-scratchpad/.active-sessions` (delete the line matching `*=<AGENT_SESSION>`)

2. Report to the user:
   - **Files created**: list with paths
   - **Files modified**: list with paths
   - **Files deleted**: list with paths
   - **Test results**: pass/fail per sub-problem, final suite run result
   - **Quality checklist**: all items passed (or note any exceptions with justification)

Remind the user: review with `git diff` and commit when satisfied.

Deactivate the scope guard now that the session is complete:
```bash
rm -f .agent-scratchpad/.scope-guard-"$AGENT_SESSION"
```

## Guardrails

- **Never auto-commit** — the user decides when to commit
- **Never skip sub-problems** — execute all in order, even if they seem trivial
- **Never proceed past failures** — fix or escalate, don't skip
- **Always confirm `[infra]` with user** (**unless `--auto`**) — infrastructure changes affect other tests
- **Rebuild only when syncer code changed** — for test-only changes, skip `rebuild` and just run `test`
- **After any context compaction**: immediately read `.agent-scratchpad/state-<AGENT_SESSION>.md` (if `AGENT_SESSION` is set). Treat its `## Cluster Deviations from Baseline` and `## Known Infrastructure Gaps` sections as ground truth. Do NOT rediscover or re-investigate problems already documented there.
- **Scope is locked to the plan** — `.agent-scratchpad/.scope-guard` is written at Phase 0 with the directories from `## Allowed Directories`. The `e2e-scope-guard` hook reads that file and blocks any `Edit` or `Write` outside those directories. To edit an out-of-scope file, stop, present the proposed change and reason to the user, update `## Allowed Directories` in the plan, rewrite `.agent-scratchpad/.scope-guard`, then proceed.
- **Scope guard**: If you find yourself about to edit a file outside `e2e-next/`, `test/`, `.agent-scratchpad/`, `Justfile.agent`, or `.claude/`, STOP. Present the proposed change to the user and wait for explicit confirmation before proceeding.
- **Old test is source of truth for behavior, not for code patterns** — the plan summarizes behavior, but the old test code is definitive for *what* is tested. However, do NOT copy code patterns (client usage, namespace lookups, assertion styles) from old tests. Old tests often have latent bugs that happen to pass (e.g., checking the wrong namespace, using deprecated APIs). Always rewrite using e2e-next conventions from `.claude/rules/e2e-conventions.md`.
- **Verify namespace/type assumptions independently** — when the old test uses `f.HostClient.Get(ctx, key, &obj)`, check which client and type the e2e-next framework expects. See the client table in `.claude/references/e2e-old-to-new-mapping.md`.
- **Respect RED/GREEN phases for decomposition** — don't implement before confirming the test fails
- **Use the failure investigation ladder** from `.claude/e2e-tdd-workflow.md` section 5 — follow steps 1-6 in order
- **Always compile-check before running tests** — `just -f Justfile.agent compile-check` catches type/import errors in 5s vs 2min in a test run
- **Max 2 retries per sub-problem** — if retry 2 fails with the same root cause as retry 1, escalate after retry 1 instead
