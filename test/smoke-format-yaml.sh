#!/usr/bin/env bash
# --format yaml: compare output to known-good expected (normalize timestamps for portability)
# Use our own dirs with fixed sizes (5 and 6 bytes) so expected output is identical on all platforms.
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
TMP="${SCRIPT_DIR}/tmp"
LEFT="${TMP}/format-yaml-left"
RIGHT="${TMP}/format-yaml-right"
mkdir -p "$LEFT" "$RIGHT"
printf '%s' "12345" > "$LEFT/f"
printf '%s' "123456" > "$RIGHT/f"
out=$("$BIN" --format yaml "$LEFT" "$RIGHT" 2>/dev/null)
# Normalize timestamps so comparison is portable
actual=$(printf '%s' "$out" | tr -d '\r' | sed -E 's/"[0-9]{4}-[0-9]{2}-[0-9]{2}T[^"]*"/"TIMESTAMP"/g')
expected=$(tr -d '\r' < "$SCRIPT_DIR/format-yaml-expected.yaml")
if [[ "$actual" != "$expected" ]]; then
  echo "YAML output did not match expected (format-yaml-expected.yaml)" >&2
  echo "--- actual (normalized) ---" >&2
  echo "$actual" >&2
  echo "--- expected ---" >&2
  echo "$expected" >&2
  exit 1
fi
