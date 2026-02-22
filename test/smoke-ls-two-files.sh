#!/usr/bin/env bash
# ffd ls on directory with two files → two lines (order may vary)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
DIR="${SCRIPT_DIR}/ls-two-files"
if [[ ! -d "$DIR" ]]; then
  echo "Missing test dir $DIR" >&2
  exit 1
fi
out=$(echo "$("$BIN" ls "$DIR" 2>/dev/null)" | sort)
want=$'a\nb'
if [[ "$out" != "$want" ]]; then
  echo "Expected a and b, got: $out" >&2
  exit 1
fi
