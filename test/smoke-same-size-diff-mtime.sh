#!/usr/bin/env bash
# Same size, different mtime → diff reported (mtime differs)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
LEFT="${SCRIPT_DIR}/mtime-left"
RIGHT="${SCRIPT_DIR}/mtime-right"
mkdir -p "$LEFT" "$RIGHT"
echo "same" > "$LEFT/f"
echo "same" > "$RIGHT/f"
touch -d "2020-01-01" "$RIGHT/f" 2>/dev/null || true
out=$("$BIN" "$LEFT" "$RIGHT" 2>/dev/null)
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0, got $exitcode" >&2
  exit 1
fi
if ! echo "$out" | grep -q "diff:"; then
  echo "Expected 'diff:' for same-size different-mtime, got: $out" >&2
  exit 1
fi
