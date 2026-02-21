#!/usr/bin/env bash
# --hash xxhash: same content → no diff; different content → diff with hash in output
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
# Same content: identical-left/right already exist
LEFT="${SCRIPT_DIR}/identical-left"
RIGHT="${SCRIPT_DIR}/identical-right"
out=$("$BIN" --hash xxhash --format text "$LEFT" "$RIGHT" 2>/dev/null)
# Same content: no size/content diff
if [[ -n "$out" ]] && echo "$out" | grep -qE "size changed|content differs"; then
  echo "Expected no size/content diff for identical dirs, got: $out" >&2
  exit 1
fi
# Different content, same size: hash-left has "aa", hash-right has "bb"
mkdir -p "${SCRIPT_DIR}/hash-left" "${SCRIPT_DIR}/hash-right"
echo -n "aa" > "${SCRIPT_DIR}/hash-left/f"
sleep 1
echo -n "bb" > "${SCRIPT_DIR}/hash-right/f"
LEFT="${SCRIPT_DIR}/hash-left"
RIGHT="${SCRIPT_DIR}/hash-right"
out=$("$BIN" --hash xxhash --format text "$LEFT" "$RIGHT" 2>/dev/null)
if ! echo "$out" | grep -q "content differs"; then
  echo "Expected diff with --hash xxhash for different content" >&2
  exit 1
fi
if ! echo "$out" | grep -qE "[0-9a-f]{8,}"; then
  echo "Expected hash (hex) in output, got: $out" >&2
  exit 1
fi
