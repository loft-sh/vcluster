#!/usr/bin/env bash
set -eo pipefail

# Decision logic for the "Update MinimumVersionTag from Platform" workflow.
#
# Source this file to use its functions, or execute it directly for the
# workflow's "decide" step (see the entrypoint at the bottom).
#
# A platform release is coupled 1:1 with a vCluster release branch: the branch
# that carries that platform minor in pkg/platform/version.go's
# MinimumVersionTag. For example platform v4.10.x is carried by the v0.35
# branch (its MinimumVersionTag is v4.10.z). This script routes a platform
# version bump to the correct release branch instead of always targeting main,
# and identifies stale bump PRs (older platform tags on the same branch) to
# close.

# Repository to query, owner/name form. Overridable for tests.
REPO="${REPO:-loft-sh/vcluster}"

# Path to the file that holds the MinimumVersionTag on each branch.
VERSION_FILE="${VERSION_FILE:-pkg/platform/version.go}"

# Prefix for the head branches this workflow creates. Kept in one place so the
# creator and the stale-PR matcher can never drift apart.
BUMP_BRANCH_PREFIX="update-platform-version-tag"

# Extract the "major.minor" of a semver tag, dropping the leading "v" and any
# patch / pre-release suffix. platform_minor "v4.10.3-rc.1" -> "4.10".
platform_minor() {
  local v="${1#v}"
  local major="${v%%.*}"
  local rest="${v#*.}"
  printf '%s.%s\n' "$major" "${rest%%.*}"
}

# version_gt A B -> exit 0 if A is strictly newer than B in semver order.
#
# GNU `sort -V` ranks a pre-release (v4.10.0-rc.5) ABOVE the final release
# (v4.10.0), which is backwards for semver. `~` sorts before everything in
# `sort -V` (including end-of-string), so translating '-' to '~' restores
# semver precedence (pre-release < final). tr is used rather than bash
# parameter substitution because "${x//-/~}" would embed a literal backslash.
version_gt() {
  local a b highest
  a=$(printf '%s' "$1" | tr '-' '~')
  b=$(printf '%s' "$2" | tr '-' '~')
  [[ "$a" == "$b" ]] && return 1
  highest=$(printf '%s\n%s\n' "$a" "$b" | sort -V | tail -n1)
  [[ "$highest" == "$a" ]]
}

# select_target_branch INCOMING_TAG < branch-map
#
# Reads a "branch=tag" map (one pair per line) on stdin and prints the single
# branch whose tag shares INCOMING_TAG's major.minor. Exits non-zero when zero
# or more than one branch matches, so an ambiguous coupling fails loudly rather
# than routing a bump to the wrong branch.
select_target_branch() {
  local incoming_minor branch tag matches count
  incoming_minor=$(platform_minor "$1")

  matches=""
  while IFS='=' read -r branch tag; do
    [[ -z "$branch" || -z "$tag" ]] && continue
    if [[ "$(platform_minor "$tag")" == "$incoming_minor" ]]; then
      matches+="${branch}"$'\n'
    fi
  done

  # Collapse to a clean, de-duplicated list.
  matches=$(printf '%s' "$matches" | sed '/^$/d' | sort -u)
  if [[ -z "$matches" ]]; then
    count=0
  else
    count=$(printf '%s\n' "$matches" | wc -l | tr -d ' ')
  fi

  if [[ "$count" -eq 1 ]]; then
    printf '%s\n' "$matches"
    return 0
  elif [[ "$count" -eq 0 ]]; then
    printf '::error::no vCluster release branch carries platform %s (checked MinimumVersionTag on each %s branch)\n' \
      "$incoming_minor" "$REPO" >&2
    return 3
  else
    printf '::error::platform %s maps to multiple branches: %s\n' \
      "$incoming_minor" "$(printf '%s' "$matches" | tr '\n' ' ')" >&2
    return 4
  fi
}

# select_stale_prs INCOMING_TAG < pr-list
#
# Reads "number tag" lines on stdin (the open bump PRs already filtered to the
# target branch) and prints the numbers of PRs whose tag is strictly older than
# INCOMING_TAG. The PR for INCOMING_TAG itself (equal tag) and any newer PR are
# left untouched.
select_stale_prs() {
  local incoming="$1" num tag
  while read -r num tag; do
    [[ -z "$num" || -z "$tag" ]] && continue
    if version_gt "$incoming" "$tag"; then
      printf '%s\n' "$num"
    fi
  done
}

# --- I/O helpers (thin gh wrappers; exercised in the workflow, not unit tests) ---

# Print the MinimumVersionTag committed on a branch, or nothing if the file or
# tag is absent there.
branch_min_version_tag() {
  local branch="$1"
  gh api -H "Accept: application/vnd.github.raw" \
    "repos/${REPO}/contents/${VERSION_FILE}?ref=${branch}" 2>/dev/null |
    grep -oP 'MinimumVersionTag\s*=\s*"\Kv[^"]+' || true
}

# Print a "branch=tag" map for every clean release branch (v0.<n>) that carries
# a MinimumVersionTag. main and hotfix branches are intentionally excluded: main
# must stop receiving these bumps, and hotfix branches (v0.x.y-hotfix) are not
# release lines.
build_branch_map() {
  local branch tag
  while read -r branch; do
    [[ -z "$branch" ]] && continue
    tag=$(branch_min_version_tag "$branch")
    [[ -n "$tag" ]] && printf '%s=%s\n' "$branch" "$tag"
  done < <(gh api --paginate "repos/${REPO}/branches" --jq '.[].name' | grep -E '^v0\.[0-9]+$')
}

# Print "number tag" lines for open bump PRs targeting BASE, where tag is the
# last path segment of the head branch (update-platform-version-tag/.../<tag>).
open_bump_prs_for_base() {
  local base="$1"
  gh pr list --repo "${REPO}" --state open --limit 200 \
    --json number,headRefName,baseRefName \
    --jq ".[]
      | select(.baseRefName == \"${base}\")
      | select(.headRefName | startswith(\"${BUMP_BRANCH_PREFIX}/\"))
      | \"\(.number) \(.headRefName | split(\"/\") | last)\""
}

# Emit key=value to the step output (or stdout when run outside Actions).
emit() {
  printf '%s=%s\n' "$1" "$2" >> "${GITHUB_OUTPUT:-/dev/stdout}"
}

# Entrypoint for the workflow's "decide" step. Runs only when executed
# directly, not when sourced by the test.
#
# Required env:
#   INCOMING_TAG     platform tag to set (e.g. v4.10.5 or v4.11.0-rc.7).
# Optional env:
#   EXPLICIT_BRANCH  target branch chosen by the caller; skips derivation.
#   GITHUB_OUTPUT    Actions step output file.
#   GH_TOKEN         token with contents:read + pull-requests:read on REPO.
#
# Outputs: tag, target, skip (true|false), stale_prs (space-separated numbers).
main() {
  : "${INCOMING_TAG:?INCOMING_TAG is required}"
  local target current explicit="${EXPLICIT_BRANCH:-}"

  if [[ -n "$explicit" ]]; then
    target="$explicit"
    current=$(branch_min_version_tag "$target")
    printf '::notice::caller pinned target branch %s (current MinimumVersionTag: %s)\n' \
      "$target" "${current:-none}"
  else
    local map
    map=$(build_branch_map)
    printf '::notice::branch coupling derived from %s:\n%s\n' "$VERSION_FILE" "$map"
    target=$(select_target_branch "$INCOMING_TAG" <<<"$map")
    current=$(printf '%s\n' "$map" | grep "^${target}=" | head -n1 | cut -d= -f2-)
    printf '::notice::platform %s -> branch %s (current MinimumVersionTag: %s)\n' \
      "$(platform_minor "$INCOMING_TAG")" "$target" "${current:-none}"
  fi

  emit tag "$INCOMING_TAG"
  emit target "$target"

  if [[ -n "$current" ]] && ! version_gt "$INCOMING_TAG" "$current"; then
    printf '::notice::%s is not newer than %s on %s; skipping.\n' "$INCOMING_TAG" "$current" "$target"
    emit skip "true"
    emit stale_prs ""
    return 0
  fi

  emit skip "false"

  local stale
  stale=$(open_bump_prs_for_base "$target" | select_stale_prs "$INCOMING_TAG" | tr '\n' ' ')
  stale="${stale%% }"
  printf '::notice::stale bump PRs to close on %s: %s\n' "$target" "${stale:-none}"
  emit stale_prs "$stale"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main
fi
