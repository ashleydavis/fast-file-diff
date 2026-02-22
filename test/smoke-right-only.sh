#!/usr/bin/env bash
# File only in right dir → reported (right only)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
LEFT="${SCRIPT_DIR}/empty-left"
RIGHT="${SCRIPT_DIR}/right-only"
mkdir -p "$LEFT"
out=$("$BIN" --format text "$LEFT" "$RIGHT" 2>/dev/null)
if ! echo "$out" | grep -q "right only"; then
  echo "Expected 'right only' in output, got: $out" >&2
  exit 1
fi
