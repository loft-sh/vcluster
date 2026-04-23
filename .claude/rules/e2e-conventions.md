---
paths:
  - "e2e-next/**/*.go"
---
<!-- Generic core: e2e-tdd-workflow plugin references/e2e-conventions-core.md -->

# e2e-next Test Conventions

Ginkgo v2 + Gomega, running against vCluster instances on Kind. Read existing tests in `e2e-next/test_*/` for patterns before writing new ones.

## Key Conventions

1. **Random suffixes** — Always use `objectmeta.GenerateName` or `random.RandomString(6)` for resource names to avoid collisions in parallel runs. Try to share the suffix across a single test spec so all resources of a single test use the same suffix. This applies to **all** identifiers that could collide, not just Kubernetes object names — Helm release names, database names, etc. If an identifier is a package-level `const` and would conflict across parallel runs, make it dynamic.
2. **Context propagation** — Setup helpers store objects in context; retrieve with `<resource>.From(ctx, name)`.
3. **Eventually for async** — vCluster operations are eventually consistent. Always wrap API checks in `Eventually` with `constants.*` timeouts. Never hardcode durations.
4. **Labels for filtering** — Tag `Describe` with resource labels and `labels.PR` if all specs in it should gate PRs. Only tag individual `It` blocks with `labels.PR` when some specs in the `Describe` should NOT gate PRs. Never duplicate a label on an `It` that is already present on its enclosing `Describe` or `Context` — Ginkgo inherits labels, so duplicates are redundant noise.
5. **Ordered contexts** — See the decision table below. Default to `BeforeEach`; only use `Ordered` for true sequential dependencies.
6. **Package registration** — Test packages self-register via `var _ = Describe(...)`. The suite imports them as blank imports in `e2e_suite_test.go`.
7. **Error assertion** — Prefer `ginkgo.Expect(...).To(ginkgo.Succeed())` over `ginkgo.Expect(...).NotTo(ginkgo.HaveOccurred())`
8. **gstruct usage** — Do not use `gstruct` for simple field assertions (use `Expect(obj.Field).To(Equal(...))` instead). Use `gstruct` with `ContainElement` or similar matchers when asserting on elements within a collection (slice, map) to avoid manual loops.

## Setup Helpers

vCluster's `e2e-next/` does not yet have a rich `setup/` builder library like loft-enterprise. During migration:

- Use existing **cluster definitions** from `e2e-next/clusters/` for test environment setup. Browse `clusters/clusters.go` for available vclusters with different configurations.
- Use the `setup/template` package (`template.MustRender()`) for rendering vcluster YAML with image overrides.
- When a setup pattern repeats across multiple tests, create a shared setup helper as an `[infra]` sub-problem. Place it in `e2e-next/setup/` following the functional-options pattern.
- For now, most test setup is done inline using the Kubernetes client directly (create namespace, create resource, DeferCleanup).

### Scoping Constants and Helpers

Test-specific constants, helper functions, and option builders that are only used by a single `test_*` package belong **in that package**, not in the shared `constants` or `setup` packages. Only promote to a shared package when a second consumer appears.

### External Service Provisioning

If the old test installs Helm charts or external services (via `values.yaml` `controlPlane.helmRelease`, host resources, or CI workflow steps), flag this as an `[infra]` sub-problem during migration. The external service needs to be provisioned self-contained within the test using `setuphelm.Upgrade()` in `BeforeAll` or via the cluster dependency system.

## vCluster Configuration Patterns

Each vCluster lives next to the `suite_*_test.go` that uses it. `HostCluster` (the kind host) is the only eagerly-provisioned dependency in `SynchronizedBeforeSuite`; every per-test vCluster is created lazily in the suite's own `BeforeAll` via `setup/lazyvcluster.LazyVCluster`, which is a thin wrapper over the framework's `vcluster.Create`.

### Embedded YAML Templates

Embed the `vcluster.yaml` in the suite file. Templates support `{{.Repository}}`, `{{.Tag}}`, `{{.HostClusterName}}` placeholders - the lazy helper renders them at `BeforeAll` time and registers temp-file cleanup automatically.

### Defining a New vCluster Suite

1. Create `e2e-next/vcluster-myfeature.yaml` with the `vcluster.yaml` config.
2. Create `e2e-next/suite_myfeature_test.go`:
   ```go
   //go:embed vcluster-myfeature.yaml
   var myFeatureYAML string

   const myFeatureName = "myfeature-vcluster"

   func init() { suiteMyFeature() }

   func suiteMyFeature() {
       Describe("myfeature-vcluster", labels.MyFeature, Ordered,
           cluster.Use(clusters.HostCluster),
           func() {
               BeforeAll(func(ctx context.Context) context.Context {
                   return lazyvcluster.LazyVCluster(ctx, myFeatureName, myFeatureYAML)
               })
               // spec functions...
           },
       )
   }
   ```
3. If a new filter label is needed, add it to `labels/labels.go`. `labels.PR` goes on the outer suite (never on specs).
4. If the vCluster needs host-side prerequisites (CRDs, PVCs, Helm install), pass `lazyvcluster.WithPreSetup(fn)`. Reusable helpers live in `setup/` (`setup.SnapshotPreSetup`, `setup.MetricsServerPreSetup`).

Do NOT add entries to `clusters/` - that package only holds `HostCluster` and `DefaultVClusterOptions`.

## Client Accessors

| Accessor | Type | Use for |
|----------|------|---------|
| `cluster.KubeClientFrom(ctx, clusterName)` | `kubernetes.Interface` | Host cluster typed client |
| `cluster.CurrentKubeClientFrom(ctx)` | `kubernetes.Interface` | Current vcluster typed client |
| `cluster.CurrentClusterClientFrom(ctx)` | `client.Client` (controller-runtime) | Current cluster CR client |

## Background Proxy Limitations

The suite-level background proxy (started by `vcluster connect` during `SynchronizedBeforeSuite`)
is a **one-shot process**. It dies when:

- The vcluster pod is paused/resumed (e.g., `certs rotate`, `certs rotate-ca`, helm upgrade)
- The vcluster's CA certificate changes
- The vcluster pod restarts for any reason

After any of these, `cluster.CurrentKubeClientFrom(ctx)` and
`cluster.CurrentClusterFrom(ctx).KubernetesRestConfig()` return connections to a dead proxy.

**If your test does destructive vcluster operations** (cert rotation, restart, config change),
you must establish a fresh connection using the `connectVCluster()` pattern documented in
`.claude/references/e2e-old-to-new-mapping.md` § "Reconnecting After Destructive Operations".

**Common pitfall:** When building `ConnectOptions`, always set `BackgroundProxyImage` to
`constants.GetVClusterImage()` — **never** use `DefaultBackgroundProxyImage(upgrade.GetVersion())`.
In dev builds the latter produces an invalid Docker image ref (empty tag), Docker rejects it,
and the code silently falls back to in-process port-forwarding that hangs the test process.

## Cleanup

For non-Ordered contexts, return the enriched context from `BeforeEach`:

```go
BeforeEach(func(ctx context.Context) context.Context {
    vClusterClient = cluster.CurrentKubeClientFrom(ctx)
    Expect(vClusterClient).NotTo(BeNil())
    return ctx
})
```

Inside `It` blocks, register cleanup immediately after creation:

```go
It("tests something", func(ctx context.Context) {
    _, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
        ObjectMeta: metav1.ObjectMeta{Name: nsName},
    }, metav1.CreateOptions{})
    Expect(err).NotTo(HaveOccurred())
    DeferCleanup(func(ctx context.Context) {
        err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
        Expect(clientpkg.IgnoreNotFound(err)).To(Succeed())
    })

    // ... test logic
})
```

## Labels

Attach labels to `Describe`, `Context`, or `It`. Check `e2e-next/labels/labels.go` for the full list. Key ones:
- `labels.PR` — gate tests that run on every PR
- `labels.Core`, `labels.Sync`, `labels.Deploy` — feature-area labels

## Ordered vs. BeforeEach

Default to `BeforeEach`. The suite will be parallelized — every unnecessary `Ordered` is a bottleneck.

| Signal | Pattern | Why |
|---|---|---|
| Specs are independent, just share setup | `BeforeEach` (no `Ordered`) | Parallelizable; no cascade failures |
| Specs form a lifecycle sequence (create → mutate → verify → delete) | `Ordered` + `BeforeAll` | Specs depend on prior spec's side effects |
| Setup is expensive (databases, vclusters with deployed services) | `BeforeEach` — pay the cost per spec | Parallel execution recovers wall time; isolation prevents false failures |
| A spec deletes or mutates the shared resource | `BeforeEach` with per-spec resource | Other specs can't rely on it existing |
| Specs must run in a fixed order but don't share mutable state | **Rethink** — probably independent specs | `Ordered` without true dependencies is a parallelization bottleneck |

> **Hard rule:** If you use `Ordered`, add a comment explaining which spec depends on which prior spec's side effect. If you can't name one, drop `Ordered`.

## Concrete Examples

See `.claude/references/e2e-examples.md` for annotated excerpts from production tests covering: DeferCleanup placement, `Eventually` with `g Gomega`, `Ordered` vs `BeforeEach` side-by-side, and cluster client usage.

## No Plan Artifacts in Code

Never include migration plan identifiers (e.g., `SP-0`, `SP-1`, `[migrate]`, `[infra]`, `[cleanup]`, `[consolidate]`) in code comments, `By()` text, or test descriptions. These are internal planning artifacts — the code should read as if no plan ever existed.

```go
// FAIL — plan artifact leaked into comment
// SP-2: verify configmap is synced to host
By("SP-2: Checking configmap sync", func() { /* ... */ })

// PASS — describes intent without plan references
// Verify the configmap is synced to the host cluster
By("Checking that the configmap is synced to the host cluster", func() { /* ... */ })
```

## New Test Checklist

1. Place the file in the appropriate `e2e-next/test_*` directory. Package name must match sibling files.
2. If creating a **new** `test_*` package, add a blank import in `e2e_suite_test.go`.
3. If testing a **new** resource type, add a label constant in `e2e-next/labels/labels.go`.
4. Use an existing test file in the same package as your starting template.
