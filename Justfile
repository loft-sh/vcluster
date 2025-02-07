set positional-arguments

[private]
alias align := check-structalign

timestamp := `date +%s`

alias c := create
alias d := delete

_default:
  @just --list

# --- Build ---

# Build the vcluster binary
build-snapshot:
  TELEMETRY_PRIVATE_KEY="" goreleaser build --id vcluster --snapshot --clean

# Build the vcluster release binary in snapshot mode
release-snapshot: gen-license-report
  TELEMETRY_PRIVATE_KEY="" goreleaser release --snapshot --clean

# --- Code quality ---

# Run golangci-lint for all packages
lint *ARGS:
  [ -f ./custom-gcl ] || golangci-lint custom
  ./custom-gcl cache clean
  ./custom-gcl run {{ARGS}}

# Check struct memory alignment and print potential improvements
[no-exit-message]
check-structalign *ARGS:
  go run github.com/dkorunic/betteralign/cmd/betteralign@latest {{ARGS}} ./...

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

# Generate versioned vCluster image files for multiple versions and distros
[private]
generate-matrix-specific-images version="0.0.0":
  #!/usr/bin/env bash

  distros=("k8s" "k3s" "k0s")
  versions=("1.30" "1.29" "1.28")

  for distro in "${distros[@]}"; do
    for version in "${versions[@]}"; do
      go run -mod vendor ./hack/assets/separate/main.go -kubernetes-distro=$distro -kubernetes-version=$version -vcluster-version={{ version }} > ./release/vcluster-images-$distro-$version.txt
    done
  done

# Generate the CLI docs
generate-cli-docs:
  go run -mod vendor -tags pro ./hack/docs/main.go

# Generate the vcluster.yaml config schema
generate-config-schema:
  go run -mod vendor ./hack/schema/main.go

# Embed the chart into the vcluster binary
[private]
embed-chart version="0.0.0":
  RELEASE_VERSION={{ version }} go generate -tags embed_chart ./...

# Run e2e tests
e2e distribution="k3s" path="./test/e2e" multinamespace="false": create-kind && delete-kind
  echo "Execute test suites ({{ distribution }}, {{ path }}, {{ multinamespace }})"

  TELEMETRY_PRIVATE_KEY="" goreleaser build --snapshot --clean

  cp dist/vcluster_linux_$(go env GOARCH | sed s/amd64/amd64_v1/g)/vcluster ./vcluster
  docker build -t vcluster:e2e-latest -f Dockerfile.release --build-arg TARGETARCH=$(uname -m) --build-arg TARGETOS=linux .
  rm ./vcluster

  kind load docker-image vcluster:e2e-latest -n vcluster

  cp test/commonValues.yaml dist/commonValues.yaml

  sed -i.bak "s|REPLACE_REPOSITORY_NAME|vcluster|g" dist/commonValues.yaml
  sed -i.bak "s|REPLACE_TAG_NAME|e2e-latest|g" dist/commonValues.yaml
  yq eval -i '.controlPlane.distro.{{distribution}}.enabled = true' dist/commonValues.yaml
  rm dist/commonValues.yaml.bak

  sed -i.bak "s|kind-control-plane|vcluster-control-plane|g" dist/commonValues.yaml
  rm dist/commonValues.yaml.bak

  kubectl create namespace from-host-sync-test
  kubectl create namespace from-host-sync-test-2

  ./dist/vcluster-cli_$(go env GOOS)_$(go env GOARCH | sed s/amd64/amd64_v1/g)/vcluster \
    create vcluster -n vcluster \
    --create-namespace \
    --debug \
    --connect=false \
    --local-chart-dir ./chart/ \
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


cli version="0.0.0" *ARGS="":
  RELEASE_VERSION={{ version }} go generate -tags embed_chart ./...
  go run -tags embed_chart -mod vendor -ldflags "-X main.version={{ version }}" ./cmd/vclusterctl/main.go {{ ARGS }}

# --- Docs ---

# Version the docs for the given version
docs-version id="pro" version="1.0.0":
  yarn docusaurus docs:version {{version}}

generate-compatibility:
  go run hack/compat-matrix/main.go generate docs/pages/deploying-vclusters/compat-matrix.mdx

validate-compat-matrix:
  go run hack/compat-matrix/main.go validate docs/pages/deploying-vclusters/compat-matrix.mdx

gen-license-report:
  rm -rf ./licenses
  (cd ./cmd/vclusterctl/cmd/credits/licenses && find . -type d -exec rm -rf "{}" \;) || true

  go-licenses save --save_path=./licenses --ignore github.com/loft-sh ./...

  cp -r ./licenses ./cmd/vclusterctl/cmd/credits

build-dev-image tag="":
  TELEMETRY_PRIVATE_KEY="" goreleaser build --snapshot --clean

  cp dist/vcluster_linux_$(go env GOARCH | sed s/amd64/amd64_v1/g)/vcluster ./vcluster
  docker build -t vcluster:dev-{{tag}} -f Dockerfile.release --build-arg TARGETARCH=$(uname -m) --build-arg TARGETOS=linux .
  rm ./vcluster

run-conformance k8s_version="1.31.1" mode="conformance-lite" tag="conf": (create-conformance k8s_version) (build-dev-image tag)
  minikube image load vcluster:dev-{{tag}}

  vcluster create vcluster -n vcluster -f ./conformance/v1.31/vcluster.yaml

  sonobuoy run --mode={{mode}} --level=debug

conformance-status:
  sonobuoy status

conformance-logs:
  sonobuoy logs

dev-conformance *ARGS:
  devspace dev --profile test-conformance --namespace vcluster {{ARGS}}

create-conformance k8s_version="1.31.1":
  minikube start --kubernetes-version {{k8s_version}} --nodes=2
  minikube addons enable metrics-server

delete-conformance:
  -minikube delete

recreate-conformance: delete-conformance create-conformance
