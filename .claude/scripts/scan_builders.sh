#\!/usr/bin/env bash
# Scan e2e-next cluster definitions and output a JSON inventory.
# Usage: scan_builders.sh [repo_root]
# Output: JSON array of {name, path, type (host|vcluster), dependencies[]}
set -uo pipefail

REPO_ROOT="${1:-$(git rev-parse --show-toplevel 2>/dev/null || echo .)}"
CLUSTERS_DIR="$REPO_ROOT/e2e-next/clusters"

if [[ \! -d "$CLUSTERS_DIR" ]]; then
  echo "Error: $CLUSTERS_DIR not found" >&2
  exit 1
fi

first=true
echo "["
for go_file in "$CLUSTERS_DIR"/*.go; do
  [[ -e "$go_file" ]] || continue

  # Extract exported variable definitions that call vcluster.Define() or cluster.Define()
  # Pattern: VarName = vcluster.Define(...) or VarName = cluster.Define(...)
  while IFS= read -r line; do
    var_name=$(echo "$line" | sed -E 's/^[[:space:]]*([A-Z][A-Za-z0-9_]*)[[:space:]]*=.*/\1/')

    # Determine type based on which Define() is called
    if echo "$line" | grep -q 'vcluster\.Define'; then
      cluster_type="vcluster"
    else
      cluster_type="host"
    fi

    # Read the full definition block from the file to extract WithName and WithDependencies
    file_content=$(cat "$go_file")

    # Extract cluster name from WithName("...") associated with this variable
    cluster_name=$(echo "$file_content" | grep -A 20 "^[[:space:]]*${var_name}[[:space:]]*=" | grep -oE 'WithName\("([^"]+)"\)' | head -1 | sed -E 's/WithName\("([^"]+)"\)/\1/')
    if [[ -z "$cluster_name" ]]; then
      cluster_name="$var_name"
    fi

    # Extract dependencies from WithDependencies(...) calls
    deps=$(echo "$file_content" | grep -A 20 "^[[:space:]]*${var_name}[[:space:]]*=" | grep -oE 'WithDependencies\(([^)]+)\)' | head -1 | sed -E 's/WithDependencies\(([^)]+)\)/\1/')
    if [[ -n "$deps" ]]; then
      # Convert comma-separated Go identifiers to JSON array
      deps_json=$(echo "$deps" | tr ',' '\n' | sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//' | jq -R . | jq -s .)
    else
      deps_json='[]'
    fi

    rel_path="e2e-next/clusters/$(basename "$go_file")"

    if [[ "$first" == true ]]; then
      first=false
    else
      echo ","
    fi

    jq -n \
      --arg name "$cluster_name" \
      --arg path "$rel_path" \
      --arg type "$cluster_type" \
      --argjson dependencies "$deps_json" \
      '{name: $name, path: $path, type: $type, dependencies: $dependencies}'

  done < <(grep -E '^[[:space:]]*[A-Z][A-Za-z0-9_]*[[:space:]]*=[[:space:]]*(vcluster|cluster)\.Define\(' "$go_file")
done
echo "]"
