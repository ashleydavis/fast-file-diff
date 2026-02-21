#!/usr/bin/env bash
# 5 files each side, one file different → exactly one diff reported
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
LEFT="${SCRIPT_DIR}/five-one-diff-left"
RIGHT="${SCRIPT_DIR}/five-one-diff-right"
out=$("$BIN" --format text "$LEFT" "$RIGHT" 2>/dev/null)
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0, got $exitcode" >&2
  exit 1
fi
if ! echo "$out" | grep -qE "size changed|content differs"; then
  echo "Expected one diff reason in output, got: $out" >&2
  exit 1
fi
if ! echo "$out" | grep -q "f3 "; then
  echo "Expected path f3 in output, got: $out" >&2
  exit 1
fi
# Should be exactly one differing file (f3)
count=$(echo "$out" | grep -cE "size changed|content differs" || true)
if [[ "$count" -ne 1 ]]; then
  echo "Expected exactly 1 diff, got $count in: $out" >&2
  exit 1
fi
