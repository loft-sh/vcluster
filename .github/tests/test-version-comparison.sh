#!/usr/bin/env bash
set -eo pipefail

# Tests the version comparison logic used in update-platform-minimum-version.yaml.
# Validates that older/equal tags are skipped and only newer tags proceed.

PASS=0
FAIL=0

check_version() {
  local current="$1" incoming="$2" expected_skip="$3" description="$4"

  highest=$(printf '%s\n%s\n' "$current" "$incoming" | sort -V | tail -n1)
  if [[ "$highest" == "$current" ]]; then
    actual_skip="true"
  else
    actual_skip="false"
  fi

  if [[ "$actual_skip" == "$expected_skip" ]]; then
    printf "  PASS: %s (current=%s incoming=%s skip=%s)\n" "$description" "$current" "$incoming" "$actual_skip"
    PASS=$((PASS + 1))
  else
    printf "  FAIL: %s (current=%s incoming=%s expected_skip=%s got=%s)\n" "$description" "$current" "$incoming" "$expected_skip" "$actual_skip"
    FAIL=$((FAIL + 1))
  fi
}

printf "Testing version comparison logic\n\n"

# older versions should be skipped
check_version "v4.7.0" "v4.4.3" "true"  "older patch release is skipped"
check_version "v4.7.0" "v3.0.0" "true"  "older major release is skipped"
check_version "v4.7.0" "v4.6.9" "true"  "older minor release is skipped"

# equal version should be skipped
check_version "v4.7.0" "v4.7.0" "true"  "equal version is skipped"

# newer versions should proceed
check_version "v4.7.0" "v4.8.0" "false" "newer minor release proceeds"
check_version "v4.7.0" "v4.7.1" "false" "newer patch release proceeds"
check_version "v4.7.0" "v5.0.0" "false" "newer major release proceeds"

printf "\nResults: %d passed, %d failed\n" "$PASS" "$FAIL"
[[ "$FAIL" -eq 0 ]]
