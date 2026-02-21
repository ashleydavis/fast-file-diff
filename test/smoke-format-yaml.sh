#!/usr/bin/env bash
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
out=$("$BIN" --format yaml "$SCRIPT_DIR/diff-left" "$SCRIPT_DIR/diff-right" 2>/dev/null)
if ! echo "$out" | python3 -c "import sys,yaml; yaml.safe_load(sys.stdin)" 2>/dev/null; then
  echo "Expected valid YAML" >&2
  exit 1
fi
