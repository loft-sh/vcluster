# e2e-framework: Conditional Dependency Mechanism

How the framework decides which clusters to create based on which tests are selected.

## Overview

When you run `ginkgo --label-filter="sync"`, only the clusters needed by `sync`-labeled tests are created. This is automatic — no manual `if` guards needed.

## How it works

### 1. `PreviewSpecsAroundNode` collects focused labels

In `TestRunE2ETests`, the suite is configured with:

```go
RunSpecs(t, "vCluster E2E Suite",
    AroundNode(suite.PreviewSpecsAroundNode(config)),
    AroundNode(e2e.ContextualAroundNode),
)
```

`PreviewSpecsAroundNode` runs before the suite starts. It previews the spec tree using the current Ginkgo configuration (`--label-filter`, `--focus`, etc.) and collects the set of labels attached to all specs that **will** run. These are the `focusedLabels`.

### 2. `cluster.Use(dep)` attaches labels to Ginkgo nodes

In test files, `cluster.Use(clusters.NodesVCluster)` does two things:
- Attaches the dependency's label to the enclosing `Describe`/`Context` node
- Registers the dependency so the framework knows about it

### 3. `IsFocused` gates Setup/Import/Teardown

Each dependency (cluster or vcluster) has an `IsFocused()` method that checks whether its label appears in the `focusedLabels` set. The `Setup`, `Import`, and `Teardown` functions silently return `(ctx, nil)` when the dependency is **not** focused — skipping all provisioning work.

This is why `SynchronizedBeforeSuite` can call `setup.AllConcurrent(vc1.Setup, vc2.Setup, ...)` for all defined vclusters: unfocused ones simply no-op.

### 4. `--label-filter` and `--focus` interaction

- `--label-filter` selects which specs run based on their labels
- `--focus` further narrows by regex match on spec text
- `PreviewSpecsAroundNode` collects labels from whatever specs survive **both** filters
- Only dependencies whose labels appear in that collected set get created

## Concrete example

Given these test files:

```go
// test_core/sync/test_node.go
var _ = Describe("Node sync", labels.Core, labels.Sync, func() {
    cluster.Use(clusters.NodesVCluster)
    cluster.Use(clusters.HostCluster)
    // ...
})

// test_core/sync/test_servicesync.go
var _ = Describe("Service sync", labels.Core, labels.Sync, func() {
    cluster.Use(clusters.ServiceSyncVCluster)
    cluster.Use(clusters.HostCluster)
    // ...
})
```

Running with `--label-filter="sync" --focus="Node sync"`:

1. `--label-filter="sync"` selects both `Node sync` and `Service sync` (both have `labels.Sync`)
2. `--focus="Node sync"` narrows to just the `Node sync` describe
3. `PreviewSpecsAroundNode` collects labels from `Node sync` specs → finds `NodesVCluster` and `HostCluster` labels
4. Result: only `HostCluster` + `NodesVCluster` get created. `ServiceSyncVCluster` is skipped.

## Current label-to-cluster mapping

Based on `cluster.Use()` calls in the test files:

| Test file | Labels | Clusters used |
|---|---|---|
| `test_core/sync/test_node.go` | `core`, `sync` | `HostCluster`, `NodesVCluster` |
| `test_core/sync/test_k8sdefaultendpoint.go` | `core`, `sync` | `HostCluster`, `K8sDefaultEndpointVCluster` |
| `test_core/sync/test_servicesync.go` | `core`, `sync` | `HostCluster`, `ServiceSyncVCluster` |
| `test_core/sync/test_limit_classes.go` | `core`, `sync` | `HostCluster`, `LimitClassesVCluster` |
| `test_deploy/test_helm_charts.go` | `deploy` | `HelmChartsVCluster` |
| `test_deploy/test_init_manifests.go` | `deploy` | `InitManifestsVCluster` |

This table changes as tests are added. Verify with: `grep -r 'cluster.Use(' e2e-next/`
