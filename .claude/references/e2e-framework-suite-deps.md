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

## Cluster-level provisioning (Kind clusters, vClusters)

Use this for: tests that need a full ephemeral Kubernetes cluster or vCluster, not a service inside the existing cluster.

The framework provides a dependency system via `pkg/setup/suite`:

```go
var (
    host = cluster.Define(
        cluster.WithName("host"),
        cluster.WithProvider(kind.NewProvider()),
    )

    vc = vcluster.Define(
        vcluster.WithName("my-vcluster"),
        vcluster.WithVersion("v0.30.0"),
        vcluster.WithHostCluster("host"),
    )
)

var _ = Describe("Feature tests", func() {
    Context("runs on the host cluster",
        cluster.Use(host),
        func() {
            It("can access current cluster clients", func(ctx context.Context) {
                crc := cluster.CurrentClusterClientFrom(ctx)
                kube := cluster.CurrentKubeClientFrom(ctx)
            })
        },
    )
})
```

Wire setup/teardown in `SynchronizedBeforeSuite`/`SynchronizedAfterSuite`:

```go
var _ = SynchronizedBeforeSuite(
    func(ctx context.Context) (context.Context, []byte) {
        ctx, err = setup.All(
            host.Setup,
            setup.AllConcurrent(vc1.Setup, vc2.Setup),
        )(ctx)
        Expect(err).NotTo(HaveOccurred())
        data, err := cluster.ExportAll(ctx)
        Expect(err).NotTo(HaveOccurred())
        return ctx, data
    },
    func(ctx context.Context, data []byte) context.Context {
        ctx, err = cluster.ImportAll(ctx, data)
        Expect(err).NotTo(HaveOccurred())
        return ctx
    },
)

var _ = SynchronizedAfterSuite(
    func(ctx context.Context) {},
    func(ctx context.Context) {
        _, err := setup.All(host.Teardown)(ctx)
        Expect(err).NotTo(HaveOccurred())
    },
)
```

**Ordering matters:** Host cluster must be ready before vClusters. Use `setup.All` for sequential ordering, nest `setup.AllConcurrent` for parallel vCluster creation. Teardown order is the reverse — vClusters are torn down automatically via their dependency on the host cluster.

Required wiring in `TestRunE2ETests`:
```go
var _ = AddTreeConstructionNodeArgsTransformer(suite.NodeTransformer)

func TestRunE2ETests(t *testing.T) {
    config, _ := GinkgoConfiguration()
    RegisterFailHandler(Fail)
    RunSpecs(t, "vCluster E2E Suite",
        AroundNode(suite.PreviewSpecsAroundNode(config)),
        AroundNode(e2e.ContextualAroundNode),
    )
}
```

**Focus is automatic**: dependencies only execute `Setup/Teardown` when their label appears in the focused specs. No cluster is spun up for tests that don't need it.

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
