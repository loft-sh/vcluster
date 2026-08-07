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

# Start a fresh comparison. Each compare_* function only accumulates, so a
# caller running more than one must reset first.
reset_drift() {
  DRIFTED="false"
  DRIFT_DETAILS=""
}

# Also on load, so sourcing this library and calling a compare_* function
# without resetting first cannot leave DRIFTED unset. Unset is not benign: the
# entrypoint's `== "false"` test would fail and report drift with an empty body.
reset_drift

# Flag drift and append one line to the Slack body. Every dimension records
# through here so a new dimension cannot forget the append-vs-assign case.
record_drift() {
  local message="$1"
  echo "::warning::${message}"
  DRIFTED="true"
  if [[ -n "$DRIFT_DETAILS" ]]; then
    DRIFT_DETAILS="${DRIFT_DETAILS}\n${message}"
  else
    DRIFT_DETAILS="${message}"
  fi
}

# Compare the promoted OSS release tag against the tap formula versions.
# Accumulates into DRIFTED / DRIFT_DETAILS.
# Args: release_tag tap_vcluster [tap_experimental]
compare_versions() {
  local release tap_vcluster tap_experimental
  release=$(normalize_version "$1")
  tap_vcluster=$(normalize_version "$2")
  tap_experimental=$(normalize_version "${3:-}")

  if [[ "$release" != "$tap_vcluster" ]]; then
    record_drift "vcluster: release=${release} tap=${tap_vcluster}"
  fi

  if [[ -n "$tap_experimental" && "$release" != "$tap_experimental" ]]; then
    record_drift "vcluster-experimental: release=${release} tap=${tap_experimental}"
  fi
}

# Compare the two Latest pointers a promotion moves together. The promotion is
# coordinated across the private Pro release and the public OSS release, and the
# shared promoter keeps paired-repository edits advisory so it can finish safe
# work after a partial failure; detecting a pointer split here ensures such a
# warning cannot leave the daily monitor green merely because the old OSS
# pointer still matches the old formula.
# Accumulates into DRIFTED / DRIFT_DETAILS.
# Args: oss_release_tag pro_release_tag
compare_release_pointers() {
  local oss_release pro_release
  oss_release=$(normalize_version "$1")
  pro_release=$(normalize_version "${2:-}")

  if [[ -n "$pro_release" && "$oss_release" != "$pro_release" ]]; then
    record_drift "release pointers: pro=${pro_release} oss=${oss_release}"
  fi
}

# Main entrypoint — runs only when executed directly, not when sourced.
# Expects env vars: RELEASE_TAG, TAP_VCLUSTER, TAP_EXPERIMENTAL, GITHUB_OUTPUT.
# PRO_RELEASE_TAG is optional; when set, it must match RELEASE_TAG.
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  reset_drift
  compare_versions "$RELEASE_TAG" "$TAP_VCLUSTER" "${TAP_EXPERIMENTAL:-}"
  compare_release_pointers "$RELEASE_TAG" "${PRO_RELEASE_TAG:-}"

  if [[ "$DRIFTED" == "false" ]]; then
    echo "No drift detected. oss=$(normalize_version "$RELEASE_TAG") pro=$(normalize_version "${PRO_RELEASE_TAG:-}") tap=$(normalize_version "$TAP_VCLUSTER")"
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
