---
paths:
  - "e2e-next/**/*.go"
---
<!-- Generic core (items 1-8): e2e-tdd-workflow plugin references/e2e-quality-checklist-core.md -->

# E2E Test Quality Checklist (e2e-next)

Every item is **pass/fail**. A test must pass all 10 items.

---

## 1. Cleanup Tolerates Already-Deleted Resources

Cleanup code must not fail if the resource was already removed (e.g., cascade-deleted
by a parent). Use `client.IgnoreNotFound` or equivalent.

```go
// PASS
DeferCleanup(func(ctx context.Context) {
    err := client.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
    Expect(clientpkg.IgnoreNotFound(err)).To(Succeed())
})

// FAIL — hard-fails if resource is already gone
DeferCleanup(func(ctx context.Context) {
    Expect(client.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})).To(Succeed())
})
```

---

## 2. DeferCleanup Registered Before Subsequent Assertions

`DeferCleanup` must be the very next statement after verifying creation succeeded.
If an assertion between creation and cleanup registration fails, the resource leaks
and can block teardown of the entire suite.

```go
// PASS — cleanup registered immediately, before any further assertions
_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
Expect(err).NotTo(HaveOccurred())
DeferCleanup(func(ctx context.Context) {
    err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
    Expect(clientpkg.IgnoreNotFound(err)).To(Succeed())
})

// now safe to assert further
Expect(ns.Name).To(Equal(expectedName))

// FAIL — assertion before cleanup registration; if it fails, namespace leaks
_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
Expect(err).NotTo(HaveOccurred())
Expect(ns.Name).To(Equal(expectedName))
DeferCleanup(func(ctx context.Context) { /* ... */ })
```

---

## 3. Timeout Constants

Use the predefined constants. Never invent ad-hoc durations.

| Constant | Duration | Use for |
|----------|----------|---------|
| `PollingTimeoutVeryShort` | 5s | Immediate state checks |
| `PollingTimeoutShort` | 20s | Quick API operations |
| `PollingTimeout` | 60s | Standard operations |
| `PollingTimeoutLong` | 120s | Resource creation |
| `PollingTimeoutVeryLong` | 300s | vCluster startup, cluster creation |

---

## 4. No Cluster-Scoped Singletons

Tests must not assume they are the only consumer of a cluster-scoped resource
(e.g., a ClusterRole with a fixed name). If a cluster-scoped resource is needed,
its name must include the suffix.

---

## 5. Resource Naming for Debuggability

### 5.1 Single suffix per `Ordered` container

One call to `random.RandomString(6)`, stored in a variable, reused for all resource
names in the container. Do not generate a separate suffix per resource.

```go
// PASS — single suffix, all resources traceable
BeforeAll(func(ctx context.Context) context.Context {
    suffix := random.RandomString(6)
    nsName = "test-sync-" + suffix
    svcName = "test-svc-" + suffix
    // ...
})

// FAIL — different suffix per resource, can't correlate
BeforeAll(func(ctx context.Context) context.Context {
    nsName = "test-sync-" + random.RandomString(6)
    svcName = "test-svc-" + random.RandomString(6)
    // ...
})
```

### 5.2 Descriptive resource name prefixes

The prefix before the suffix should identify the test or resource purpose.

```go
// PASS
nsName = "svc-sync-test-" + suffix

// FAIL — generic prefix, hard to trace in cluster
nsName = "ns-" + suffix
```

---

## 6. Use the Lazy vCluster Pattern

Per-test vClusters live in `suite_*_test.go` and are created lazily in the suite's `BeforeAll` via `setup/lazyvcluster.LazyVCluster`. The only eager cluster is `clusters.HostCluster` (the kind host).

```go
// PASS — lazy creation inside the suite
Describe("myfeature-vcluster", labels.MyFeature, Ordered,
    cluster.Use(clusters.HostCluster),
    func() {
        BeforeAll(func(ctx context.Context) context.Context {
            return lazyvcluster.LazyVCluster(ctx, myFeatureName, myFeatureYAML)
        })
        // specs...
    },
)

// FAIL — eager registration in clusters/ + cluster.Use(vc) is the old pattern; do not reintroduce it.
```

Do not add definitions to `clusters/`. Pre-setup hooks (CRD install, PVC, Helm) go into `setup/` and are passed via `lazyvcluster.WithPreSetup`. See `e2e-next/README.md` for the full pattern.

---

## 7. Test Setup Does Not Patch System-Managed State

`BeforeEach` and `BeforeAll` must not directly set fields or conditions that a controller or deployment configuration would normally own. If setup code bypasses the system's real initialization path to establish a condition, the test is not exercising production behavior.

**FAIL**: `BeforeEach` patches a vcluster status field that the syncer reconciles into existence.

**PASS**: The vcluster is bootstrapped with the correct YAML configuration so the system establishes the expected state through its normal code path.

If you find yourself writing setup that patches system-managed state, stop. Read all existing tests that share the same labels — they show how the environment should be configured.

---

## 8. `By()` Text Style and Closure

`By()` must use a closure (see `e2e-test-structure.md`), and the text should be a human-readable sentence fragment describing *what* is happening, not *how*.

```go
// PASS — closure + descriptive text
By("Waiting for the replicated service to appear in vcluster", func() {
    // ... test logic
})

// FAIL — no closure; Ginkgo can't attribute failures to the step
By("Waiting for the replicated service to appear in vcluster")

// FAIL — too terse, sounds like a function name
By("WaitServiceSync", func() { /* ... */ })

// FAIL — describes implementation, not intent
By("Calling vClusterClient.CoreV1().Services().Get in a loop", func() { /* ... */ })
```

---

## 9. Lint Passes

`just -f Justfile.agent lint ./e2e-next/...` runs clean. This runs the full `golangci-lint` suite (formatting, static analysis, 28+ linters) matching CI. Note: `compile-check` only runs `go build` + `go vet`, which is a subset — always run `lint` as well.

---

## 10. No Hardcoded Cluster or Context Names

Never hardcode kind cluster names or kubectl context strings. Use `constants.GetHostClusterName()`
for the cluster name and derive the context from it. The cluster name is configurable via the
`KIND_NAME` env var and the `--cluster-name` flag.

```go
// PASS — derived from configurable constant
hostCluster := constants.GetHostClusterName()
kubectlContext := "kind-" + hostCluster

// FAIL — hardcoded, breaks when KIND_NAME is set
kubectlContext := "kind-kind-cluster"
```
