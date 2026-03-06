# vCluster

Virtual Kubernetes clusters — lightweight, isolated clusters running inside a host cluster.

## Commands

| Command | Purpose |
|---------|---------|
| `just` | List all available targets |
| `go build -mod vendor ./cmd/vclusterctl/` | Build the CLI |
| `just vendor` | Vendor dependencies (**NOT** `go mod vendor`) |
| `just create-kind` | Create local kind cluster |
| `just delete-kind` | Tear down kind cluster |
| `golangci-lint run --timeout 5m ./pkg/<pkg>/...` | Lint a package |

- **Task runner**: `just` — use it for all common operations
- **Vendor rule**: always `just vendor` or `go work vendor` (NOT `go mod vendor`) — this repo uses Go workspaces
- **Deps**: vendored (`-mod vendor` on all build/run commands)

## Dev Environment

| Property | Value |
|----------|-------|
| Kind cluster | `vcluster` |
| Kube context | `kind-vcluster` |
| Container workdir | `/vcluster-dev` |
| Run command | `CGO_ENABLED=0 go run -mod vendor cmd/vcluster/main.go start` |
| Debug port | `2345` (container) → `2346` (host) |
| Prerequisites | kind, devspace, go, docker, kubectl |

### Workflow

```bash
just create-kind                  # create kind cluster "vcluster"
devspace dev                      # build image, deploy, file sync, open shell
# inside container:
CGO_ENABLED=0 go run -mod vendor cmd/vcluster/main.go start
# when done:
just delete-kind                  # cleanup
```

- First compile is slow (~5-15 min, no cache). Subsequent runs are fast.
- File sync is bidirectional — edit locally, changes appear in `/vcluster-dev`.
- DevSpace handles image build, Helm deploy, and file sync automatically.

## Build Guidelines

- Compile repos **sequentially**, not in parallel — parallel Go builds compete for CPU/memory.
- All builds use vendored deps (`-mod vendor`).
- Lint: `golangci-lint run --timeout 5m ./pkg/<package>/...` for quick checks.

## Testing

- **Always test end-to-end.** Unit tests and compilation alone are insufficient.
- Spin up a test environment, create resources, trigger the code paths you changed, and verify expected outcomes.

## Cross-Repo Dependencies

```
loft-enterprise (independent)
       ↑              ↑
       │              │
   vcluster ←── vcluster-pro
```

- This repo is consumed by `vcluster-pro` (vendored pin). It does NOT depend on `loft-enterprise`.
- **API drift warning**: `main` may diverge from what `vcluster-pro` expects. Be prepared to fix compatibility issues when vendoring.

### Cross-repo change procedure (vcluster → vcluster-pro)

```bash
# 1. Make and commit your change in vcluster
# 2. In vcluster-pro/go.mod, add:
#    replace github.com/loft-sh/vcluster => ../vcluster
# 3. Re-vendor and build:
go mod tidy && go mod vendor
CGO_ENABLED=0 go build -mod=vendor -trimpath ./cmd/vcluster/
# 4. Before committing vcluster-pro, REMOVE the replace directive
#    and update to the new commit hash instead:
#    go get github.com/loft-sh/vcluster@<commit-sha>
#    go mod tidy && go mod vendor
```

### CI testing against a vcluster branch

In a vcluster-pro PR, add this block to `.github/PULL_REQUEST_TEMPLATE.md`:

```
<!-- CONFIG:
VCLUSTER_OSS_BRANCH=<your_branch>
/-->
```

## Boundaries

### Always
- Use `-mod vendor` on all Go build/run commands
- Use `just vendor` or `go work vendor` to vendor (never `go mod vendor`)
- Test end-to-end, not just unit tests and compilation
- Run `golangci-lint` before submitting changes

### Ask first
- Adding or upgrading dependencies
- Modifying CI/CD configuration or Dockerfiles
- Changes to Helm chart templates or values schema

### Never
- Commit `replace` directives in go.mod
- Skip vendoring after dependency changes
- Force push to main
- Run `go mod vendor` (use `just vendor`)

## Code Investigation Techniques

- **Grep** for types, function names, error messages; trace call chains from entry points
- **Git history**: `git log --oneline -30 -- <path>`, `git blame <file>`, `git show <commit>`
- **K8s controllers**: find `Reconcile` methods, CRD types, watch predicates, RBAC roles, webhooks, informer caches
- **Cross-component deps**: check imports across `pkg/`, `cmd/`, `vendor/`
