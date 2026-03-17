#!/usr/bin/env bash
set -eo pipefail

# Library for homebrew tap version drift detection.
# Source this file to use functions, or execute directly for the compare step.

# Strip v prefix for consistent version comparison.
normalize_version() {
  echo "${1#v}"
}

# Fetch version string from a homebrew tap formula with retry.
fetch_formula_version() {
  local formula="$1"
  local version=""
  for attempt in 1 2 3; do
    version=$(curl -sfL --max-time 10 \
      "https://raw.githubusercontent.com/loft-sh/homebrew-tap/main/Formula/${formula}.rb" \
      | grep -oP 'version "\K[^"]+') && break
    version=""
    delay=$((1 << attempt))
    echo "::warning::Attempt $attempt/3 to fetch ${formula} version failed, retrying in ${delay}s..." >&2
    sleep "$delay"
  done
  echo "$version"
}

# Compare release tag against tap versions.
# Sets globals: DRIFTED (true/false), DRIFT_DETAILS (human-readable summary).
# Args: release_tag tap_vcluster [tap_experimental]
compare_versions() {
  local release tap_vcluster tap_experimental
  release=$(normalize_version "$1")
  tap_vcluster=$(normalize_version "$2")
  tap_experimental=$(normalize_version "${3:-}")

  DRIFTED="false"
  DRIFT_DETAILS=""

  if [[ "$release" != "$tap_vcluster" ]]; then
    echo "::warning::vcluster tap drift: release=${release} tap=${tap_vcluster}"
    DRIFTED="true"
    DRIFT_DETAILS="vcluster: release=${release} tap=${tap_vcluster}"
  fi

  if [[ -n "$tap_experimental" && "$release" != "$tap_experimental" ]]; then
    echo "::warning::vcluster-experimental tap drift: release=${release} tap=${tap_experimental}"
    DRIFTED="true"
    if [[ -n "$DRIFT_DETAILS" ]]; then
      DRIFT_DETAILS="${DRIFT_DETAILS}\nvcluster-experimental: release=${release} tap=${tap_experimental}"
    else
      DRIFT_DETAILS="vcluster-experimental: release=${release} tap=${tap_experimental}"
    fi
  fi
}

# Main entrypoint — runs only when executed directly, not when sourced.
# Expects env vars: RELEASE_TAG, TAP_VCLUSTER, TAP_EXPERIMENTAL, GITHUB_OUTPUT.
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  compare_versions "$RELEASE_TAG" "$TAP_VCLUSTER" "${TAP_EXPERIMENTAL:-}"

  if [[ "$DRIFTED" == "false" ]]; then
    echo "No drift detected. release=$(normalize_version "$RELEASE_TAG") tap=$(normalize_version "$TAP_VCLUSTER")"
    echo "drifted=false" >> "$GITHUB_OUTPUT"
  else
    echo "drifted=true" >> "$GITHUB_OUTPUT"
    {
      echo "details<<EOF"
      echo -e "$DRIFT_DETAILS"
      echo "EOF"
    } >> "$GITHUB_OUTPUT"
  fi
fi
