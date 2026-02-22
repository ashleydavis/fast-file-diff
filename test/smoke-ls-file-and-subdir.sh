#!/usr/bin/env bash
# ffd ls on dir with one file and one subdir containing one file → two lines
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
DIR="${SCRIPT_DIR}/ls-file-and-subdir"
if [[ ! -d "$DIR" ]]; then
  echo "Missing test dir $DIR" >&2
  exit 1
fi
out=$(echo "$("$BIN" ls "$DIR" 2>/dev/null)" | sort)
want=$'sub/nested\ntop'
if [[ "$out" != "$want" ]]; then
  echo "Expected 'top' and 'sub/nested', got: $out" >&2
  exit 1
fi
