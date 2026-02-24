#!/usr/bin/env bash
# --full: always hash every pair; exit 0 and correct diff results
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
TMP="${SCRIPT_DIR}/tmp"

# Identical dirs with --full: no diff, exit 0
LEFT="${TMP}/full-identical-left"
RIGHT="${TMP}/full-identical-right"
mkdir -p "$LEFT" "$RIGHT"
printf '%s' "same" > "$LEFT/f"
printf '%s' "same" > "$RIGHT/f"
out=$("$BIN" --full --format text "$LEFT" "$RIGHT" 2>/dev/null)
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0 with --full on identical dirs, got $exitcode" >&2
  exit 1
fi
if [[ -n "$out" ]] && echo "$out" | grep -qE "size changed|content differs"; then
  echo "Identical dirs with --full should not show size/content diff, got: $out" >&2
  exit 1
fi

# Same size, different content with --full: content differs reported
LEFT="${TMP}/full-mtime-left"
RIGHT="${TMP}/full-mtime-right"
mkdir -p "$LEFT" "$RIGHT"
printf '%s' "aa" > "$LEFT/f"
sleep 1
printf '%s' "bb" > "$RIGHT/f"
out=$("$BIN" --full "$LEFT" "$RIGHT" 2>/dev/null)
exitcode=$?
if [[ $exitcode -ne 0 ]]; then
  echo "Expected exit 0 with --full on same-size diff content, got $exitcode" >&2
  exit 1
fi
if ! echo "$out" | grep -q "content differs"; then
  echo "Expected 'content differs' with --full for same-size different content, got: $out" >&2
  exit 1
fi
