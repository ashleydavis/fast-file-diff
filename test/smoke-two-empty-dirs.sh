#!/usr/bin/env bash
# Two existing empty dirs → exit 0 (no diff output)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
LEFT="${SCRIPT_DIR}/empty-left"
RIGHT="${SCRIPT_DIR}/empty-right"
if [[ ! -d "$LEFT" ]] || [[ ! -d "$RIGHT" ]]; then
  echo "Missing test dirs $LEFT or $RIGHT" >&2
  exit 1
fi
"$BIN" "$LEFT" "$RIGHT" >/tmp/ffd-out.$$ 2>/tmp/ffd-err.$$
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0, got $exitcode" >&2
  cat /tmp/ffd-err.$$ >&2
  rm -f /tmp/ffd-out.$$ /tmp/ffd-err.$$
  exit 1
fi
rm -f /tmp/ffd-out.$$ /tmp/ffd-err.$$
