set positional-arguments

timestamp := `date +%s`

alias c := create
alias d := delete

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
lint *ARGS:
  golangci-lint run {{ARGS}}

# --- Kind ---

# Create a kubernetes cluster using the specified distro
create distro="kind":
  just create-{{distro}}

# Create a kubernetes cluster for the specified distro
delete distro="kind":
  just delete-{{distro}}

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

# Embed the charts into the vcluster binary
[private]
embed-charts version="0.0.0":
  RELEASE_VERSION={{ version }} go generate -tags embed_charts ./...

# Run e2e tests
e2e distribution="k3s" path="./test/e2e" multinamespace="false": create-kind && delete-kind
  echo "Execute test suites ({{ distribution }}, {{ path }}, {{ multinamespace }})"

  TELEMETRY_PRIVATE_KEY="" goreleaser build --snapshot --clean

  cp dist/vcluster_linux_$(go env GOARCH | sed s/amd64/amd64_v1/g)/vcluster ./vcluster
  docker build -t vcluster:e2e-latest -f Dockerfile.release --build-arg TARGETARCH=$(uname -m) --build-arg TARGETOS=linux .
  rm ./vcluster

  kind load docker-image vcluster:e2e-latest -n vcluster

  cp test/commonValues.yaml dist/commonValues.yaml

  sed -i.bak "s|REPLACE_IMAGE_NAME|vcluster:e2e-latest|g" dist/commonValues.yaml
  rm dist/commonValues.yaml.bak

  sed -i.bak "s|kind-control-plane|vcluster-control-plane|g" dist/commonValues.yaml
  rm dist/commonValues.yaml.bak

  ./dist/vcluster-cli_$(go env GOOS)_$(go env GOARCH | sed s/amd64/amd64_v1/g)/vcluster \
    create vcluster -n vcluster \
    --create-namespace \
    --debug \
    --connect=false \
    --distro={{ distribution }} \
    --local-chart-dir ./charts/{{ distribution }} \
    -f ./dist/commonValues.yaml \
    -f {{ path }}/values.yaml \
    $([[ "{{ multinamespace }}" = "true" ]] && echo "-f ./test/multins_values.yaml" || echo "")

  kubectl wait --for=condition=ready pod -l app=vcluster -n vcluster --timeout=300s

  cd {{path}} && VCLUSTER_SUFFIX=vcluster \
    VCLUSTER_NAME=vcluster \
    VCLUSTER_NAMESPACE=vcluster \
    MULTINAMESPACE_MODE={{ multinamespace }} \
    KIND_NAME=vcluster \
    go test -v -ginkgo.v -ginkgo.skip='.*NetworkPolicy.*' -ginkgo.fail-fast
