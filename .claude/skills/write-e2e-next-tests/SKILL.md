---
name: write-e2e-next-tests
description: Write new e2e-next tests for vCluster. Use when adding test specs, understanding test patterns, or creating new test categories in e2e-next.
keywords:
  - e2e
  - test
  - write
  - ginkgo
  - gomega
  - test pattern
---

# Writing e2e-next Tests

## Framework

- **Runner**: Ginkgo v2 (`github.com/onsi/ginkgo/v2`)
- **Assertions**: Gomega (`github.com/onsi/gomega`)
- **Cluster lifecycle**: `github.com/loft-sh/e2e-framework`

## Adding a New Test

1. Create `e2e-next/test_<category>/test_<name>.go`
2. Add side-effect import in `e2e-next/e2e_suite_test.go`:
   ```go
   _ "github.com/loft-sh/vcluster/e2e-next/test_<category>"
   ```
3. Add label in `e2e-next/labels/labels.go` if new category

## Test Structure

```go
package test_mycategory

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "github.com/loft-sh/vcluster/e2e-next/labels"
    "github.com/loft-sh/vcluster/e2e-next/constants"
)

var _ = Describe("My feature", labels.MyLabel, func() {
    It("should do something", func() {
        Eventually(func(g Gomega) {
            // assertions
        }).
            WithPolling(constants.PollingInterval).
            WithTimeout(constants.PollingTimeout).
            Should(Succeed())
    })
})
```

## Kind-based Tests (host cluster required)

Define clusters in `e2e-next/clusters/clusters.go`:

```go
var MyVCluster = vcluster.Define(
    vcluster.WithName("my-test-vcluster"),
    vcluster.WithVClusterYAML(DefaultVClusterYAML),
    vcluster.WithOptions(DefaultVClusterOptions...),
    vcluster.WithDependencies(HostCluster),
)
```

Get clients in tests:
```go
cluster.Use(clusters.MyVCluster)  // decorator on Describe
hostClient := cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
vClient := cluster.CurrentKubeClientFrom(ctx)
```

## Docker/Vind Tests (no host cluster)

Use `os/exec` directly — no e2e-framework cluster provider needed.

```go
// Use DeferCleanup, not AfterEach
DeferCleanup(func() {
    runVCluster(ctx, "delete", name, "--driver", "docker", "--ignore-not-found")
})
```

- Generate names: `"e2e-vind-" + random.String(6)` (`pkg/util/random`)
- Docker naming constants (`pkg/constants/cli.go`):
  - Control plane: `vcluster.cp.<name>`
  - Worker nodes: `vcluster.node.<name>.<worker>`
  - Load balancers: `vcluster.lb.<name>.<lb>`
  - Network: `vcluster.<name>`
  - Volumes: `vcluster.cp.<name>.{var,etc,bin,cni-bin}`
- Kube context: `vcluster-docker_<name>`

## Key Directories

| Path | Purpose |
|------|---------|
| `clusters/clusters.go` | Cluster definitions (Kind + vCluster) |
| `constants/timeouts.go` | `PollingInterval` (2s), `PollingTimeout` (60s), etc. |
| `constants/image.go` | `GetVClusterImage()`, `GetRepository()`, `GetTag()` |
| `labels/labels.go` | Test labels for filtering |
| `setup/template/template.go` | YAML template rendering (`MustRender`) |
| `init/init.go` | Ginkgo tree initialization (auto-loaded) |

## Conventions

- Use `DeferCleanup` for teardown, not `AfterEach`
- Use `Eventually` with `WithPolling`/`WithTimeout` for async assertions
- Always use `By("description", func() { ... })` with a closure — never bare `By("description")`
- Random cluster names prevent collisions in parallel runs
- Labels enable selective test execution (`--label-filter`)
- Use `Expect(err).To(Succeed())` instead of `Expect(err).NotTo(HaveOccurred())` — applies to both `Expect` and `g.Expect`
