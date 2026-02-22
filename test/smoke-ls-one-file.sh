#!/usr/bin/env bash
# ffd ls on directory with one file → one line
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
DIR="${SCRIPT_DIR}/ls-one-file"
if [[ ! -d "$DIR" ]]; then
  echo "Missing test dir $DIR" >&2
  exit 1
fi
out=$("$BIN" ls "$DIR" 2>/dev/null)
if [[ "$out" != "a" ]]; then
  echo "Expected 'a', got: $out" >&2
  exit 1
fi
