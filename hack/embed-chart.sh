#!/usr/bin/env bash

# This script generates packaged helm releases to embed in the vcluster binary

set -eu

VCLUSTER_ROOT="$(dirname ${0})/.."
RELEASE_VERSION="${RELEASE_VERSION:-0.0.1}"
RELEASE_VERSION="${RELEASE_VERSION#"v"}" # remove "v" prefix
EMBED_DIR="${VCLUSTER_ROOT}/pkg/embed/chart"

rm -rfv "${EMBED_DIR}"
mkdir "${EMBED_DIR}"

touch "${EMBED_DIR}/gitkeep.tgz"
helm package --version "${RELEASE_VERSION}" "${VCLUSTER_ROOT}/chart" -d "${EMBED_DIR}"
