#!/usr/bin/env bash
# Wrong number of arguments (e.g. one arg) → exit 1 (usage)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/../bin/ffd"
VALID="${SCRIPT_DIR}/empty-left"
set +e
"$BIN" "$VALID" 2>/dev/null
exitcode=$?
set -e
if [[ $exitcode -ne 1 ]]; then
  echo "Expected exit 1 for one argument (usage error), got $exitcode" >&2
  exit 1
fi
