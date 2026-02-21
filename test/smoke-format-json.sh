#!/usr/bin/env bash
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
out=$("$BIN" --format json "$SCRIPT_DIR/diff-left" "$SCRIPT_DIR/diff-right" 2>/dev/null)
if ! echo "$out" | python3 -c "import sys,json; json.load(sys.stdin)" 2>/dev/null; then
  echo "Expected valid JSON" >&2
  exit 1
fi
