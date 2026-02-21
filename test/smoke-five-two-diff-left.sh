#!/usr/bin/env bash
# 5 files each side, two different on the left (f2, f4) → two diffs reported
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
LEFT="${SCRIPT_DIR}/five-two-diff-left"
RIGHT="${SCRIPT_DIR}/five-two-diff-right"
out=$("$BIN" --format text "$LEFT" "$RIGHT" 2>/dev/null)
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0, got $exitcode" >&2
  exit 1
fi
if ! echo "$out" | grep -qE "size changed|content differs"; then
  echo "Expected diff reasons in output, got: $out" >&2
  exit 1
fi
for path in f2 f4; do
  if ! echo "$out" | grep -q "${path} "; then
    echo "Expected path $path in output, got: $out" >&2
    exit 1
  fi
done
count=$(echo "$out" | grep -cE "size changed|content differs" || true)
if [[ "$count" -ne 2 ]]; then
  echo "Expected exactly 2 diffs, got $count in: $out" >&2
  exit 1
fi
