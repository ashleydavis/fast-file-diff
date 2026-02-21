#!/usr/bin/env bash
# 4 files on left, 5 on right → one right-only (f5) reported
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
LEFT="${SCRIPT_DIR}/five-one-right-only-left"
RIGHT="${SCRIPT_DIR}/five-one-right-only-right"
out=$("$BIN" --format text "$LEFT" "$RIGHT" 2>/dev/null)
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0, got $exitcode" >&2
  exit 1
fi
if ! echo "$out" | grep -q "right only"; then
  echo "Expected 'right only' in output, got: $out" >&2
  exit 1
fi
if ! echo "$out" | grep -q "f5 "; then
  echo "Expected path f5 (right only) in output, got: $out" >&2
  exit 1
fi
