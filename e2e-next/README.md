# e2e-next Test Suite

End-to-end tests for vCluster using the [e2e-framework](https://github.com/loft-sh/e2e-framework) with Ginkgo v2.

## Directory Structure

```
e2e-next/
├── e2e_config_test.go                    # Infrastructure (flags, BeforeSuite, AfterSuite)
├── suite_e2e_test.go                     # Suite: common-vcluster (main PR-gating)
├── suite_fromhost_limitclasses_test.go   # Suite: fromhost-limitclasses-vcluster
├── suite_servicesync_test.go             # Suite: service-sync-vcluster
├── suite_kubeletproxy_test.go            # Suite: kubelet-proxy-vcluster
├── suite_snapshot_test.go                # Suite: snapshot-vcluster
├── suite_ha_certs_test.go                # Suite: certs-vcluster (Ordered)
├── suite_cert_rotation_test.go           # Suite: short-certs-vcluster (Ordered)
├── suite_ha_cert_rotation_test.go        # Suite: ha-short-certs-vcluster
├── suite_metricsproxy_test.go            # Suite: metricsproxy-vcluster
├── suite_node_test.go                    # Suite: node-sync-vcluster
├── suite_scheduler_test.go               # Suite: scheduler-vcluster
├── suite_isolation_mode_test.go          # Suite: isolation-mode-vcluster
├── suite_rootless_test.go                # Suite: rootless-vcluster
├── suite_plugin_test.go                  # Suite: plugin-vcluster
├── suite_lifecycle_test.go               # Suite: cli-vcluster (Ordered)
├── suite_export_kubeconfig_test.go       # Suite: export-kubeconfig-vcluster
├── suite_vind_test.go                    # Suite: vind (docker driver)
│
├── vcluster-*.yaml                       # Per-suite embedded vcluster.yaml templates
│
├── clusters/                      # Shared cluster infrastructure
│   ├── registry.go                # HostCluster (kind) + DefaultVClusterOptions
│   └── export_kubeconfig.go       # Cross-package constants for the export-kubeconfig tests
│
├── setup/
│   ├── lazyvcluster/              # Per-suite lazy vCluster helper (see Architecture)
│   ├── snapshot.go                # SnapshotPreSetup (CSI + PVC)
│   ├── metricsproxy.go            # MetricsServerPreSetup (helm install)
│   ├── csi.go
│   └── template/
│
├── test_core/                     # Test logic (self-describing spec functions)
│   ├── sync/                      # Sync tests (pods, pvc, networkpolicy, servicesync, etc.)
│   │   └── fromhost/              # FromHost sync tests (configmaps, secrets, etc.)
│   ├── coredns/                   # CoreDNS resolution tests
│   ├── export_kubeconfig/         # Export-kubeconfig additional-secret tests
│   ├── lifecycle/                 # CLI connect / pause-resume tests
│   └── ...
│
├── test_deploy/                   # Deploy tests (helm charts, init manifests)
├── test_integration/              # Plugin, metrics-proxy integration tests
├── test_modes/                    # scheduler, nodesync mode tests
├── test_security/                 # Webhook, rootless, isolation, certs, kubeletproxy tests
├── test_storage/                  # Snapshot tests
├── labels/                        # Ginkgo label constants
├── constants/                     # Shared constants (timeouts, cluster name, image)
└── init/                          # Framework initialization
```

## Architecture

Tests are split into two layers plus a shared lifecycle helper:

1. **Spec functions** (`test_*/`) - self-describing: each carries its own `Describe` text and feature labels (e.g. `labels.Core, labels.Pods, labels.Sync`). No cluster binding, no `labels.PR`. PR gating is decided by the enclosing suite.
2. **Suite files** (`suite_*_test.go`) - one per vCluster configuration. Owns:
   - the embedded `vcluster-*.yaml` template
   - the vCluster name const
   - the vCluster lifecycle: `Ordered` + `BeforeAll` that calls `lazyvcluster.LazyVCluster`
   - the scheduling labels (`labels.PR` on PR-gated suites, plus one primary label per suite like `labels.Rootless`)
3. **`setup/lazyvcluster`** - the shared helper, a thin wrapper over the framework's `vcluster.Create`.

Only `HostCluster` (the kind host) stays in the framework's dependency mechanism (eagerly provisioned in `SynchronizedBeforeSuite`). Every per-test vCluster is created lazily in its own suite's `BeforeAll` and destroyed after the last spec in that suite finishes.

```go
// Spec function (test_core/sync/test_pods.go) - no cluster binding
func PodSyncSpec() {
    Describe("Pod sync from vCluster to host",
        labels.Core, labels.Pods, labels.Sync,   // feature labels
        func() { /* test logic */ },
    )
}

// Suite file (suite_rootless_test.go) - owns cluster lifecycle
//go:embed vcluster-rootless.yaml
var rootlessVClusterYAML string

const rootlessVClusterName = "rootless-vcluster"

func init() { suiteRootlessVCluster() }

func suiteRootlessVCluster() {
    Describe("rootless-vcluster", Ordered,
        cluster.Use(clusters.HostCluster),
        func() {
            BeforeAll(func(ctx context.Context) context.Context {
                return lazyvcluster.LazyVCluster(ctx, rootlessVClusterName, rootlessVClusterYAML)
            })

            rootless.RootlessModeSpec()
            coredns.CoreDNSSpec()
            test_core.PodSyncSpec()
        },
    )
}
```

The same spec can run against multiple vClusters. The suite controls which vCluster, the lifecycle, and whether the tests gate PRs.

## Lazy vCluster lifecycle

Each suite's outer `Describe` is `Ordered` so Ginkgo fires `BeforeAll` + `AfterAll` once per Describe. The local `lazyvcluster.LazyVCluster` helper is a thin wrapper over the framework's `vcluster.Create` (in `github.com/loft-sh/e2e-framework/pkg/setup/vcluster`) that:

- renders the embedded YAML with `{{.Repository}}`, `{{.Tag}}`, `{{.HostClusterName}}` (plus any `WithExtraTemplateVars`)
- runs the provided `WithPreSetup` hook first (optional - used for CSI install, metrics-server install, CRDs, etc.)
- delegates to `vcluster.Create(ctx, vcluster.Spec{...})` which creates the vCluster, wires failure-aware teardown, and calls `cluster.UseCluster` so `cluster.CurrentKubeClientFrom(ctx)` resolves inside specs

On spec failure the framework keeps the failed vCluster alive and attaches diagnostics (rendered config, pods, events, syncer logs) as report entries on the failing spec. On pass, teardown destroys the vCluster normally.

Peak concurrent vClusters is bounded by `ginkgo --procs`, not by the number of suite files. A label-filtered run only materializes vClusters for suites whose outer Describe matches.

## Test Suites

Each suite file maps to one vCluster. One file, one vCluster, one function.

| Suite file | vCluster | PR-gated |
|------------|----------|----------|
| `suite_e2e_test.go` | `common-vcluster` | yes |
| `suite_fromhost_limitclasses_test.go` | `fromhost-limitclasses-vcluster` | yes |
| `suite_servicesync_test.go` | `service-sync-vcluster` | yes |
| `suite_kubeletproxy_test.go` | `kubelet-proxy-vcluster` | yes |
| `suite_snapshot_test.go` | `snapshot-vcluster` | no |
| `suite_ha_certs_test.go` | `certs-vcluster` | no |
| `suite_cert_rotation_test.go` | `short-certs-vcluster` | no |
| `suite_ha_cert_rotation_test.go` | `ha-short-certs-vcluster` | no |
| `suite_metricsproxy_test.go` | `metricsproxy-vcluster` | no |
| `suite_isolation_mode_test.go` | `isolation-mode-vcluster` | no |
| `suite_node_test.go` | `node-sync-vcluster` | no |
| `suite_rootless_test.go` | `rootless-vcluster` | no |
| `suite_scheduler_test.go` | `scheduler-vcluster` | no |
| `suite_plugin_test.go` | `plugin-vcluster` | no |
| `suite_lifecycle_test.go` | `cli-vcluster` | no |
| `suite_export_kubeconfig_test.go` | `export-kubeconfig-vcluster` | no |
| `suite_vind_test.go` | (self-managed) | no |

## Labels

Labels are defined in `labels/labels.go`. `labels.PR` goes on suites that should gate every PR. Every opt-in suite has one primary label that matches its vCluster (e.g. `labels.Rootless` for `rootless-vcluster`) so `--label-filter='rootless'` targets just that suite.

**Scheduling:**

| Label | Applied to | Run it with |
|-------|------------|-------------|
| `pr` | PR-gated suites | `--label-filter='pr'` |
| `non-default` | Tests needing special infra (e.g. Calico) | excluded by default |

**Per-suite primary labels:**

| Label | Suite |
|-------|-------|
| `certs` | `short-certs-vcluster`, `ha-short-certs-vcluster`, `certs-vcluster` |
| `cli` | `cli-vcluster` |
| `exportkubeconfig` | `export-kubeconfig-vcluster` |
| `isolation` | `isolation-mode-vcluster` |
| `metricsproxy` | `metricsproxy-vcluster` |
| `nodesync` | `node-sync-vcluster` |
| `plugin` | `plugin-vcluster` |
| `rootless` | `rootless-vcluster` |
| `scheduler` | `scheduler-vcluster` |
| `snapshots` | `snapshot-vcluster` |
| `vind` | `test_vind` |

**Feature-area labels (spec level, for cross-suite filters):**

`core`, `sync`, `deploy`, `storage`, `security`, `integration`, plus resource-specific `pods`, `pvcs`, `coredns`, `webhooks`, `events`, `configmaps`, `secrets`, `networkpolicies`, `priorityclasses`, `runtimeclasses`, `storageclasses`, `ingressclasses`.

## Timeout Constants

Use these instead of hardcoded durations. Defined in `constants/timeouts.go`.

| Constant | Duration | Use for |
|----------|----------|---------|
| `PollingInterval` | 2s | Polling interval for all `Eventually`/`Consistently` |
| `PollingTimeoutVeryShort` | 5s | Immediate state checks (resource already exists) |
| `PollingTimeoutShort` | 20s | Quick API operations (get, list, delete) |
| `PollingTimeout` | 60s | Standard operations (pod ready, secret created) |
| `PollingTimeoutLong` | 120s | Resource creation (namespace, VCI becoming Ready) |
| `PollingTimeoutVeryLong` | 300s | vCluster startup, cluster creation |

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

1. Create a test file in the appropriate `test_*/` subdirectory.
2. Export a spec function that calls `Describe` with feature labels, but NO `cluster.Use`:
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
3. Register the spec in the appropriate suite file, inside the `BeforeAll` container:
   ```go
   func suiteCommonVCluster() {
       Describe("common-vcluster", labels.PR, Ordered,
           cluster.Use(clusters.HostCluster),
           func() {
               BeforeAll(func(ctx context.Context) context.Context {
                   return lazyvcluster.LazyVCluster(ctx, commonVClusterName, commonVClusterYAML)
               })

               mypackage.MyFeatureSpec()
           },
       )
   }
   ```
4. If the test needs a new vCluster config - see the next section.
5. If the test needs a new label, add it to `labels/labels.go`.

## Adding a New vCluster Configuration

vClusters live next to the suite that uses them. Do NOT add entries to `clusters/`.

1. Create `e2e-next/vcluster-myfeature.yaml` (sibling of the suite files) with the `vcluster.yaml` config. Use `{{.Repository}}`, `{{.Tag}}`, `{{.HostClusterName}}` template vars.
2. Create `suite_myfeature_test.go`:
   ```go
   package e2e_next

   import (
       "context"
       _ "embed"

       "github.com/loft-sh/e2e-framework/pkg/setup/cluster"
       "github.com/loft-sh/vcluster/e2e-next/clusters"
       "github.com/loft-sh/vcluster/e2e-next/setup/lazyvcluster"
       . "github.com/onsi/ginkgo/v2"
   )

   //go:embed vcluster-myfeature.yaml
   var myFeatureVClusterYAML string

   const myFeatureVClusterName = "myfeature-vcluster"

   func init() { suiteMyFeatureVCluster() }

   func suiteMyFeatureVCluster() {
       Describe("myfeature-vcluster", Ordered,
           cluster.Use(clusters.HostCluster),
           func() {
               BeforeAll(func(ctx context.Context) context.Context {
                   return lazyvcluster.LazyVCluster(ctx, myFeatureVClusterName, myFeatureVClusterYAML)
               })

               // spec functions...
           },
       )
   }
   ```
3. Only focused suites run under a label filter, so the lazy `BeforeAll` only fires for matching suites - adding a new vCluster does not slow down other runs.

### vCluster with a PreSetup hook

If the vCluster needs a host-side prerequisite (CRD, PVC, helm chart, namespace/RBAC) before it starts, pass `WithPreSetup`:

```go
BeforeAll(func(ctx context.Context) context.Context {
    return lazyvcluster.LazyVCluster(ctx,
        myFeatureVClusterName,
        myFeatureVClusterYAML,
        lazyvcluster.WithPreSetup(func(ctx context.Context) error {
            // install CRDs, create PVC, etc.
            return nil
        }),
    )
})
```

Reusable pre-setup helpers live in `setup/` (e.g. `setup.SnapshotPreSetup(name)`, `setup.MetricsServerPreSetup()`).

### Extra template vars or cluster options

```go
lazyvcluster.LazyVCluster(ctx, name, yaml,
    lazyvcluster.WithExtraTemplateVars(map[string]any{"MyFlag": "value"}),
    lazyvcluster.WithExtraClusterOpts(myProviderOpt),
)
```

## Custom Linters

The e2e-next tests are checked by custom golangci-lint plugins (built via `golangci-lint custom` from `.custom-gcl.yml`). These run in CI and locally via `just lint ./e2e-next/...`.

| Linter | What it checks |
|--------|---------------|
| `describefunc` | Spec functions in `test_*` packages must not call `Describe()` with `cluster.Use()`. Cluster binding belongs in suite files, not in specs. This is critical because spec functions are imported by vcluster-pro via Go vendoring - if they contain `cluster.Use`, they hardcode OSS cluster references that conflict with Pro's own cluster definitions (different image, platform, `pro: true`). |
| `defercleanupcluster` | `cluster.Create()` calls must have a matching `DeferCleanup(cluster.Destroy(...))`. |
| `defercleanupctx` | `DeferCleanup` must not be called with a `setup.Func`; use `e2e.DeferCleanupCtx(ctx, fn)` instead. |
| `ginkgoreturnctx` | Ginkgo node functions that reassign `context.Context` must return it. |

If a linter flags your code, the error message explains the fix. Source code for all linters lives in [loft-sh/e2e-framework/linters/](https://github.com/loft-sh/e2e-framework/tree/main/linters).

## Cross-repo Usage (vcluster-pro)

Spec functions are exported and carry their own labels. vcluster-pro imports them and registers against its own vCluster suites using the same `Ordered` + `BeforeAll(LazyVCluster(...))` pattern (see the equivalent `setup/lazyvcluster` helper in vcluster-pro):

```go
// vcluster-pro/e2e-next/suite_deploy_etcd_test.go
//go:embed vcluster-deploy-etcd.yaml
var deployEtcdVClusterYAML string

const deployEtcdVClusterName = "deploy-etcd-vcluster"

func init() { suiteDeployEtcdVCluster() }

func suiteDeployEtcdVCluster() {
    Describe("deploy-etcd-vcluster", Ordered,
        cluster.Use(proClusters.HostCluster),
        func() {
            BeforeAll(func(ctx context.Context) context.Context {
                return lazyvcluster.LazyVCluster(ctx, deployEtcdVClusterName, deployEtcdVClusterYAML)
            })

            test_core.PodSyncSpec()
            test_core.PVCSyncSpec()
            // pro-specific specs...
        },
    )
}
```
