set positional-arguments

timestamp := `date +%s`

_default:
  @just --list

# --- Build ---

# Build the vcluster binary
build-snapshot:
  TELEMETRY_PRIVATE_KEY="" goreleaser build --snapshot --clean --single-target

# Build the vcluster release binary in snapshot mode
release-snapshot:
  TELEMETRY_PRIVATE_KEY="" goreleaser release --snapshot --clean

# --- Code quality ---

# Run golangci-lint for all packages
lint:
  golangci-lint run $@

# --- Kind ---

# Create a local kind cluster
create-kind:
  kind create cluster -n vcluster

# Delete the local kind cluster
delete-kind:
  kind delete cluster -n vcluster
