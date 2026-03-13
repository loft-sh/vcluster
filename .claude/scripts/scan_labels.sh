#\!/usr/bin/env bash
# Scan e2e-next labels and output a JSON inventory.
# Usage: scan_labels.sh [repo_root]
# Output: JSON array of {variable, label}
set -euo pipefail

REPO_ROOT="${1:-$(git rev-parse --show-toplevel 2>/dev/null || echo .)}"
LABELS_FILE="$REPO_ROOT/e2e-next/labels/labels.go"

if [[ \! -f "$LABELS_FILE" ]]; then
  echo "Error: $LABELS_FILE not found" >&2
  exit 1
fi

# Extract lines like: VarName = Label("label-string")
# Use grep to find, sed to extract variable and label separately
grep -E '[A-Za-z0-9_]+[[:space:]]*=[[:space:]]*Label\("' "$LABELS_FILE" \
  | while IFS= read -r line; do
    var=$(echo "$line" | sed -E 's/^[[:space:]]*([A-Za-z0-9_]+)[[:space:]]*=.*/\1/')
    label=$(echo "$line" | sed -E 's/.*Label\("([^"]+)"\).*/\1/')
    printf '{"variable":"%s","label":"%s"}\n' "$var" "$label"
  done | jq -s .
