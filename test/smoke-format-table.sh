#!/usr/bin/env bash
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
out=$("$BIN" --format table "$SCRIPT_DIR/diff-left" "$SCRIPT_DIR/diff-right" 2>/dev/null)
if ! echo "$out" | grep -q "path.*size"; then
  echo "Expected header with path/size" >&2
  exit 1
fi
if ! echo "$out" | grep -q "f"; then
  echo "Expected path in output" >&2
  exit 1
fi
