#!/usr/bin/env bash
# Smoke tests for ffd. Usage: ./smoke-tests.sh [ls|<test-name>]
#   no args  = run all tests
#   ls       = list test names that can be run individually
#   <name>   = run that one test (script must exist at ./test/smoke-<name>.sh)
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${SCRIPT_DIR}/bin/ffd"
TEST_DIR="${SCRIPT_DIR}/test"

# Add test names here; each should have a corresponding ./test/smoke-<name>.sh
TESTS=(help two-empty-dirs invalid-dir identical-dirs)

list_tests() {
  if [[ ${#TESTS[@]} -eq 0 ]]; then
    echo "No smoke tests defined. Add names to the TESTS array in smoke-tests.sh and create ./test/smoke-<name>.sh for each."
    return 0
  fi
  for t in "${TESTS[@]}"; do
    echo "$t"
  done
}

run_one() {
  local name="$1"
  local script="${TEST_DIR}/smoke-${name}.sh"
  if [[ ! -f "$script" ]]; then
    echo "Unknown or missing test: $name (expected $script)" >&2
    return 1
  fi
  if [[ ! -x "$BIN" ]]; then
    echo "Binary not found or not executable: $BIN (run ./build.sh first)" >&2
    return 1
  fi
  "$script"
}

run_all() {
  local failed=0
  if [[ ${#TESTS[@]} -eq 0 ]]; then
    echo "No smoke tests defined."
    return 0
  fi
  for t in "${TESTS[@]}"; do
    echo "=== smoke: $t ==="
    if run_one "$t"; then
      echo "--- $t: OK"
    else
      echo "--- $t: FAILED" >&2
      ((failed++)) || true
    fi
  done
  return $failed
}

case "${1:-}" in
  ls)
    list_tests
    ;;
  "")
    run_all
    ;;
  *)
    run_one "$1"
    ;;
esac
