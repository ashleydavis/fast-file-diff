#!/usr/bin/env bash
# --format text: diff output is tree-shaped and includes path/size/reason
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
TMP="${SCRIPT_DIR}/tmp"
LEFT="${TMP}/format-text-left"
RIGHT="${TMP}/format-text-right"
mkdir -p "$LEFT" "$RIGHT"
printf '%s' "12345" > "$LEFT/f"
printf '%s' "123456" > "$RIGHT/f"
out=$("$BIN" --format text "$LEFT" "$RIGHT" 2>/dev/null)
if ! echo "$out" | grep -q "size="; then
  echo "Expected size= in text format output" >&2
  exit 1
fi
if ! echo "$out" | grep -q "f "; then
  echo "Expected path in output" >&2
  exit 1
fi
if ! echo "$out" | grep -q "size changed\|content differs"; then
  echo "Expected reason in output" >&2
  exit 1
fi
