#!/usr/bin/env bash
# 5 files on left, 4 on right → one left-only (f5) reported
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
LEFT="${SCRIPT_DIR}/five-one-left-only-left"
RIGHT="${SCRIPT_DIR}/five-one-left-only-right"
out=$("$BIN" --format text "$LEFT" "$RIGHT" 2>/dev/null)
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0, got $exitcode" >&2
  exit 1
fi
if ! echo "$out" | grep -q "left only"; then
  echo "Expected 'left only' in output, got: $out" >&2
  exit 1
fi
if ! echo "$out" | grep -q "f5 "; then
  echo "Expected path f5 (left only) in output, got: $out" >&2
  exit 1
fi
