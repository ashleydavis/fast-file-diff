#!/usr/bin/env bash
# --phase build-pairs: exit 0 and stderr contains "build-pairs: <duration>"
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
LEFT="${SCRIPT_DIR}/empty-left"
RIGHT="${SCRIPT_DIR}/empty-right"
mkdir -p "$LEFT" "$RIGHT"
stderr=$(mktemp)
trap 'rm -f "$stderr"' EXIT
"$BIN" --phase build-pairs "$LEFT" "$RIGHT" >/dev/null 2>"$stderr"
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0, got $exitcode" >&2
  cat "$stderr" >&2
  exit 1
fi
if ! grep -q '^build-pairs: ' "$stderr"; then
  echo "Expected stderr to contain 'build-pairs: <duration>'"
  cat "$stderr" >&2
  exit 1
fi
