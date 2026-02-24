#!/usr/bin/env bash
# --phase walk-left: exit 0 and stderr contains "walk-left: <duration>"
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
LEFT="${SCRIPT_DIR}/empty-left"
RIGHT="${SCRIPT_DIR}/empty-right"
mkdir -p "$LEFT" "$RIGHT"
stderr=$(mktemp)
trap 'rm -f "$stderr"' EXIT
"$BIN" --phase walk-left "$LEFT" "$RIGHT" >/dev/null 2>"$stderr"
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0, got $exitcode" >&2
  cat "$stderr" >&2
  exit 1
fi
if ! grep -q '^walk-left: ' "$stderr"; then
  echo "Expected stderr to contain 'walk-left: <duration>'"
  cat "$stderr" >&2
  exit 1
fi
