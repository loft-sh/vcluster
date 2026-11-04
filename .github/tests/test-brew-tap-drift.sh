#!/usr/bin/env bash
set -eo pipefail

# Tests for .github/scripts/brew-tap-drift.sh
# Sources the library and exercises its functions directly.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../scripts/brew-tap-drift.sh"

PASS=0
FAIL=0

assert_drift() {
  local release="$1" tap="$2" expected_drifted="$3" description="$4"

  reset_drift
  compare_versions "$release" "$tap"

  if [[ "$DRIFTED" == "$expected_drifted" ]]; then
    printf "  PASS: %s (release=%s tap=%s drifted=%s)\n" \
      "$description" "$(normalize_version "$release")" "$(normalize_version "$tap")" "$DRIFTED"
    PASS=$((PASS + 1))
  else
    printf "  FAIL: %s (release=%s tap=%s expected=%s got=%s)\n" \
      "$description" "$(normalize_version "$release")" "$(normalize_version "$tap")" "$expected_drifted" "$DRIFTED"
    FAIL=$((FAIL + 1))
  fi
}

assert_experimental_drift() {
  local release="$1" tap="$2" experimental="$3" expected_drifted="$4" description="$5"

  reset_drift
  compare_versions "$release" "$tap" "$experimental"

  if [[ "$DRIFTED" == "$expected_drifted" ]]; then
    printf "  PASS: %s (drifted=%s)\n" "$description" "$DRIFTED"
    PASS=$((PASS + 1))
  else
    printf "  FAIL: %s (expected=%s got=%s)\n" "$description" "$expected_drifted" "$DRIFTED"
    FAIL=$((FAIL + 1))
  fi
}

assert_pointer_drift() {
  local oss="$1" pro="$2" expected_drifted="$3" description="$4"

  reset_drift
  compare_release_pointers "$oss" "$pro"

  if [[ "$DRIFTED" == "$expected_drifted" ]]; then
    printf "  PASS: %s (pro=%s oss=%s drifted=%s)\n" \
      "$description" "$(normalize_version "$pro")" "$(normalize_version "$oss")" "$DRIFTED"
    PASS=$((PASS + 1))
  else
    printf "  FAIL: %s (pro=%s oss=%s expected=%s got=%s)\n" \
      "$description" "$(normalize_version "$pro")" "$(normalize_version "$oss")" "$expected_drifted" "$DRIFTED"
    FAIL=$((FAIL + 1))
  fi
}

# The Slack body is the only output of this monitor, so the message matters as
# much as the boolean. Runs the same two comparisons the entrypoint runs, in the
# same order, and asserts every expected line is present.
# Args: release tap_vcluster tap_experimental pro_release description line...
assert_details() {
  local release="$1" tap="$2" experimental="$3" pro="$4" description="$5"
  shift 5

  reset_drift
  compare_versions "$release" "$tap" "$experimental"
  compare_release_pointers "$release" "$pro"

  local expanded missing=() line
  # $DRIFT_DETAILS holds literal \n that the entrypoint expands with echo -e
  # before writing $GITHUB_OUTPUT; expand it the same way so the assertions see
  # what Slack receives.
  expanded=$(echo -e "$DRIFT_DETAILS")

  for line in "$@"; do
    grep -qxF -- "$line" <<<"$expanded" || missing+=("$line")
  done

  if [[ ${#missing[@]} -eq 0 ]]; then
    printf "  PASS: %s\n" "$description"
    PASS=$((PASS + 1))
  else
    printf "  FAIL: %s (missing %s in: %s)\n" "$description" "${missing[*]}" "${expanded//$'\n'/ | }"
    FAIL=$((FAIL + 1))
  fi
}

assert_normalize() {
  local input="$1" expected="$2" description="$3"
  local actual
  actual=$(normalize_version "$input")

  if [[ "$actual" == "$expected" ]]; then
    printf "  PASS: %s (%s -> %s)\n" "$description" "$input" "$actual"
    PASS=$((PASS + 1))
  else
    printf "  FAIL: %s (%s -> expected=%s got=%s)\n" "$description" "$input" "$expected" "$actual"
    FAIL=$((FAIL + 1))
  fi
}

printf "=== normalize_version ===\n\n"
assert_normalize "v0.33.0" "0.33.0" "strips v prefix"
assert_normalize "0.33.0"  "0.33.0" "no-op without v prefix"
assert_normalize "v1.0.0"  "1.0.0"  "strips v from major version"

printf "\n=== compare_versions (single formula) ===\n\n"

# matching versions — no drift
assert_drift "v0.23.0" "v0.23.0" "false" "matching versions show no drift"
assert_drift "v1.0.0"  "v1.0.0"  "false" "matching major versions show no drift"

# tap behind release — drift
assert_drift "v0.23.0" "v0.22.0" "true" "tap behind release is drift"
assert_drift "v0.23.1" "v0.23.0" "true" "tap behind by patch is drift"
assert_drift "v1.0.0"  "v0.23.0" "true" "tap behind by major is drift"

# tap ahead of release — also drift (manual tap edit)
assert_drift "v0.22.0" "v0.23.0" "true" "tap ahead of release is drift"

# different patch versions
assert_drift "v0.23.2" "v0.23.1" "true" "different patch versions is drift"

# mixed v prefix — should match after normalization
assert_drift "v0.33.0" "0.33.0" "false" "tag with v vs tap without v shows no drift"
assert_drift "0.33.0" "v0.33.0" "false" "tag without v vs tap with v shows no drift"
assert_drift "v0.33.0" "0.32.0" "true" "mixed prefix with version mismatch is drift"

printf "\n=== compare_versions (with experimental) ===\n\n"

assert_experimental_drift "v0.33.0" "0.33.0" "0.33.0" "false" "all match — no drift"
assert_experimental_drift "v0.33.0" "0.33.0" "0.32.0" "true"  "experimental behind — drift"
assert_experimental_drift "v0.33.0" "0.32.0" "0.33.0" "true"  "vcluster behind — drift"
assert_experimental_drift "v0.33.0" "0.33.0" ""        "false" "empty experimental — no drift"

printf "\n=== compare_release_pointers (coordinated promotion) ===\n\n"

assert_pointer_drift "v0.36.1" "v0.36.1" "false" "matching Pro and OSS Latest pointers show no drift"
assert_pointer_drift "v0.36.1" "v0.37.0" "true" "Pro ahead of OSS is drift"
assert_pointer_drift "v0.37.0" "v0.36.1" "true" "OSS ahead of Pro is drift"
assert_pointer_drift "v0.36.1" "" "false" "absent Pro pointer is not drift"

printf "\n=== DRIFT_DETAILS (the Slack body) ===\n\n"

assert_details "v0.23.0" "v0.22.0" "" "" "tap drift alone names the formula" \
  "vcluster: release=0.23.0 tap=0.22.0"

assert_details "v0.23.0" "v0.23.0" "" "v0.24.0" "pointer split alone names both repos" \
  "release pointers: pro=0.24.0 oss=0.23.0"

# Two dimensions at once: the second record_drift must append rather than
# overwrite, which a boolean-only assertion cannot see.
assert_details "v0.23.0" "v0.22.0" "" "v0.24.0" "tap drift and pointer split both appear" \
  "vcluster: release=0.23.0 tap=0.22.0" \
  "release pointers: pro=0.24.0 oss=0.23.0"

assert_details "v0.23.0" "v0.22.0" "v0.21.0" "v0.24.0" "all three dimensions appear" \
  "vcluster: release=0.23.0 tap=0.22.0" \
  "vcluster-experimental: release=0.23.0 tap=0.21.0" \
  "release pointers: pro=0.24.0 oss=0.23.0"

printf "\nResults: %d passed, %d failed\n" "$PASS" "$FAIL"
[[ "$FAIL" -eq 0 ]]
