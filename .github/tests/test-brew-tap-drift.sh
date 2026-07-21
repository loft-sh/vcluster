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

  compare_versions "$release" "$tap" "$experimental"

  if [[ "$DRIFTED" == "$expected_drifted" ]]; then
    printf "  PASS: %s (drifted=%s)\n" "$description" "$DRIFTED"
    PASS=$((PASS + 1))
  else
    printf "  FAIL: %s (expected=%s got=%s)\n" "$description" "$expected_drifted" "$DRIFTED"
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

printf "\nResults: %d passed, %d failed\n" "$PASS" "$FAIL"
[[ "$FAIL" -eq 0 ]]
