#!/usr/bin/env bash
# ffd ls on empty directory → no lines on stdout
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
DIR="${SCRIPT_DIR}/ls-empty"
mkdir -p "$DIR"
out=$("$BIN" ls "$DIR" 2>/dev/null)
if [[ -n "$out" ]]; then
  echo "Expected no output, got: $out" >&2
  exit 1
fi
