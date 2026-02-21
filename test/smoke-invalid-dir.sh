#!/usr/bin/env bash
# Invalid or missing path → exit 1 or 2 and clear error on stderr
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
VALID="${SCRIPT_DIR}/empty-left"
INVALID="${SCRIPT_DIR}/nonexistent-dir-$$"
set +e
"$BIN" "$INVALID" "$VALID" 2>/tmp/ffd-err.$$
exitcode=$?
set -e
if [[ $exitcode -ne 2 ]]; then
  echo "Expected exit 2 for invalid left path, got $exitcode" >&2
  cat /tmp/ffd-err.$$ >&2
  rm -f /tmp/ffd-err.$$
  exit 1
fi
if ! grep -q "directory\|left" /tmp/ffd-err.$$; then
  echo "Expected clear error mentioning directory or left" >&2
  cat /tmp/ffd-err.$$ >&2
  rm -f /tmp/ffd-err.$$
  exit 1
fi
set +e
"$BIN" "$VALID" "$INVALID" 2>/tmp/ffd-err.$$
exitcode=$?
set -e
rm -f /tmp/ffd-err.$$
if [[ $exitcode -ne 2 ]]; then
  echo "Expected exit 2 for invalid right path, got $exitcode" >&2
  exit 1
fi
