#!/usr/bin/env bash
# No args → help on stdout, exit 0
set -e
BIN="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/bin/ffd"
out=$("$BIN" 2>/dev/null)
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0, got $exitcode" >&2
  exit 1
fi
if ! echo "$out" | grep -q "Usage:"; then
  echo "Expected help (Usage:) on stdout, got: $out" >&2
  exit 1
fi
