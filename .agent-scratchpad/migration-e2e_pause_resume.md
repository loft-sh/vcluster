# Migration: e2e_pause_resume — test/e2e_pause_resume -> e2e-next

## Problem Summary

The old `test/e2e_pause_resume` suite contains a single `Describe` with one `It` block that tests the full vCluster pause/resume lifecycle via the CLI. It verifies vCluster pods are running, pauses the vCluster with `vcluster pause`, confirms all pods are removed, resumes with `vcluster resume`, and waits for all pods to return to Running state. The test uses empty `values.yaml` (default config) and only interacts with the host cluster client. Target: `e2e-next/test_core/lifecycle/test_pause_resume.go`.

## Bootstrap Requirements

- **Standard bootstrap works**: yes — `bootstrap 'core'` is sufficient. The test uses default vCluster configuration (no special Helm charts or external services). A new cluster definition using `DefaultVClusterYAML` is needed but requires no additional infrastructure beyond what bootstrap provides.

## Old -> New Translation

| Old Pattern | New Pattern |
|---|---|
| `framework.DefaultFramework` singleton | Context-based client accessors |
| `f.HostClient` | `cluster.KubeClientFrom(ctx, constants.GetHostClusterName())` |
| `f.VClusterName` | `clusters.PauseResumeVClusterName` (constant from cluster definition) |
| `f.VClusterNamespace` | `"vcluster-" + vClusterName` (derived from cluster name) |
| `f.Context` | `ctx` (Ginkgo spec function parameter) |
| `exec.Command("vcluster", ...)` | `exec.Command(filepath.Join(os.Getenv("GOBIN"), "vcluster"), ...)` (use GOBIN path, same as cluster provisioning) |
| `framework.ExpectNoError(err)` | `Expect(err).NotTo(HaveOccurred())` |
| `framework.ExpectEqual(true, len(pods.Items) > 0)` | `Expect(pods.Items).NotTo(BeEmpty())` |
| `framework.ExpectEqual(true, len(pods.Items) == 0)` | `Expect(pods.Items).To(BeEmpty())` |
| `wait.PollUntilContextTimeout(f.Context, time.Second, time.Minute*2, ...)` | `Eventually(...).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong)` |
| Hardcoded `time.Minute*2` timeout | `constants.PollingTimeoutLong` (120s — matches the old 2min timeout) |
| Hardcoded `time.Second` polling interval | `constants.PollingInterval` (2s) |

## Sub-Problems

### SP-0: [infra] Add PauseResumeVCluster cluster definition

**Old**: N/A
**Acceptance**: A `PauseResumeVCluster` definition exists in `e2e-next/clusters/clusters.go` using `DefaultVClusterYAML`, is wired into `SynchronizedBeforeSuite` in `e2e_suite_test.go`, and the test import is registered.
**Steps**:
1. Add `PauseResumeVClusterName` and `PauseResumeVCluster` to `e2e-next/clusters/clusters.go` using `DefaultVClusterYAML` and `DefaultVClusterOptions`, depending on `HostCluster`
2. Add `clusters.PauseResumeVCluster.Setup` to the `AllConcurrent` block in `e2e-next/e2e_suite_test.go`
3. Add blank import `_ "github.com/loft-sh/vcluster/e2e-next/test_core/lifecycle"` to `e2e_suite_test.go`

### SP-1: [migrate] Pause and resume vCluster via CLI

**Old**: `It("run vcluster pause and vcluster resume", ...)`
**Acceptance**: A single `It` block verifies the full pause/resume lifecycle: pods running → pause → pods gone → resume → pods running. Uses host client only. CLI invoked via `exec.Command` with GOBIN path.
**Steps**:
1. Create `e2e-next/test_core/lifecycle/test_pause_resume.go` with package `lifecycle`
2. Translate the single `It` block using the translation table, wrapping the post-resume poll in `Eventually`
3. Use `By()` closures for each phase: verify running, pause, verify paused, resume, verify resumed
4. Use `--context` flag with `"kind-" + constants.GetHostClusterName()` in vcluster CLI commands to ensure correct kubeconfig context (no hardcoded context names)

### SP-2: [cleanup] Remove old test/e2e_pause_resume

**Old**: N/A
**Acceptance**: The directory `test/e2e_pause_resume/` and all its contents are deleted. No other old tests are affected.
**Steps**:
1. Delete `test/e2e_pause_resume/` directory entirely (it contains only the migrated test)
2. Verify no other test files import from `test/e2e_pause_resume/`

## Helper Consolidation

No consolidation needed. The old test has no inline helpers, and the new test requires no operations that overlap with existing inline helpers in other `e2e-next/` test files. The `endpointIPs` helper in `test_servicesync.go` is unrelated to pause/resume functionality.

## Structure

```
Describe("Pause and resume vCluster", labels.Core,
    cluster.Use(clusters.PauseResumeVCluster),
    cluster.Use(clusters.HostCluster))
  BeforeEach:
    - hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
  It("pauses and resumes the vCluster via CLI")
```

No `Ordered` needed — there is a single `It` block, so no spec-to-spec dependencies exist.

## Design Decisions

1. **Separate file in new `lifecycle` package**: Pause/resume is a lifecycle operation, not a sync or deploy concern. Creating `test_core/lifecycle/` keeps the test organized by concern. If more lifecycle tests are added later (e.g., delete, upgrade), they belong in the same package.

2. **Own vCluster definition (`PauseResumeVCluster`)**: The test pauses the vCluster, which kills all its pods and the background proxy. This makes the vCluster unusable by any other concurrent test. A dedicated vCluster definition prevents interference. It uses `DefaultVClusterYAML` since the old test uses empty values.

3. **No reconnection after resume**: The old test only uses the host client after resume (polling host pods). It never reconnects to the vCluster API server. The migrated test preserves this behavior — no `connectVCluster()` pattern needed. If future tests need to verify vCluster API server readiness after resume, they should add reconnection as a separate concern.

4. **CLI via `exec.Command` with GOBIN path**: The old test calls `vcluster pause/resume` as a subprocess. The new test does the same but uses `filepath.Join(os.Getenv("GOBIN"), "vcluster")` to locate the binary, consistent with how `providervcluster.WithPath()` references it in cluster definitions. This tests the real CLI code path.

5. **No `labels.PR`**: The old `e2e_pause_resume` suite does not appear in any CI workflow matrix, so it was not running on every PR. The migrated test carries only `labels.Core`.

6. **`PollingTimeoutLong` for post-resume wait**: The old test uses `time.Minute*2` (120s) for waiting after resume. `constants.PollingTimeoutLong` is exactly 120s, making it the correct replacement.

7. **`--context` flag for CLI commands**: The vcluster CLI needs the correct kubeconfig context. Use `"kind-" + constants.GetHostClusterName()` to derive it dynamically, avoiding hardcoded cluster names per quality checklist item 10.

## Allowed Directories

- e2e-next/clusters
- e2e-next/test_core/lifecycle
- e2e-next/e2e_suite_test.go
- test/e2e_pause_resume

## Validation

```bash
just -f Justfile.agent test-focus "core" "Pause and resume vCluster"
```

Then verify against `.claude/rules/e2e-quality-checklist.md` (auto-loaded).
