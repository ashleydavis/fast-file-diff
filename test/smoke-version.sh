#!/usr/bin/env bash
# version command and --version flag print version and exit 0
set -e
BIN="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/bin/ffd"
for flag in "version" "--version"; do
  out=$("$BIN" $flag 2>/dev/null)
  exitcode=$?
  if [[ $exitcode -ne 0 ]]; then
    echo "Expected exit 0 for $flag, got $exitcode" >&2
    exit 1
  fi
  if [[ -z "$out" ]]; then
    echo "Expected non-empty version output for $flag, got: $out" >&2
    exit 1
  fi
  if [[ "$out" != *[0-9a-zA-Z.]* ]]; then
    echo "Expected version-like output for $flag, got: $out" >&2
    exit 1
  fi
done
