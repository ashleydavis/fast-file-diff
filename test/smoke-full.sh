#!/usr/bin/env bash
# --full: always hash every pair; exit 0 and correct diff results
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"

# Identical dirs with --full: no diff, exit 0
LEFT="${SCRIPT_DIR}/identical-left"
RIGHT="${SCRIPT_DIR}/identical-right"
out=$("$BIN" --full --format text "$LEFT" "$RIGHT" 2>/dev/null)
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0 with --full on identical dirs, got $exitcode" >&2
  exit 1
fi
if [[ -n "$out" ]] && echo "$out" | grep -qE "size changed|content differs"; then
  echo "Identical dirs with --full should not show size/content diff, got: $out" >&2
  exit 1
fi

# Same size, different content with --full: content differs reported
LEFT="${SCRIPT_DIR}/mtime-left"
RIGHT="${SCRIPT_DIR}/mtime-right"
mkdir -p "$LEFT" "$RIGHT"
echo "aa" > "$LEFT/f"
sleep 1
echo "bb" > "$RIGHT/f"
out=$("$BIN" --full "$LEFT" "$RIGHT" 2>/dev/null)
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0 with --full on same-size diff content, got $exitcode" >&2
  exit 1
fi
if ! echo "$out" | grep -q "content differs"; then
  echo "Expected 'content differs' with --full for same-size different content, got: $out" >&2
  exit 1
fi
