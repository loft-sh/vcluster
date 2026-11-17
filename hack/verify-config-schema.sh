#!/usr/bin/env bash

# Fails when chart/values.yaml or chart/values.schema.json is not what
# hack/schema/main.go generates from the config types, which happens when they
# change without a regen. Run by the lint workflows on every pull request.

set -euo pipefail

VCLUSTER_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${VCLUSTER_ROOT}"

GENERATED=(chart/values.yaml chart/values.schema.json)

# Comparing before/after instead of requiring a clean tree keeps the script
# usable on a work in progress checkout: it reports what the generator changes,
# not what the developer changed.
before="$(mktemp -d)"
trap 'rm -rf "${before}"' EXIT
for file in "${GENERATED[@]}"; do
  mkdir -p "${before}/$(dirname "${file}")"
  cp "${file}" "${before}/${file}"
done

GOWORK=off go run -mod vendor ./hack/schema/main.go

stale=0
for file in "${GENERATED[@]}"; do
  if ! diff -u --label "${file} (committed)" --label "${file} (generated)" "${before}/${file}" "${file}"; then
    stale=1
  fi
done

if [[ "${stale}" -ne 0 ]]; then
  {
    echo
    echo "hack/schema/main.go changed the files above, so they are out of date with the config types."
    echo "Run 'just generate-config-schema' and commit the result."
  } >&2
  exit 1
fi

echo "config schema is up to date."
