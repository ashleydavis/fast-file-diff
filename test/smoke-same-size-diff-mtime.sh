#!/usr/bin/env bash
# Same size, different content (and mtime) → diff reported (content differs)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
LEFT="${SCRIPT_DIR}/mtime-left"
RIGHT="${SCRIPT_DIR}/mtime-right"
mkdir -p "$LEFT" "$RIGHT"
echo "aa" > "$LEFT/f"
sleep 1
echo "bb" > "$RIGHT/f"
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
