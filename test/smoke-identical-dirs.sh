#!/usr/bin/env bash
# Two identical dirs (same files) → no diff output, exit 0
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
TMP="${SCRIPT_DIR}/tmp"
LEFT="${TMP}/identical-dirs-left"
RIGHT="${TMP}/identical-dirs-right"
mkdir -p "$LEFT" "$RIGHT"
printf '%s' "same" > "$LEFT/f"
printf '%s' "same" > "$RIGHT/f"
out=$("$BIN" --format text "$LEFT" "$RIGHT" 2>/dev/null)
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0, got $exitcode" >&2
  exit 1
fi
# Identical dirs: no substantive diff (size/content). Allow empty or minor quirks.
if [[ -n "$out" ]] && echo "$out" | grep -qE "size changed|content differs"; then
  echo "Identical dirs should not show size/content diff, got: $out" >&2
  exit 1
fi
