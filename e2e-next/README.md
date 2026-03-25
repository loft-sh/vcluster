# e2e-next Test Suite

End-to-end tests for vCluster using the [e2e-framework](https://github.com/loft-sh/e2e-framework) with Ginkgo v2.

## Directory Structure

```
e2e-next/
├── e2e_suite_test.go              # Infrastructure (flags, BeforeSuite, AfterSuite)
├── suite_e2e_test.go              # Suite: e2e (main - comprehensive vCluster)
├── suite_ha_certs_test.go         # Suite: ha_certs (HA cert rotation)
├── suite_node_test.go             # Suite: node (all-nodes sync)
├── suite_scheduler_test.go        # Suite: scheduler (virtual scheduler)
├── suite_isolation_mode_test.go   # Suite: isolation_mode (PSS, quota, limitrange)
├── suite_rootless_test.go         # Suite: rootless (non-root UID)
│
├── clusters/                      # vCluster definitions (1 file + 1 YAML per cluster)
│   ├── registry.go                # Registration infrastructure (register, PreSetup, SetupFuncs)
│   ├── default.go                 # K8sDefaultEndpointVCluster (comprehensive e2e config)
│   ├── ha.go                      # HAVCluster (3 replicas)
│   ├── isolation_mode.go          # IsolationModeVCluster (PSS, quota, limitrange)
│   ├── node_sync.go               # NodeSyncVCluster (all nodes, virtualScheduler)
│   ├── scheduler.go               # SchedulerVCluster (k8s scheduler, all nodes)
│   ├── rootless.go                # RootlessVCluster (runAsUser: 12345)
│   ├── servicesync.go             # ServiceSyncVCluster (replicateServices)
│   ├── fromhost_limitclasses.go   # FromHostLimitClassesVCluster (label-selector sync)
│   ├── snapshot.go                # SnapshotVCluster (CSI + PVC presetup)
│   └── *.yaml                     # Embedded vcluster.yaml templates
│
├── test_core/                     # Test logic (exported Describe* functions)
│   ├── sync/                      # Sync tests (pods, pvc, networkpolicy, servicesync, etc.)
│   │   └── fromhost/              # FromHost sync tests (configmaps, secrets, etc.)
│   ├── coredns/                   # CoreDNS resolution tests
│   ├── webhook/                   # Admission webhook tests
│   ├── certs/                     # Cert rotation tests (HA)
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

## Test Suites

Each suite registers tests against a specific vCluster configuration. Tests are exported
functions (`Describe*(vcluster suite.Dependency) bool`) so the same test logic can run
against different vCluster configs and be imported by other repos (e.g. vcluster-pro).

| Suite | File | vCluster | Run command |
|-------|------|----------|-------------|
| e2e (main) | `suite_e2e_test.go` | `common-vcluster` | `just run-e2e '/common-vcluster/ && !non-default'` |
| ha_certs | `suite_ha_certs_test.go` | `ha-certs-vcluster` | `just run-e2e '/ha-certs-vcluster/ && !non-default'` |
| node | `suite_node_test.go` | `node-sync-vcluster` | `just run-e2e '/node-sync-vcluster/ && !non-default'` |
| scheduler | `suite_scheduler_test.go` | `scheduler-vcluster` | `just run-e2e '/scheduler-vcluster/ && !non-default'` |
| isolation_mode | `suite_isolation_mode_test.go` | `isolation-mode-vcluster` | `just run-e2e '/isolation-mode/ && !non-default'` |
| rootless | `suite_rootless_test.go` | `rootless-vcluster` | `just run-e2e '/rootless-vcluster/ && !non-default'` |

## Labels

| Label | Description |
|-------|-------------|
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
| `pr` | Tests that gate pull requests |

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
# All tests (excluding non-default):
just run-e2e '!non-default'

# All tests including NetworkPolicy:
just run-e2e ''

# Specific suite:
just run-e2e '/k8s-default-endpoint/ && !non-default'
just run-e2e '/ha-certs-vcluster/'
just run-e2e '/scheduler-vcluster/'

# By feature label:
just run-e2e 'pods'
just run-e2e 'coredns'
just run-e2e 'snapshots'
just run-e2e 'security'

# Iterate without teardown:
just iterate-e2e 'pods'
```

### Teardown

```bash
just teardown
```

## Adding a New Test

1. Create a test file in the appropriate `test_core/` or `test_deploy/` subdirectory.
2. Export a `Describe*` function that accepts `vcluster suite.Dependency`:
   ```go
   func DescribeMyFeature(vcluster suite.Dependency) bool {
       return Describe("My feature",
           labels.Core,
           cluster.Use(vcluster),
           cluster.Use(clusters.HostCluster),
           func() { /* test logic */ },
       )
   }
   ```
3. Use `cluster.CurrentClusterNameFrom(ctx)` for the vCluster name (not hardcoded).
4. Register the test in the appropriate `e2e_*_test.go` suite file:
   ```go
   var _ = mypackage.DescribeMyFeature(clusters.K8sDefaultEndpointVCluster)
   ```
5. If the test needs a new vCluster config, add a YAML + Go file in `clusters/`.
6. If the test needs a new label, add it to `labels/labels.go`.

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
3. Only focused vClusters are provisioned (via `--label-filter`), so adding a new
   cluster definition does not slow down other suites.

## Cross-repo Usage (vcluster-pro)

Tests are exported functions, so vcluster-pro can import and register them
against its own vCluster definitions:

```go
// vcluster-pro/e2e-next/register_tests_test.go
import test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"

var _ = test_core.DescribePodSync(proClusters.IsolationModeVCluster)
```
