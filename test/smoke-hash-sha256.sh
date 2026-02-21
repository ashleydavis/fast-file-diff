#!/usr/bin/env bash
# --hash sha256: same content → no diff; different content → diff with hash in output
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
LEFT="${SCRIPT_DIR}/identical-left"
RIGHT="${SCRIPT_DIR}/identical-right"
out=$("$BIN" --hash sha256 --format text "$LEFT" "$RIGHT" 2>/dev/null)
if [[ -n "$out" ]] && echo "$out" | grep -qE "size changed|content differs"; then
  echo "Expected no size/content diff for identical dirs with sha256" >&2
  exit 1
fi
mkdir -p "${SCRIPT_DIR}/hash-sha256-left" "${SCRIPT_DIR}/hash-sha256-right"
echo -n "aa" > "${SCRIPT_DIR}/hash-sha256-left/f"
sleep 1
echo -n "bb" > "${SCRIPT_DIR}/hash-sha256-right/f"
LEFT="${SCRIPT_DIR}/hash-sha256-left"
RIGHT="${SCRIPT_DIR}/hash-sha256-right"
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
