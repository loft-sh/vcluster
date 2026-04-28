# e2e-framework: External Infrastructure Patterns

Two distinct patterns depending on what needs provisioning. Pick the right one before writing the plan.

## Service-level provisioning (Helm charts, external APIs)

Use this for: ingress-nginx, fluent-bit, metrics-server, cert-manager, CSI drivers, or any service installed into the test cluster.

Install in `BeforeAll`, tear down in `AfterAll`. Never assume the service is pre-provisioned.

```go
BeforeAll(func(ctx context.Context) context.Context {
    By("installing <service>")
    ctx, err := setuphelm.Upgrade(
        setuphelm.WithHelmOptions(
            constants.Get<Service>HelmOptions()...,
            helm.WithWait(),
        ),
    )(ctx)
    Expect(err).NotTo(HaveOccurred())
    DeferCleanup(...)
    return ctx
})

AfterAll(func(ctx context.Context) {
    _, err := setuphelm.Uninstall(
        setuphelm.WithHelmOptions(
            helm.WithName("<release>"),
            helm.WithNamespace("<ns>"),
        ),
    )(ctx)
    Expect(err).NotTo(HaveOccurred())
    _, err = namespace.Delete("<ns>")(ctx)
    Expect(err).NotTo(HaveOccurred())
})
```

**Why `AfterAll` and not `DeferCleanup`**: `DeferCleanup` in `BeforeAll` runs after all specs, which is equivalent here — but `AfterAll` is explicit and survives `--setup-only` runs where you want the service to persist between dev iterations.

**Chart constants**: Add a `e2e-next/constants/<service>.go` with `Get<Service>HelmOptions()` following the pattern in existing constants files.

---

## Cluster-level provisioning (Kind host, vClusters)

Use this for: tests that need a full ephemeral Kubernetes cluster or vCluster, not a service inside an existing cluster.

**Two-layer model:**

1. **Host kind cluster** - eagerly provisioned once in `SynchronizedBeforeSuite` via the framework's `cluster.Define` + `cluster.Setup`. Defined in `e2e-next/clusters/registry.go` as `clusters.HostCluster`. Reused by every per-test vCluster.
2. **Per-test vClusters** - created lazily, one per `suite_*_test.go`, in the suite's outer `BeforeAll` via `setup/lazyvcluster.LazyVCluster`. The helper is a thin wrapper over the framework's `vcluster.Create` (see `github.com/loft-sh/e2e-framework/pkg/setup/vcluster`). Peak concurrent vClusters is bounded by `ginkgo --procs`, not by the number of suite files.

```go
// e2e-next/suite_myfeature_test.go
//go:embed vcluster-myfeature.yaml
var myFeatureYAML string

const myFeatureName = "myfeature-vcluster"

func init() { suiteMyFeature() }

func suiteMyFeature() {
    Describe("myfeature-vcluster", labels.MyFeature, Ordered,
        cluster.Use(clusters.HostCluster),
        func() {
            BeforeAll(func(ctx context.Context) context.Context {
                return lazyvcluster.LazyVCluster(ctx, myFeatureName, myFeatureYAML,
                    // optional: host-side prerequisites (CRDs, PVCs, Helm install)
                    lazyvcluster.WithPreSetup(setup.MyPrereq()),
                )
            })

            // spec functions...
        },
    )
}
```

Inside specs: `cluster.CurrentKubeClientFrom(ctx)` resolves to the suite's vCluster. `cluster.KubeClientFrom(ctx, constants.GetHostClusterName())` resolves to the host.

**Failure-aware teardown**: on spec failure the framework keeps the vCluster alive and attaches diagnostics (rendered config, pods, events, syncer logs) to the failing spec's report entries via `AddReportEntry`. On pass, teardown destroys the vCluster.

**Focus is automatic**: a label-filtered run only materializes vClusters whose outer suite Describe matches. Unmatched suites' `BeforeAll` never fires, so no vCluster is spun up for tests that do not need it.

---

## Identifying what needs provisioning

When migrating an old test, check these files in order:

1. **Old test's `values.yaml`** — Look for `controlPlane.helmRelease` entries (charts the vcluster deploys internally), `experimental.deploy.vcluster.helm` sections, and any inline manifests in `experimental.deploy.vcluster.manifests`.
2. **`host-resources.yaml`** (if present) — PVCs for volume snapshots, CRDs, or other host-cluster prerequisites.
3. **`.github/workflows/e2e.yaml`** — CI workflow steps that install external dependencies before running the test suite.

### Known External Dependencies by Suite

| Dependency | Suites | Notes |
|---|---|---|
| CSI volume snapshots | e2e, e2e_node, e2e_scheduler, e2e_isolation_mode, e2e_rootless | Snapshot CRDs + controller |
| ingress-nginx 4.14.2 | e2e, e2e_node, e2e_scheduler, e2e_isolation_mode, e2e_rootless | Ingress controller for service exposure tests |
| fluent-bit 3.1.13 (OCI) | e2e, e2e_node, e2e_scheduler, e2e_isolation_mode, e2e_rootless | Deployed as Helm chart inside vcluster |
| metrics-server | e2e_metrics_proxy | Required for metrics API proxy tests |
| Plugin images | e2e_plugin | Custom plugin container images |

### Dependency-Free Suites

These suites have no external service dependencies:
- `e2e_certs`
- `e2e_cli`
- `e2e_ha`
- `e2e_limit_classes`
- `e2e_pause_resume`

**How to read the old test setup**: Look at the suite's CI workflow step. Each `helm install` or `kubectl apply` in the workflow's pre-test phase becomes either a `setuphelm.Upgrade()` call in `BeforeAll` or a cluster-level dependency via `cluster.Define`/`vcluster.Define`. The vcluster's `values.yaml` shows what config was baked into the vcluster itself — this becomes embedded YAML in `clusters/*.go`.
