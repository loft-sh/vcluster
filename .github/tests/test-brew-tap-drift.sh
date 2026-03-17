#!/usr/bin/env bash
set -eo pipefail

# Tests the version comparison logic used in check-brew-tap-drift.yaml.
# Validates that drift is correctly detected between release tags and tap versions.

PASS=0
FAIL=0

check_drift() {
  local release="$1" tap="$2" expected_drifted="$3" description="$4"

  if [[ "$release" != "$tap" ]]; then
    actual_drifted="true"
  else
    actual_drifted="false"
  fi

  if [[ "$actual_drifted" == "$expected_drifted" ]]; then
    printf "  PASS: %s (release=%s tap=%s drifted=%s)\n" "$description" "$release" "$tap" "$actual_drifted"
    PASS=$((PASS + 1))
  else
    printf "  FAIL: %s (release=%s tap=%s expected_drifted=%s got=%s)\n" "$description" "$release" "$tap" "$expected_drifted" "$actual_drifted"
    FAIL=$((FAIL + 1))
  fi
}

printf "Testing brew tap drift detection logic\n\n"

# matching versions — no drift
check_drift "v0.23.0" "v0.23.0" "false" "matching versions show no drift"
check_drift "v1.0.0"  "v1.0.0"  "false" "matching major versions show no drift"

# tap behind release — drift
check_drift "v0.23.0" "v0.22.0" "true" "tap behind release is drift"
check_drift "v0.23.1" "v0.23.0" "true" "tap behind by patch is drift"
check_drift "v1.0.0"  "v0.23.0" "true" "tap behind by major is drift"

# tap ahead of release — also drift (manual tap edit)
check_drift "v0.22.0" "v0.23.0" "true" "tap ahead of release is drift"

# different patch versions
check_drift "v0.23.2" "v0.23.1" "true" "different patch versions is drift"

printf "\nResults: %d passed, %d failed\n" "$PASS" "$FAIL"
[[ "$FAIL" -eq 0 ]]
