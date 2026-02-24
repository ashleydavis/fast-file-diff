#!/usr/bin/env bash
# --phase hash-right: exit 0 and stderr contains "hash-right: <duration>"
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
LEFT="${SCRIPT_DIR}/empty-left"
RIGHT="${SCRIPT_DIR}/empty-right"
mkdir -p "$LEFT" "$RIGHT"
stderr=$(mktemp)
trap 'rm -f "$stderr"' EXIT
"$BIN" --phase hash-right "$LEFT" "$RIGHT" >/dev/null 2>"$stderr"
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0, got $exitcode" >&2
  cat "$stderr" >&2
  exit 1
fi
if ! grep -q '^hash-right: ' "$stderr"; then
  echo "Expected stderr to contain 'hash-right: <duration>'"
  cat "$stderr" >&2
  exit 1
fi
