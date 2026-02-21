#!/usr/bin/env bash
# --quiet: piped stdout has no progress or "check error log" on stderr
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
LEFT="${SCRIPT_DIR}/diff-left"
RIGHT="${SCRIPT_DIR}/diff-right"
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
