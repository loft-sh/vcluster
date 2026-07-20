#!/usr/bin/env bash
set -eo pipefail

# Tests for .github/scripts/platform-version-bump.sh
# Sources the library and exercises its pure decision functions directly.
# The gh-backed I/O helpers are not covered here; they are thin wrappers.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../scripts/platform-version-bump.sh"

PASS=0
FAIL=0

assert_eq() {
  local actual="$1" expected="$2" description="$3"
  if [[ "$actual" == "$expected" ]]; then
    printf "  PASS: %s (got %q)\n" "$description" "$actual"
    PASS=$((PASS + 1))
  else
    printf "  FAIL: %s (expected %q got %q)\n" "$description" "$expected" "$actual"
    FAIL=$((FAIL + 1))
  fi
}

# assert_gt TAG_A TAG_B EXPECTED(true|false) DESC
assert_gt() {
  local a="$1" b="$2" expected="$3" description="$4" actual
  if version_gt "$a" "$b"; then actual="true"; else actual="false"; fi
  assert_eq "$actual" "$expected" "$description"
}

printf "=== platform_minor ===\n\n"
assert_eq "$(platform_minor v4.10.3)"      "4.10" "strips v and patch"
assert_eq "$(platform_minor v4.11.0-rc.7)" "4.11" "drops pre-release suffix"
assert_eq "$(platform_minor v4.7.0)"       "4.7"  "single-digit minor"
assert_eq "$(platform_minor 4.10.5)"       "4.10" "no leading v"
assert_eq "$(platform_minor v4.10.0-rc.1)" "4.10" "rc tag on x.0"

printf "\n=== version_gt (semver, pre-release aware) ===\n\n"
assert_gt v4.10.5 v4.10.3 "true"  "newer patch"
assert_gt v4.10.3 v4.10.5 "false" "older patch"
assert_gt v4.10.3 v4.10.3 "false" "equal is not greater"
assert_gt v4.8.0  v4.7.9  "true"  "newer minor beats higher patch"
assert_gt v5.0.0  v4.99.99 "true" "newer major"
# Pre-release ordering: a pre-release is OLDER than its final release.
assert_gt v4.10.0-rc.5 v4.10.0    "false" "pre-release is older than final"
assert_gt v4.10.0      v4.10.0-rc.5 "true" "final is newer than its pre-release"
assert_gt v4.11.0-rc.7 v4.11.0-rc.6 "true"  "newer rc"
assert_gt v4.11.0-rc.6 v4.11.0-rc.7 "false" "older rc"
assert_gt v4.10.0-rc.10 v4.10.0-rc.9 "true" "rc.10 newer than rc.9 (numeric, not lexical)"

printf "\n=== select_target_branch (derive coupled release branch) ===\n\n"
# Mirrors the real coupling: each branch's committed MinimumVersionTag.
MAP=$'v0.32=v4.7.0\nv0.33=v4.8.0\nv0.34=v4.9.2\nv0.35=v4.10.3\nv0.36=v4.11.0'

assert_eq "$(select_target_branch v4.10.5 <<<"$MAP")"      "v0.35" "4.10.x routes to v0.35"
assert_eq "$(select_target_branch v4.11.0-rc.8 <<<"$MAP")" "v0.36" "4.11 rc routes to v0.36"
assert_eq "$(select_target_branch v4.7.3 <<<"$MAP")"       "v0.32" "4.7.x routes to v0.32"

# No branch carries platform 4.99 -> non-zero exit, nothing on stdout.
if out=$(select_target_branch v4.99.0 <<<"$MAP" 2>/dev/null); then
  assert_eq "matched:$out" "should-have-failed" "unknown minor fails loudly"
else
  assert_eq "failed" "failed" "unknown minor exits non-zero"
fi

# Two branches sharing a minor -> ambiguous, non-zero exit.
DUP_MAP=$'v0.35=v4.10.3\nv0.35-old=v4.10.1'
if out=$(select_target_branch v4.10.5 <<<"$DUP_MAP" 2>/dev/null); then
  assert_eq "matched:$out" "should-have-failed" "ambiguous minor fails loudly"
else
  assert_eq "failed" "failed" "ambiguous minor exits non-zero"
fi

printf "\n=== select_stale_prs (older tags on same branch) ===\n\n"
# Open bump PRs on one branch; incoming is v4.10.5.
PRS=$'4065 v4.10.4\n4020 v4.10.3\n4100 v4.10.5\n4200 v4.10.6\n4090 v4.10.0-rc.5'
STALE=$(printf '%s\n' "$PRS" | select_stale_prs v4.10.5 | sort -n | tr '\n' ' ')
STALE="${STALE%% }"
# 4065(4.10.4), 4020(4.10.3), 4090(rc.5) are older; 4100(equal) and 4200(newer) stay open.
assert_eq "$STALE" "4020 4065 4090" "closes only strictly-older PRs, keeps equal and newer"

STALE_NONE=$(printf '4100 v4.10.5\n' | select_stale_prs v4.10.5 | tr '\n' ' ')
assert_eq "${STALE_NONE%% }" "" "the incoming tag's own PR is never stale"

printf "\nResults: %d passed, %d failed\n" "$PASS" "$FAIL"
[[ "$FAIL" -eq 0 ]]
