# Old → New Pattern Mapping (vcluster)

Translation table for migrating tests from `test/framework` to `e2e-next/`.

## Client Accessors

| Old Pattern (`test/framework`) | New Pattern (`e2e-next`) |
|---|---|
| `framework.DefaultFramework` global singleton | Context-based: `cluster.KubeClientFrom(ctx, name)`, `cluster.CurrentKubeClientFrom(ctx)` |
| `f.HostClient` / `f.HostCRClient` | `cluster.KubeClientFrom(ctx, constants.GetHostClusterName())` |
| `f.VClusterClient` / `f.VClusterCRClient` | `cluster.CurrentKubeClientFrom(ctx)` / `cluster.CurrentClusterClientFrom(ctx)` |
| `f.VClusterName` / `f.VClusterNamespace` | Derived from cluster definition name in `clusters/` (e.g., `clusters.ServiceSyncVClusterName`) |
| `os.Getenv("VCLUSTER_NAME")` etc. | Cluster definitions with typed config in `constants/` |
| `f.Suffix` / `translate.VClusterName` | `translate` package used directly where needed |

## Setup and Lifecycle

| Old Pattern | New Pattern |
|---|---|
| `framework.CreateFramework(ctx)` in `BeforeSuite` | `SynchronizedBeforeSuite` with `clusters.HostCluster.Setup` + `clusters.XVCluster.Setup` |
| `f.Cleanup()` in `AfterSuite` | `SynchronizedAfterSuite` with `clusters.HostCluster.Teardown` |
| `f.RefreshVirtualClient()` | Handled by framework's cluster setup |
| Per-suite `values.yaml` | Per-cluster embedded YAML in `clusters/*.go` with `template.MustRender()` |
| Single vcluster per suite | Multiple vclusters with different configs (DefaultVCluster, NodesVCluster, etc.) |

## Test Execution

| Old Pattern | New Pattern |
|---|---|
| `go test -v -ginkgo.v` | `ginkgo --label-filter="..." ./e2e-next` |
| `time.Sleep()` | `Eventually().WithPolling().WithTimeout()` |
| Ad-hoc durations | `constants.PollingTimeout*` constants |
| Env-based config (`VCLUSTER_SUFFIX`, etc.) | Flag-based config (`--vcluster-image`, `--cluster-name`) |

## Timeouts

| Old Constant | New Constant | Duration |
|---|---|---|
| `framework.PollInterval` (5s) | `constants.PollingInterval` | 2s |
| `framework.PollTimeout` (1min) | `constants.PollingTimeout` | 60s |
| `framework.PollTimeoutLong` (2min) | `constants.PollingTimeoutLong` | 120s |
| N/A | `constants.PollingTimeoutVeryShort` | 5s |
| N/A | `constants.PollingTimeoutShort` | 20s |
| N/A | `constants.PollingTimeoutVeryLong` | 300s |

## Cluster Definitions

The old framework uses a single vcluster per suite. The new framework defines multiple vclusters in `e2e-next/clusters/clusters.go`:

```go
// Host cluster (Kind)
clusters.HostCluster = cluster.Define(
    cluster.WithName(constants.GetHostClusterName()),
    cluster.WithProvider(kind.NewProvider()),
    cluster.WithConfigFile("e2e-kind.config.yaml"),
)

// Virtual clusters — each with specific config
clusters.DefaultVCluster = vcluster.Define(
    vcluster.WithName("default-vcluster"),
    vcluster.WithVClusterYAML(DefaultVClusterYAML),
    vcluster.WithOptions(DefaultVClusterOptions...),
    vcluster.WithDependencies(HostCluster),
)
```

Use `cluster.Use(clusters.XVCluster)` in `Describe` to wire a test to a specific cluster:

```go
var _ = Describe("My test",
    labels.Core,
    cluster.Use(clusters.DefaultVCluster),
    cluster.Use(clusters.HostCluster),
    func() { ... },
)
```
