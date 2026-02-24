#!/usr/bin/env bash
# Same size, different content (and mtime) → diff reported (content differs)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
TMP="${SCRIPT_DIR}/tmp"
LEFT="${TMP}/same-size-diff-mtime-left"
RIGHT="${TMP}/same-size-diff-mtime-right"
mkdir -p "$LEFT" "$RIGHT"
printf '%s' "aa" > "$LEFT/f"
sleep 1
printf '%s' "bb" > "$RIGHT/f"
out=$("$BIN" "$LEFT" "$RIGHT" 2>/dev/null)
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0, got $exitcode" >&2
  exit 1
fi
if ! echo "$out" | grep -q "content differs\|size changed"; then
  echo "Expected diff reason for same-size different-content, got: $out" >&2
  exit 1
fi
