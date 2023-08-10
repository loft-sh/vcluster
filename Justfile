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

# --- Build ---

# Clean the release folder
[private]
clean-release:
  rm -rf ./release

# Copy the assets to the release folder
[private]
copy-assets:
  mkdir -p ./release
  cp -a assets/. release/

# Generate the vcluster images file
[private]
generate-vcluster-images version="0.0.0":
  go run -mod vendor ./hack/assets/main.go {{ version }} > ./release/vcluster-images.txt
