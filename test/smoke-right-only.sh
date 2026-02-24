#!/usr/bin/env bash
# File only in right dir → reported (right only)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
TMP="${SCRIPT_DIR}/tmp"
LEFT="${TMP}/right-only-left"
RIGHT="${TMP}/right-only-right"
mkdir -p "$LEFT" "$RIGHT"
printf '%s' "only-in-right" > "$RIGHT/f"
out=$("$BIN" --format text "$LEFT" "$RIGHT" 2>/dev/null)
if ! echo "$out" | grep -q "right only"; then
  echo "Expected 'right only' in output, got: $out" >&2
  exit 1
fi
