#!/usr/bin/env bash
# 5 files each side, all same → no diff, exit 0
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
LEFT="${SCRIPT_DIR}/five-same-left"
RIGHT="${SCRIPT_DIR}/five-same-right"
out=$("$BIN" --format text "$LEFT" "$RIGHT" 2>/dev/null)
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0, got $exitcode" >&2
  exit 1
fi
if [[ -n "$out" ]] && echo "$out" | grep -qE "size changed|content differs"; then
  echo "Five same files should not show size/content diff, got: $out" >&2
  exit 1
fi
