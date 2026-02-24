#!/usr/bin/env bash
# --hash md5: same content → no diff; different content → diff with hash in output
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
TMP="${SCRIPT_DIR}/tmp"
LEFT="${TMP}/md5-identical-left"
RIGHT="${TMP}/md5-identical-right"
mkdir -p "$LEFT" "$RIGHT"
printf '%s' "same" > "$LEFT/f"
printf '%s' "same" > "$RIGHT/f"
out=$("$BIN" --hash md5 --format text "$LEFT" "$RIGHT" 2>/dev/null)
if [[ -n "$out" ]] && echo "$out" | grep -qE "size changed|content differs"; then
  echo "Expected no size/content diff for identical dirs with md5" >&2
  exit 1
fi
mkdir -p "${TMP}/md5-diff-left" "${TMP}/md5-diff-right"
printf '%s' "aa" > "${TMP}/md5-diff-left/f"
sleep 1
printf '%s' "bb" > "${TMP}/md5-diff-right/f"
LEFT="${TMP}/md5-diff-left"
RIGHT="${TMP}/md5-diff-right"
out=$("$BIN" --hash md5 --format text "$LEFT" "$RIGHT" 2>/dev/null)
if ! echo "$out" | grep -q "content differs"; then
  echo "Expected diff with --hash md5 for different content" >&2
  exit 1
fi
# MD5 hex is 32 chars
if ! echo "$out" | grep -qE "[0-9a-f]{32}"; then
  echo "Expected 32-char md5 hash in output" >&2
  exit 1
fi
