#!/usr/bin/env bash
# --quiet: piped stdout has no progress or "check error log" on stderr
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
TMP="${SCRIPT_DIR}/tmp"
LEFT="${TMP}/quiet-left"
RIGHT="${TMP}/quiet-right"
mkdir -p "$LEFT" "$RIGHT"
printf '%s' "12345" > "$LEFT/f"
printf '%s' "123456" > "$RIGHT/f"
err=$("$BIN" --quiet "$LEFT" "$RIGHT" 2>&1 >/dev/null)
if echo "$err" | grep -q "processed\|pending\|Main log\|Error log\|check the error log"; then
  echo "With --quiet, stderr should not contain progress or log messages, got: $err" >&2
  exit 1
fi
# Piped stdout should be just the diff
out=$("$BIN" --quiet "$LEFT" "$RIGHT" 2>/dev/null)
if ! echo "$out" | grep -q "size changed\|content differs"; then
  echo "Expected diff output on stdout" >&2
  exit 1
fi
