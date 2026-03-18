---
name: run-e2e-next-tests
description: Build and run e2e-next tests for vCluster using Ginkgo. Use when running, debugging, or filtering e2e tests in the e2e-next directory.
keywords:
  - e2e
  - test
  - ginkgo
  - vind
  - docker driver
---

# Running e2e-next Tests

## Build Prerequisites

```bash
# Build CLI binary (installs to $GOBIN/vcluster)
just build-cli-snapshot

# Build vcluster container image (ghcr.io/loft-sh/vcluster:dev-next)
just build-snapshot
```

## Running Tests

### Via Justfile (Kind-based tests)

```bash
just run-e2e                              # label=core, default image
just run-e2e "core && sync"               # custom label filter
just iterate-e2e "deploy"                 # skip teardown for iteration
just setup "core"                         # setup only, no tests
just teardown "core"                      # teardown only
```

### Via Ginkgo directly

```bash
ginkgo -timeout=0 -v --procs=4 --label-filter="<filter>" ./e2e-next -- [flags]
```

### Vind tests (no Kind cluster needed)

```bash
ginkgo -timeout=0 -v --label-filter="vind" ./e2e-next
```

Vind tests create Docker-based vclusters directly — no host Kubernetes cluster required.

## Label Filters

| Label | Scope |
|-------|-------|
| `pr` | Run on every PR |
| `core` | Core functionality |
| `sync` | Resource syncing |
| `deploy` | Helm charts, init manifests |
| `vind` | Docker driver lifecycle |

Combine with `&&`, `||`: `--label-filter="core && sync"`

## Test Binary Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--vcluster-image` | `ghcr.io/loft-sh/vcluster:dev-next` | Image to test |
| `--cluster-name` | `kind-cluster` | Kind cluster name |
| `--setup-only` | `false` | Setup environment, skip tests |
| `--teardown` | `true` | Teardown after tests |
| `--teardown-only` | `false` | Teardown only, skip tests |

Flags also accept env vars (via `ff`): `VCLUSTER_IMAGE`, `CLUSTER_NAME`, etc.

## Troubleshooting

- **Kind tests fail with "cluster not found"**: Run `just setup "core"` first
- **Vind tests fail with "container already exists"**: Clean up with `vcluster delete <name> --driver docker --ignore-not-found`
- **Stale Docker resources**: `docker ps -a --filter name=^vcluster.` to find leftovers
