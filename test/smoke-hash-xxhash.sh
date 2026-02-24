#!/usr/bin/env bash
# --hash xxhash: same content → no diff; different content → diff with hash in output
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
TMP="${SCRIPT_DIR}/tmp"
# Same content: use test-specific identical dirs
LEFT="${TMP}/xxhash-identical-left"
RIGHT="${TMP}/xxhash-identical-right"
mkdir -p "$LEFT" "$RIGHT"
printf '%s' "same" > "$LEFT/f"
printf '%s' "same" > "$RIGHT/f"
out=$("$BIN" --hash xxhash --format text "$LEFT" "$RIGHT" 2>/dev/null)
# Same content: no size/content diff
if [[ -n "$out" ]] && echo "$out" | grep -qE "size changed|content differs"; then
  echo "Expected no size/content diff for identical dirs, got: $out" >&2
  exit 1
fi
# Different content, same size: xxhash-diff-left has "aa", xxhash-diff-right has "bb"
mkdir -p "${TMP}/xxhash-diff-left" "${TMP}/xxhash-diff-right"
printf '%s' "aa" > "${TMP}/xxhash-diff-left/f"
sleep 1
printf '%s' "bb" > "${TMP}/xxhash-diff-right/f"
LEFT="${TMP}/xxhash-diff-left"
RIGHT="${TMP}/xxhash-diff-right"
out=$("$BIN" --hash xxhash --format text "$LEFT" "$RIGHT" 2>/dev/null)
if ! echo "$out" | grep -q "content differs"; then
  echo "Expected diff with --hash xxhash for different content" >&2
  exit 1
fi
if ! echo "$out" | grep -qE "[0-9a-f]{8,}"; then
  echo "Expected hash (hex) in output, got: $out" >&2
  exit 1
fi
