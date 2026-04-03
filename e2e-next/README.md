# e2e-next Test Suite

End-to-end tests for vCluster using the [e2e-framework](https://github.com/loft-sh/e2e-framework) with Ginkgo v2.

## Directory Structure

```
e2e-next/
├── e2e_suite_test.go                    # Infrastructure (flags, BeforeSuite, AfterSuite)
├── suite_e2e_test.go                    # Suite: common-vcluster (main PR-gating)
├── suite_fromhost_limitclasses_test.go  # Suite: fromhost-limitclasses-vcluster
├── suite_servicesync_test.go            # Suite: service-sync-vcluster
├── suite_kubeletproxy_test.go           # Suite: kubelet-proxy-vcluster
├── suite_snapshot_test.go              # Suite: snapshot-vcluster
├── suite_ha_certs_test.go               # Suite: certs-vcluster (Ordered)
├── suite_metricsproxy_test.go           # Suite: metricsproxy-vcluster
├── suite_node_test.go                   # Suite: node-sync-vcluster
├── suite_scheduler_test.go              # Suite: scheduler-vcluster
├── suite_isolation_mode_test.go         # Suite: isolation-mode-vcluster
├── suite_rootless_test.go               # Suite: rootless-vcluster
├── suite_vind_test.go                   # Suite: vind (docker driver)
│
├── clusters/                      # vCluster definitions (1 file + 1 YAML per cluster)
│   ├── registry.go                # Registration infrastructure (register, PreSetup, SetupFuncs)
│   ├── default.go                 # CommonVCluster (comprehensive e2e config)
│   ├── certs.go                   # CertsVCluster (single-replica, deploy etcd)
│   ├── isolation_mode.go          # IsolationModeVCluster (PSS, quota, limitrange)
│   ├── node_sync.go               # NodeSyncVCluster (all nodes, virtualScheduler)
│   ├── scheduler.go               # SchedulerVCluster (k8s scheduler, all nodes)
│   ├── rootless.go                # RootlessVCluster (runAsUser: 12345)
│   ├── servicesync.go             # ServiceSyncVCluster (replicateServices)
│   ├── fromhost_limitclasses.go   # FromHostLimitClassesVCluster (label-selector sync)
│   ├── snapshot.go                # SnapshotVCluster (CSI + PVC presetup)
│   ├── kubeletproxy.go            # KubeletProxyVCluster
│   ├── metricsproxy.go            # MetricsProxyVCluster
│   └── *.yaml                     # Embedded vcluster.yaml templates
│
├── test_core/                     # Test logic (self-describing spec functions)
│   ├── sync/                      # Sync tests (pods, pvc, networkpolicy, servicesync, etc.)
│   │   └── fromhost/              # FromHost sync tests (configmaps, secrets, etc.)
│   ├── coredns/                   # CoreDNS resolution tests
│   ├── webhook/                   # Admission webhook tests
│   ├── certs/                     # Cert rotation tests
│   ├── isolation/                 # Isolation mode tests
│   ├── nodesync/                  # Node sync (all nodes) tests
│   ├── scheduler/                 # Scheduler taint/toleration + WaitForFirstConsumer
│   ├── rootless/                  # Rootless mode tests
│   └── snapshot/                  # Snapshot & restore tests
│
├── test_deploy/                   # Deploy tests (helm charts, init manifests)
├── setup/                         # Setup helpers (CSI driver install, snapshot PVC)
├── labels/                        # Ginkgo label constants
├── constants/                     # Shared constants (timeouts, cluster name, image)
└── init/                          # Framework initialization
```

## Architecture

Tests are split into two layers:

1. **Spec functions** (`test_core/`, `test_deploy/`) - self-describing: each carries its own
   Describe text and feature labels, but no cluster binding or scheduling labels.
2. **Suite files** (`suite_*_test.go`) - one per vCluster: binds specs to a cluster via
   `cluster.Use()` and optionally adds scheduling labels like `labels.PR`.

```go
// Spec function (test_core/sync/test_pods.go)
func PodSyncSpec() {
    Describe("Pod sync from vCluster to host",
        labels.Core, labels.Pods, labels.Sync,   // feature labels
        func() { /* test logic */ },
    )
}

// Suite file (suite_rootless_test.go)
func suiteRootlessVCluster() {
    Describe("rootless-vcluster",                 // vCluster name
        cluster.Use(clusters.RootlessVCluster),   // cluster binding
        cluster.Use(clusters.HostCluster),
        func() {
            rootless.RootlessModeSpec()            // list of specs
            coredns.CoreDNSSpec()
            test_core.PodSyncSpec()
        },
    )
}
```

The same spec can run against multiple vClusters. The suite controls which vCluster
and whether the tests gate PRs.

## Test Suites

Each suite file maps to one vCluster. One file, one vCluster, one function.

| Suite file | vCluster | PR-gated | Specs |
|------------|----------|----------|-------|
| `suite_e2e_test.go` | `common-vcluster` | yes | 14 (sync, fromHost, coredns, webhook, deploy) |
| `suite_fromhost_limitclasses_test.go` | `fromhost-limitclasses-vcluster` | yes | 4 (ingress/storage/priority/runtime classes) |
| `suite_servicesync_test.go` | `service-sync-vcluster` | yes | 1 (service replication) |
| `suite_kubeletproxy_test.go` | `kubelet-proxy-vcluster` | yes | 1 (kubelet proxy) |
| `suite_snapshot_test.go` | `snapshot-vcluster` | no | 1 (snapshot & restore, Ordered) |
| `suite_ha_certs_test.go` | `certs-vcluster` | no | 1 (cert rotation, Ordered) |
| `suite_metricsproxy_test.go` | `metricsproxy-vcluster` | no | 1 (metrics proxy) |
| `suite_isolation_mode_test.go` | `isolation-mode-vcluster` | no | 6 (isolation + shared specs) |
| `suite_node_test.go` | `node-sync-vcluster` | no | 6 (nodesync + shared specs) |
| `suite_rootless_test.go` | `rootless-vcluster` | no | 6 (rootless + shared specs) |
| `suite_scheduler_test.go` | `scheduler-vcluster` | no | 7 (scheduler + shared specs) |
| `suite_vind_test.go` | (self-managed) | no | 1 (docker driver lifecycle) |

## Labels

| Label | Description |
|-------|-------------|
| `pr` | Scheduling label: tests that gate pull requests (on suite, not spec) |
| `core` | Core vCluster functionality |
| `sync` | Resource sync tests (toHost/fromHost) |
| `deploy` | Deployment tests (helm, manifests) |
| `storage` | PVC/PV storage tests |
| `security` | Webhook, cert rotation, isolation tests |
| `scheduler` | Virtual scheduler tests |
| `pods` | Pod sync tests |
| `pvcs` | PVC sync tests |
| `coredns` | CoreDNS resolution tests |
| `webhooks` | Admission webhook tests |
| `events` | Event sync tests |
| `snapshots` | Snapshot & restore tests |
| `configmaps` | ConfigMap fromHost sync |
| `secrets` | Secret fromHost sync |
| `networkpolicies` | NetworkPolicy sync (requires Calico CNI) |
| `non-default` | Tests requiring special infra (e.g. Calico CNI) - excluded by default |

## Running Tests

### Prerequisites

```bash
# Install ginkgo CLI
go install github.com/onsi/ginkgo/v2/ginkgo

# Install kind
# https://kind.sigs.k8s.io/docs/user/quick-start/
```

### Full cycle (setup + run + teardown)

```bash
just dev-e2e
```

### Setup environment only (no tests)

```bash
just setup
```

### Run tests (environment already set up)

```bash
# All PR-gating tests (excluding non-default):
just run-e2e 'pr && !non-default'

# All tests including NetworkPolicy:
just run-e2e ''

# Specific vCluster suite:
just run-e2e 'common-vcluster'
just run-e2e 'certs-vcluster'
just run-e2e 'scheduler-vcluster'

# By feature label (across all vClusters):
just run-e2e 'pods'
just run-e2e 'coredns'
just run-e2e 'snapshots'
just run-e2e 'security'

# Combine:
just run-e2e 'pr && pods'

# Iterate without teardown:
just iterate-e2e 'pods'
```

### Teardown

```bash
just teardown
```

## Adding a New Test

1. Create a test file in the appropriate `test_core/` or `test_deploy/` subdirectory.
2. Export a spec function that calls Describe with feature labels (no `cluster.Use`):
   ```go
   func MyFeatureSpec() {
       Describe("My feature does something",
           labels.Core, labels.Sync,
           func() {
               It("should work", func(ctx context.Context) {
                   client := cluster.CurrentKubeClientFrom(ctx)
                   // test logic
               })
           },
       )
   }
   ```
3. Register the spec in the appropriate suite file:
   ```go
   func suiteCommonVCluster() {
       Describe("common-vcluster", labels.PR,
           cluster.Use(clusters.CommonVCluster),
           cluster.Use(clusters.HostCluster),
           func() {
               mypackage.MyFeatureSpec()
           },
       )
   }
   ```
4. If the test needs a new vCluster config, add a YAML + Go file in `clusters/`.
5. If the test needs a new label, add it to `labels/labels.go`.

## Adding a New vCluster Configuration

1. Create `clusters/vcluster-myfeature.yaml` with the vcluster.yaml config.
   Use `{{.Repository}}`, `{{.Tag}}`, `{{.HostClusterName}}` template vars.
2. Create `clusters/myfeature.go`:
   ```go
   package clusters

   import _ "embed"

   //go:embed vcluster-myfeature.yaml
   var myFeatureYAML string

   var (
       MyFeatureVClusterName = "myfeature-vcluster"
       MyFeatureVCluster     = register(MyFeatureVClusterName, myFeatureYAML)
   )
   ```
3. Create a new suite file `suite_myfeature_test.go`:
   ```go
   func init() {
       suiteMyFeatureVCluster()
   }

   func suiteMyFeatureVCluster() {
       Describe("myfeature-vcluster",
           cluster.Use(clusters.MyFeatureVCluster),
           cluster.Use(clusters.HostCluster),
           func() {
               mypackage.MyFeatureSpec()
           },
       )
   }
   ```
4. Only focused vClusters are provisioned (via `--label-filter`), so adding a new
   cluster definition does not slow down other suites.

## Custom Linters

The e2e-next tests are checked by custom golangci-lint plugins (built via `golangci-lint custom`
from `.custom-gcl.yml`). These run in CI and locally via `just lint ./e2e-next/...`.

| Linter | What it checks |
|--------|---------------|
| `describefunc` | Spec functions in `test_*` packages must not call `Describe()` with `cluster.Use()`. Cluster binding belongs in suite files, not in specs. |
| `defercleanupcluster` | `cluster.Create()` calls must have a matching `DeferCleanup(cluster.Destroy(...))`. |
| `defercleanupctx` | `DeferCleanup` must not be called with a `setup.Func`; use `e2e.DeferCleanupCtx(ctx, fn)` instead. |
| `ginkgoreturnctx` | Ginkgo node functions that reassign `context.Context` must return it. |

If a linter flags your code, the error message explains the fix. Source code for all
linters lives in [loft-sh/e2e-framework/linters/](https://github.com/loft-sh/e2e-framework/tree/main/linters).

## Cross-repo Usage (vcluster-pro)

Spec functions are exported and carry their own labels. vcluster-pro imports them
and registers against its own vCluster definitions:

```go
// vcluster-pro/e2e-next/suite_deploy_etcd_test.go
import test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"

func init() {
    suiteDeployEtcdVCluster()
}

func suiteDeployEtcdVCluster() {
    Describe("deploy-etcd-vcluster",
        cluster.Use(proClusters.DeployEtcdVCluster),
        cluster.Use(proClusters.HostCluster),
        func() {
            test_core.PodSyncSpec()
            test_core.PVCSyncSpec()
            // pro-specific specs...
        },
    )
}
```
