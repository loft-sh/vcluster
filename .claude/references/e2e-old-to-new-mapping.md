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
| `f.RefreshVirtualClient()` | `connectVCluster()` — see "Reconnecting after destructive operations" below |
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

## Reconnecting After Destructive Operations

The suite-level background proxy (established in `SynchronizedBeforeSuite`) does **not** survive
vcluster pause/resume or cert rotation. `cluster.CurrentKubeClientFrom(ctx)` and
`cluster.CurrentClusterFrom(ctx).KubernetesRestConfig()` both point to the suite proxy — if the
vcluster has been paused, restarted, or had its certs rotated, these return dead connections.

The old framework's `f.RefreshVirtualClient()` re-established the connection. The e2e-next
equivalent is to run `vcluster connect` programmatically with a background proxy:

1. Create a temp file for the kubeconfig
2. Build a `connectcmd.ConnectCmd` with `BackgroundProxy: true`, `KubeConfig` pointing to the temp file, and `BackgroundProxyImage` set to `constants.GetVClusterImage()` — the same image already loaded into Kind
3. Call `cmd.Run(ctx, []string{vclusterName})`
4. Poll with `Eventually` until the kubeconfig file is non-empty and `clientcmd.RESTConfigFromKubeConfig` succeeds
5. Verify the connection by making a lightweight API call (e.g., `Get` the `default` ServiceAccount)
6. Return the `*rest.Config` and a cleanup func that removes the temp file

**Critical: use `constants.GetVClusterImage()` for `BackgroundProxyImage`**, not
`vclusterconstants.DefaultBackgroundProxyImage(upgrade.GetVersion())`.

Key imports: `connectcmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"`,
`"github.com/loft-sh/vcluster/pkg/cli"`, `"github.com/loft-sh/vcluster/pkg/cli/flags"`,
`loftlog "github.com/loft-sh/log"`,
`"github.com/spf13/cobra"`, `"k8s.io/client-go/tools/clientcmd"`,
`"github.com/loft-sh/vcluster/e2e-next/constants"`.
