# E2E TDD Workflow for vCluster

All commands use `just -f Justfile.agent`. Alias for convenience: `alias ja='just -f Justfile.agent'`

## 1. Pre-flight

```bash
just -f Justfile.agent preflight
```

If cluster is missing, bootstrap from scratch (one-time, ~90s):
```bash
just -f Justfile.agent bootstrap "<label>"
```
Bootstrap builds the image, creates the kind cluster, and runs setup for the specified label only. Pass the same label you'll use for testing (e.g., `"core"`) to avoid setting up resources for all tests.

If pod exists but is not Ready, check the namespace events and pod logs for the specific vcluster.

### How `bootstrap` selects clusters

`bootstrap "<label>"` only creates clusters referenced by tests matching that label. See [`.claude/references/e2e-framework-conditional-deps.md`](.claude/references/e2e-framework-conditional-deps.md) for the full mechanism.

## 2. Choose Test Strategy

| Signal | Strategy | Cycle Time |
|--------|----------|------------|
| Pure logic, no cluster needed (transforms, validation) | Unit test only | ~5s |
| vCluster syncer operations, resource sync, multi-namespace flows | E2E test | ~50s |
| Complex feature touching both logic and integration | Unit for fast loop + E2E for validation | ~5s + ~50s |

Rule: if the code under test needs the Kubernetes API, real vcluster instances, or sync verification, use E2E. Otherwise, unit test.

## 3. The TDD Loop

### Always compile-check before running tests

```bash
just -f Justfile.agent compile-check   # ~20s — catches type/import errors + go vet immediately
```

This saves 2+ minutes vs discovering errors after a full test run. Do this after every code change, before any `test` or `push`.

**Note:** `compile-check` runs `go build` + `go vet` only. It does NOT run the full `golangci-lint` suite (formatting, static analysis, 28+ linters). Before declaring work done, always also run:

```bash
just -f Justfile.agent lint ./e2e-next/...
```

### Scoping test runs: focus first, verify at the end

During development, run only the specific test you're working on:

```bash
just -f Justfile.agent test-focus "<label>" "<context name>"
```

Example: `just -f Justfile.agent test-focus "sync" "pod sync"`

Once your test is green, run the **full label group** to verify you haven't broken siblings:

```bash
just -f Justfile.agent test "<label>"
```

This catches regressions in related tests that share the same Describe or setup.

### Two iteration speeds

| Change type | Command | Cycle time |
|------------|---------|------------|
| Test-only (no syncer code modified) | `test` / `test-focus` directly | ~50s |
| Syncer changes | `push` = compile check + image rebuild + kind load | ~15-45s |

### RED — write failing test, run it

```bash
just -f Justfile.agent compile-check
just -f Justfile.agent test-focus "<label>" "<context>"
```

Confirm exit code is non-zero. If it passes, the test isn't testing the right thing.

### GREEN — write implementation, push, test

```bash
just -f Justfile.agent compile-check                # catch errors early (~20s)
just -f Justfile.agent push                         # rebuild + kind load (~15-45s)
just -f Justfile.agent test-focus "<label>" "<context>"
```

On failure, investigate:
```bash
just -f Justfile.agent report       # extract failure summary from the report JSON
```
Then read the file and line from `ErrorAt`. See section 5 for the full investigation ladder.

### REFACTOR — clean up, confirm green

```bash
just -f Justfile.agent compile-check
just -f Justfile.agent push
just -f Justfile.agent test-focus "<label>" "<context>"
```

### Final verification — run the full label group

After all changes are green with `test-focus`, run the full label group once:

```bash
just -f Justfile.agent test "<label>"
```

This confirms your changes don't break sibling tests under the same label.

### Test-only changes (no syncer code modified)

Skip the push — the vcluster stays running:
```bash
just -f Justfile.agent compile-check
just -f Justfile.agent test-focus "<label>" "<context>"
```

> **With AGENT_SESSION**: "skip push" only works if the session-tagged Docker image already exists in kind. If it doesn't, tag an existing one: `docker tag ghcr.io/loft-sh/vcluster:dev-next ghcr.io/loft-sh/vcluster:dev-next-$AGENT_SESSION && kind load docker-image ghcr.io/loft-sh/vcluster:dev-next-$AGENT_SESSION --name ${KIND_NAME:-kind-cluster}`

## 4. Writing New E2E Tests

### File placement
- Place in `e2e-next/test_<area>/` (e.g., `test_sync/`, `test_core/`)
- Package name matches sibling files in that directory
- If new package, add blank import in `e2e-next/e2e_suite_test.go`

### Labels
- Every `Describe` gets at least one label from `e2e-next/labels/labels.go`
- Labels are simple strings like `"pr"`, `"core"`, `"sync"` — no slash wrappers
- Add `labels.PR` to `It` blocks that should gate PRs
- Add `labels.NonDefault` for tests requiring special infrastructure
- If new resource type, add a label constant in `labels.go`

Small-scope labels for fast iteration:
- `pr` — PR-gating tests
- `core` — core functionality
- `sync` — resource sync tests

### Structure
```go
var _ = Describe("resource sync",
    labels.Core, labels.Sync,
    func() {
        Context("behavior", Ordered, func() {
            var (
                resourceName string
            )

            cluster.Use(clusters.XVCluster)

            BeforeAll(func(ctx context.Context) {
                suffix := random.RandomString(6)
                resourceName = "descriptive-prefix-" + suffix
            })

            It("should sync X", labels.PR, func(ctx context.Context) {
                By("Describing the step")
                vClient := cluster.VClusterClient(ctx)
                Eventually(func(g Gomega) {
                    // g.Expect(...)
                    _ = vClient
                }).WithPolling(constants.PollingInterval).
                  WithTimeout(constants.PollingTimeoutShort).
                  Should(Succeed())
            })
        })
    },
)
```

### Rules
- `Eventually` with `func(g Gomega)` pattern — never boolean callbacks
- Timeouts from `constants/timeouts.go` — never ad-hoc durations
- No `time.Sleep` — always poll for state
- `DeferCleanup` at creation time — cleanup tolerates already-deleted (`client.IgnoreNotFound`)
- Random suffix via `random.RandomString(6)` — one per `Ordered` container
- Clients from context — never global singletons
- `By()` wraps every logical step in `It` blocks
- Each `It` is independent — no spec depends on side effects from another

## 5. Failure Investigation

Follow top-to-bottom. Stop when root cause is found.

| Step | Command / Action | Tokens |
|------|-----------------|--------|
| 1. Failure summary | `just -f Justfile.agent report` | ~200 |
| 2. Read test code | Read file at `ErrorAt` line | ~150 |
| 3. Pod logs (targeted) | See pod log patterns below | ~300 |
| 4. Namespace events | `jq '.[0].SpecReports[] \| select(.State == "failed") \| .ReportEntries[] \| select(.Name == "Namespace Events") \| .Value.Representation' /tmp/e2e-report*.json` | ~250 |
| 5. Live cluster | `kubectl --context kind-kind-cluster get/describe <resource>` | ~300 |
| 6. Verbose re-run | `ginkgo -timeout=0 -v --no-color --focus='test name' ./e2e-next -- --vcluster-image="ghcr.io/loft-sh/vcluster:dev-next" --teardown=false 2>&1 \| tail -100` | ~3000 |

### Pod log patterns (step 3)

Pod logs are the fastest way to diagnose syncer failures, resource sync errors, and controller issues. **Never read the full log file** — always grep for the resource name.

Check syncer pod logs directly:
```bash
# Logs from the syncer pod in a specific vcluster namespace
kubectl --context kind-kind-cluster logs -n vcluster-<name> -l app=vcluster -c syncer --tail=100

# Grep for specific resource or error patterns
kubectl --context kind-kind-cluster logs -n vcluster-<name> -l app=vcluster -c syncer | grep -iE 'error|failed|panic' | head -20

# Resource-specific sync issues
kubectl --context kind-kind-cluster logs -n vcluster-<name> -l app=vcluster -c syncer | grep '<resource-name>' | head -10
```

Common root causes revealed by pod logs:
- **Sync errors** → Check if the resource type is registered and the syncer is configured for it
- **"not found" errors** → Race condition during startup, usually transient
- **Panic/crash loops** → Check pod events and container restart count

Bail-out: if steps 1-5 don't clarify after two iterations, use step 6. If still unclear, escalate to the user.

Note: `report` reads the JSON report from the most recent `test` run. If you run `report` without running `test` first, it shows stale results.

## 6. Parallel Agents

By default, all agents share the same image tag (`dev-next`) and report file (`/tmp/e2e-report.json`). Set `AGENT_SESSION` to isolate these per agent:

```bash
export AGENT_SESSION="feat-sync"   # any unique string
just -f Justfile.agent push
just -f Justfile.agent test "sync"
just -f Justfile.agent report
```

This gives each agent its own:
- Image tag: `ghcr.io/loft-sh/vcluster:dev-next-feat-sync`
- Report file: `/tmp/e2e-report-feat-sync.json`

All agents share the same Kind cluster, so `push` (rebuild + kind load) operations should be serialized to avoid image tag conflicts.

## 7. Context Efficiency Rules

1. Never read the report JSON raw (~1.3MB). Use `just -f Justfile.agent report`.
2. Label filters — run only relevant specs. Succinct mode output is already minimal (one line per run).
3. Compile check before push — `just -f Justfile.agent compile-check` catches errors early.
4. Read only the failing test file, not the whole suite.
5. Use `./e2e-next` (no `...` suffix) — avoids walking sub-packages.

## Quick Reference

| Command | What | Time |
|---------|------|------|
| `just -f Justfile.agent preflight` | Check cluster and vcluster readiness | ~5s |
| `just -f Justfile.agent bootstrap "<label>"` | First-time setup (build + cluster + label-scoped setup) | ~90s |
| `just -f Justfile.agent compile-check` | Type errors + go vet (all packages) | ~20s |
| `just -f Justfile.agent rebuild` | Full image rebuild + kind load | ~15-45s |
| `just -f Justfile.agent push` | Compile check + rebuild | ~15-45s (cached) |
| `just -f Justfile.agent test "label"` | Run all tests for a label (succinct + JSON report) | ~50s |
| `just -f Justfile.agent test-focus "label" "context"` | Run a single focused test (fast iteration) | ~50s |
| `just -f Justfile.agent teardown "label"` | Tear down vclusters for label (keeps kind cluster) | ~30s |
| `just -f Justfile.agent report` | Failure summary from last run | ~5s |
