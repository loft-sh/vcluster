# vCluster — Agent Instructions

See `CLAUDE.md` for general agent workflows and CI conventions.

## Cursor Cloud specific instructions

### Environment overview

This is a Go 1.26+ project (vendored deps). The main binary is `vcluster` (syncer) and there is also a CLI.

### Build & test

```bash
go build ./...
go vet ./...
go test ./pkg/... -count=1 -short -timeout 120s
```

### Lint

Full lint requires `just lint` which builds a custom golangci-lint binary with private linter plugins from `github.com/loft-sh/e2e-framework`. This requires GitHub auth to private repos. Without private repo access, use `go vet ./...` as a baseline lint check.

Standard golangci-lint (without custom plugins) can be used for basic checks:
```bash
golangci-lint run --no-config --enable=govet,errcheck,staticcheck --timeout 5m -- ./pkg/...
```

### Key caveats

- All Go dependencies are vendored — no network needed for `go build/test`.
- E2E tests (`e2e-next/`) require a Kubernetes cluster (kind) and cannot be run without Docker + kind.
- The Justfile recipes (`just --list`) document all supported workflows.
