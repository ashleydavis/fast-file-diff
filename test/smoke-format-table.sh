#!/usr/bin/env bash
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
TMP="${SCRIPT_DIR}/tmp"
LEFT="${TMP}/format-table-left"
RIGHT="${TMP}/format-table-right"
mkdir -p "$LEFT" "$RIGHT"
printf '%s' "12345" > "$LEFT/f"
printf '%s' "123456" > "$RIGHT/f"
out=$("$BIN" --format table "$LEFT" "$RIGHT" 2>/dev/null)
if ! echo "$out" | grep -q "path.*size"; then
  echo "Expected header with path/size" >&2
  exit 1
fi
if ! echo "$out" | grep -q "f"; then
  echo "Expected path in output" >&2
  exit 1
fi
