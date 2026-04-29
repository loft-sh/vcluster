set positional-arguments

timestamp := `date +%s`

GOOS := env("GOOS", `go env GOOS`)
GOARCH := env("GOARCH", `go env GOARCH`)
GOBIN := env("GOBIN", `go env GOPATH`+"/bin")
PRIVATE_GO_ENV := "GOPRIVATE=github.com/loft-sh/* GONOSUMDB=github.com/loft-sh/*"

DIST_FOLDER := if GOARCH == "amd64" { "dist/vcluster_linux_amd64_v1" } else if GOARCH == "arm64" { "dist/vcluster_linux_arm64_v8.0" } else { "unknown" }
DIST_FOLDER_CLI := if GOARCH == "amd64" { "dist/vcluster-cli_" + GOOS + "_amd64_v1" } else if GOARCH == "arm64" { "dist/vcluster-cli_" + GOOS + "_arm64_v8.0" } else { "unknown" }

ASSETS_RUN := "go run -mod vendor ./hack/assets/cmd/main.go"

_default:
  @just --list

# --- Build ---

# Build the vcluster-cli binary
build-cli-snapshot:
  goreleaser build --id vcluster-cli --single-target --snapshot --clean
  mv {{DIST_FOLDER_CLI}}/vcluster {{GOBIN}}/vcluster

# Build the vcluster binary (we force linux here to allow building on mac os or windows)
build-snapshot:
  GOOS=linux goreleaser build --id vcluster --single-target --snapshot --clean
  cp Dockerfile.release {{DIST_FOLDER}}/Dockerfile
  cd {{DIST_FOLDER}} && docker buildx build --load . -t ghcr.io/loft-sh/vcluster:dev-next

# --- vind ---

# Create a local vind cluster
create-vind:
  vcluster delete vcluster --driver docker 2>/dev/null || true
  vcluster use driver docker
  vcluster create vcluster --connect=false
  vcluster connect vcluster --update-current

# Delete the local vind cluster
delete-vind:
  vcluster delete vcluster --driver docker

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

# Generate the vcluster latest/minimal images file
[private]
generate-vcluster-latest-images version="0.0.0":
  {{ASSETS_RUN}} {{ version }} > ./release/images.txt

# Generate the vcluster optional images file
[private]
generate-vcluster-optional-images version="0.0.0":
  {{ASSETS_RUN}} --optional {{ version }} > ./release/images-optional.txt

# Generate versioned vCluster image files for multiple versions and distros
[private]
generate-matrix-specific-images version="0.0.0":
  #!/usr/bin/env bash

  distros=(`{{ASSETS_RUN}} --list-distros`)
  versions=(`{{ASSETS_RUN}}  --list-versions`)
  for distro in "${distros[@]}"; do
    for version in "${versions[@]}"; do
      {{ASSETS_RUN}} --kubernetes-distro=$distro --kubernetes-version=$version {{ version }} > ./release/vcluster-images-$distro-$version.txt
    done
  done

# Generate the vcluster.yaml config schema
generate-config-schema:
  go run -mod vendor ./hack/schema/main.go

# Embed the chart into the vcluster binary
[private]
embed-chart version="0.0.0":
  RELEASE_VERSION={{ version }} go generate -tags embed_chart ./...

test-chart:
  helm unittest chart

# --- Lint ---

# Rebuild tools/golangci-lint if sources changed or binary is missing
[private]
_ensure-linters:
  #!/usr/bin/env bash
  if [ ! -f ./tools/golangci-lint ] || \
     [ -n "$(find .custom-gcl.yml -newer ./tools/golangci-lint \( -name '*.yml' \) 2>/dev/null | head -1)" ]; then
    echo "Custom linters changed - rebuilding tools/golangci-lint..."
    {{PRIVATE_GO_ENV}} golangci-lint custom
  fi

# Run golangci-lint for all packages
lint *ARGS: _ensure-linters
  ./tools/golangci-lint cache clean
  ./tools/golangci-lint run {{ARGS}} -- ./...

# Build the custom golangci-lint binary (required after linter code changes)
build-linters:
  golangci-lint custom

# Run custom linters against e2e-next (with autofix)
lint-e2e: _ensure-linters
  ./tools/golangci-lint run --fix -- ./e2e-next/...

setup-csi-volume-snapshots:
  # Deploy upstream CSI volume snapshot CRDs and snapshot-controller
  kubectl kustomize https://github.com/kubernetes-csi/external-snapshotter/client/config/crd | kubectl create -f -
  kubectl kustomize https://github.com/kubernetes-csi/external-snapshotter/deploy/kubernetes/snapshot-controller | kubectl create -f -

  # Deploy CSI driver, StorageClass and VolumeSnapshotClass
  temp_git_dir=$(mktemp -d) && \
    git clone https://github.com/kubernetes-csi/csi-driver-host-path.git $temp_git_dir && \
    $temp_git_dir/deploy/kubernetes-latest/deploy.sh && \
    kubectl apply -f $temp_git_dir/examples/csi-storageclass.yaml && \
    kubectl apply -f $temp_git_dir/examples/csi-volumesnapshotclass.yaml && \
    kubectl annotate volumesnapshotclass csi-hostpath-snapclass \
      snapshot.storage.kubernetes.io/is-default-class="true" && \
    rm -rf $temp_git_dir

  # wait for snapshot-controller to be ready
  kubectl wait --for=condition=Available -n kube-system deploy/snapshot-controller --timeout=60s

#e2e-next tests
@dev-e2e label-filter="core" image="ghcr.io/loft-sh/vcluster:dev-next" *ARGS='': \
  (setup label-filter image) \
  (run-e2e label-filter image "false") \
  (teardown label-filter)

@run-e2e label-filter="core" image="ghcr.io/loft-sh/vcluster:dev-next" teardown="true":
  ginkgo -timeout=0 -v --procs=8 --label-filter="{{label-filter}}" ./e2e-next -- --vcluster-image="{{image}}" --teardown={{teardown}}

@iterate-e2e label-filter="core" image="ghcr.io/loft-sh/vcluster:dev-next": \
  (run-e2e label-filter image "false")

@setup label-filter="core" image="ghcr.io/loft-sh/vcluster:dev-next":
  GINKGO_EDITOR_INTEGRATION=just ginkgo -timeout=0 -v --label-filter="{{label-filter}}" --silence-skips ./e2e-next -- --vcluster-image="{{image}}" --setup-only

@teardown label-filter="core":
  GINKGO_EDITOR_INTEGRATION=just ginkgo -timeout=0 -v --label-filter="{{label-filter}}" --silence-skips ./e2e-next -- --teardown-only

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

  cp dist/vcluster_linux_$(go env GOARCH | sed s/amd64/amd64_v1/g | sed s/arm64/arm64_v8.0/g)/vcluster ./vcluster
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
  minikube delete

recreate-conformance: delete-conformance create-conformance
