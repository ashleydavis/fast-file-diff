#!/usr/bin/env bash
# Two dirs with one file different (same path, different content) → diff reported on stdout
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
LEFT="${SCRIPT_DIR}/diff-left"
RIGHT="${SCRIPT_DIR}/diff-right"
out=$("$BIN" "$LEFT" "$RIGHT" 2>/dev/null)
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0, got $exitcode" >&2
  exit 1
fi
if ! echo "$out" | grep -q "size changed\|content differs"; then
  echo "Expected reason in output, got: $out" >&2
  exit 1
fi
if ! echo "$out" | grep -q "f "; then
  echo "Expected path 'f' in output, got: $out" >&2
  exit 1
fi
