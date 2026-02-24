#!/usr/bin/env bash
# File only in left dir → reported as left only
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
TMP="${SCRIPT_DIR}/tmp"
LEFT="${TMP}/left-only-left"
RIGHT="${TMP}/left-only-right"
mkdir -p "$LEFT" "$RIGHT"
printf '%s' "only-in-left" > "$LEFT/f"
out=$("$BIN" --format text "$LEFT" "$RIGHT" 2>/dev/null)
if ! echo "$out" | grep -q "left only"; then
  echo "Expected 'left only' in output, got: $out" >&2
  exit 1
fi
