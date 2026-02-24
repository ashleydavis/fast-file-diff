#!/usr/bin/env bash
# --format yaml: compare output to known-good expected (normalize timestamps for portability)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
out=$("$BIN" --format yaml "$SCRIPT_DIR/diff-left" "$SCRIPT_DIR/diff-right" 2>/dev/null)
# Normalize timestamps so comparison is portable
actual=$(printf '%s' "$out" | tr -d '\r' | sed -E 's/"[0-9]{4}-[0-9]{2}-[0-9]{2}T[^"]*"/"TIMESTAMP"/g')
expected=$(tr -d '\r' < "$SCRIPT_DIR/format-yaml-expected.yaml")
if [[ "$actual" != "$expected" ]]; then
  echo "YAML output did not match expected (format-yaml-expected.yaml)" >&2
  echo "--- actual (normalized) ---" >&2
  echo "$actual" >&2
  echo "--- expected ---" >&2
  echo "$expected" >&2
  exit 1
fi
