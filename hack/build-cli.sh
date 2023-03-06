#!/usr/bin/env bash
# This script will build vcluster and calculate hash for each
# (VCLUSTER_BUILD_PLATFORMS, VCLUSTER_BUILD_ARCHS) pair.
# VCLUSTER_BUILD_PLATFORMS="linux" VCLUSTER_BUILD_ARCHS="amd64" ./hack/build-all.bash
# can be called to build only for linux-amd64

set -e

export GO111MODULE=on
export GOFLAGS=-mod=vendor

# Update vendor directory
# go mod vendor

VCLUSTER_ROOT=$(git rev-parse --show-toplevel)
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null)
DATE=$(date "+%Y-%m-%d")
BUILD_PLATFORM=$(uname -a | awk '{print tolower($1);}')

echo "Current working directory is $(pwd)"
echo "PATH is $PATH"
echo "GOPATH is $GOPATH"

if [[ "$(pwd)" != "${VCLUSTER_ROOT}" ]]; then
  echo "you are not in the root of the repo" 1>&2
  echo "please cd to ${VCLUSTER_ROOT} before running this script" 1>&2
  exit 1
fi

RELEASE_VERSION="${RELEASE_VERSION}" go generate ${VCLUSTER_ROOT}/...

GO_BUILD_CMD="go build -a"
GO_BUILD_LDFLAGS="-s -w -X main.commitHash=${COMMIT_HASH} -X main.buildDate=${DATE} -X main.version=${RELEASE_VERSION}"

if [[ -z "${VCLUSTER_BUILD_PLATFORMS}" ]]; then
    VCLUSTER_BUILD_PLATFORMS="linux windows darwin"
fi

if [[ -z "${VCLUSTER_BUILD_ARCHS}" ]]; then
    VCLUSTER_BUILD_ARCHS="amd64 386 arm64"
fi

# Create the release directory
mkdir -p "${VCLUSTER_ROOT}/release"

# copy assets
cp -a "${VCLUSTER_ROOT}/assets/." "${VCLUSTER_ROOT}/release/"

# generate vcluster-images.txt
go run -mod vendor "${VCLUSTER_ROOT}/hack/assets/main.go" ${RELEASE_VERSION} > "${VCLUSTER_ROOT}/release/vcluster-images.txt"

for OS in ${VCLUSTER_BUILD_PLATFORMS[@]}; do
  for ARCH in ${VCLUSTER_BUILD_ARCHS[@]}; do
    NAME="vcluster-${OS}-${ARCH}"
    if [[ "${OS}" == "windows" ]]; then
      NAME="${NAME}.exe"
    fi
    
    # darwin 386 is deprecated and shouldn't be used anymore
    if [[ "${ARCH}" == "386" && "${OS}" == "darwin" ]]; then
        echo "Building for ${OS}/${ARCH} not supported."
        continue
    fi
    
    # arm64 build is only supported for darwin
    if [[ "${ARCH}" == "arm64" && "${OS}" == "windows" ]]; then
        echo "Building for ${OS}/${ARCH} not supported."
        continue
    fi

    echo "Building for ${OS}/${ARCH}"
    GOARCH=${ARCH} GOOS=${OS} ${GO_BUILD_CMD} -ldflags "${GO_BUILD_LDFLAGS}"\
      -o "${VCLUSTER_ROOT}/release/${NAME}" cmd/vclusterctl/main.go
    shasum -a 256 "${VCLUSTER_ROOT}/release/${NAME}" | cut -d ' ' -f 1 > "${VCLUSTER_ROOT}/release/${NAME}".sha256
    cosign sign-blob --yes --output-signature "${VCLUSTER_ROOT}/release/${NAME}".sha256.sig --output-certificate "${VCLUSTER_ROOT}/release/${NAME}".sha256.pem "${VCLUSTER_ROOT}/release/${NAME}".sha256
  done
done
