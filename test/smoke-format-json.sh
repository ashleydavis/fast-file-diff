#!/usr/bin/env bash
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
TMP="${SCRIPT_DIR}/tmp"
LEFT="${TMP}/format-json-left"
RIGHT="${TMP}/format-json-right"
mkdir -p "$LEFT" "$RIGHT"
printf '%s' "12345" > "$LEFT/f"
printf '%s' "123456" > "$RIGHT/f"
out=$("$BIN" --format json "$LEFT" "$RIGHT" 2>/dev/null)
if ! echo "$out" | python3 -c "import sys,json; json.load(sys.stdin)" 2>/dev/null; then
  echo "Expected valid JSON" >&2
  exit 1
fi
