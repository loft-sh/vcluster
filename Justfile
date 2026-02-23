set positional-arguments

timestamp := `date +%s`

GOOS := env("GOOS", `go env GOOS`)
GOARCH := env("GOARCH", `go env GOARCH`)
GOBIN := env("GOBIN", `go env GOPATH`+"/bin")

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

# --- Kind ---

# Create a local kind cluster
create-kind:
  kind delete cluster -n vcluster
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

# Run e2e tests
e2e distribution="k8s" path="./test/e2e" multinamespace="false": create-kind setup-csi-volume-snapshots && delete-kind
  echo "Execute test suites ({{ distribution }}, {{ path }}, {{ multinamespace }})"

  TELEMETRY_PRIVATE_KEY="" goreleaser build --snapshot --clean
  cp dist/vcluster_linux_$(go env GOARCH | sed s/amd64/amd64_v1/g | sed s/arm64/arm64_v8.0/g)/vcluster ./vcluster
  docker build -t vcluster:e2e-latest -f Dockerfile.release --build-arg TARGETARCH=$(uname -m | sed s/x86_64/amd64/g) --build-arg TARGETOS=linux .
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
  ./dist/vcluster-cli_$(go env GOOS)_$(go env GOARCH | sed s/amd64/amd64_v1/g | sed s/arm64/arm64_v8.0/g)/vcluster \
    create vcluster -n vcluster \
    --create-namespace \
    --debug \
    --connect=false \
    --local-chart-dir ./chart/ \
    -f ./dist/commonValues.yaml \
    -f {{ path }}/values.yaml \
    $([[ "{{ multinamespace }}" = "true" ]] && echo "-f ./test/multins_values.yaml" || echo "")

  [ -f "{{ path }}/host-resources.yaml" ] && kubectl apply -f "{{ path }}/host-resources.yaml" -n vcluster
  kubectl wait --for=condition=ready pod -l app=vcluster -n vcluster --timeout=300s

  cd {{path}} && VCLUSTER_SUFFIX=vcluster \
    VCLUSTER_NAME=vcluster \
    VCLUSTER_NAMESPACE=vcluster \
    MULTINAMESPACE_MODE={{ multinamespace }} \
    VCLUSTER_BACKGROUND_PROXY_IMAGE=vcluster:e2e-latest \
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
