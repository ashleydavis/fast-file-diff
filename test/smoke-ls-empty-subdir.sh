#!/usr/bin/env bash
# ffd ls on directory that contains only an empty subdir → no files listed
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
DIR="${SCRIPT_DIR}/ls-empty-subdir"
mkdir -p "$DIR/sub"
out=$("$BIN" ls "$DIR" 2>/dev/null)
if [[ -n "$out" ]]; then
  echo "Expected no output, got: $out" >&2
  exit 1
fi
