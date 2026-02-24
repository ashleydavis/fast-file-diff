#!/usr/bin/env bash
# --hash sha256: same content → no diff; different content → diff with hash in output
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
TMP="${SCRIPT_DIR}/tmp"
LEFT="${TMP}/sha256-identical-left"
RIGHT="${TMP}/sha256-identical-right"
mkdir -p "$LEFT" "$RIGHT"
printf '%s' "same" > "$LEFT/f"
printf '%s' "same" > "$RIGHT/f"
out=$("$BIN" --hash sha256 --format text "$LEFT" "$RIGHT" 2>/dev/null)
if [[ -n "$out" ]] && echo "$out" | grep -qE "size changed|content differs"; then
  echo "Expected no size/content diff for identical dirs with sha256" >&2
  exit 1
fi
mkdir -p "${TMP}/sha256-diff-left" "${TMP}/sha256-diff-right"
printf '%s' "aa" > "${TMP}/sha256-diff-left/f"
sleep 1
printf '%s' "bb" > "${TMP}/sha256-diff-right/f"
LEFT="${TMP}/sha256-diff-left"
RIGHT="${TMP}/sha256-diff-right"
out=$("$BIN" --hash sha256 --format text "$LEFT" "$RIGHT" 2>/dev/null)
if ! echo "$out" | grep -q "content differs"; then
  echo "Expected diff with --hash sha256 for different content" >&2
  exit 1
fi
# SHA256 hex is 64 chars
if ! echo "$out" | grep -qE "[0-9a-f]{64}"; then
  echo "Expected 64-char sha256 hash in output" >&2
  exit 1
fi
