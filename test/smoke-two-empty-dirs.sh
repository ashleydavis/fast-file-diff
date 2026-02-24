#!/usr/bin/env bash
# Two existing empty dirs → exit 0 (no diff output)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
TMP="${SCRIPT_DIR}/tmp"
LEFT="${TMP}/two-empty-dirs-left"
RIGHT="${TMP}/two-empty-dirs-right"
mkdir -p "$LEFT" "$RIGHT"
"$BIN" "$LEFT" "$RIGHT" >/tmp/ffd-out.$$ 2>/tmp/ffd-err.$$
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0, got $exitcode" >&2
  cat /tmp/ffd-err.$$ >&2
  rm -f /tmp/ffd-out.$$ /tmp/ffd-err.$$
  exit 1
fi
rm -f /tmp/ffd-out.$$ /tmp/ffd-err.$$
