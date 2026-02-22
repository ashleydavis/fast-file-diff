#!/usr/bin/env bash
# ffd ls on directory that contains only an empty subdir → no files listed
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
DIR="${SCRIPT_DIR}/ls-empty-subdir"
if [[ ! -d "$DIR" ]]; then
  echo "Missing test dir $DIR" >&2
  exit 1
fi
out=$("$BIN" ls "$DIR" 2>/dev/null)
if [[ -n "$out" ]]; then
  echo "Expected no output, got: $out" >&2
  exit 1
fi
