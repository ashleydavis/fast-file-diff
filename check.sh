#!/usr/bin/env bash
# Pre-commit gate: build, unit tests, smoke tests. Do not commit if this fails.
# Usage: ./check.sh
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"
echo "=== build ==="
./build.sh
echo "=== unit tests ==="
./test.sh
echo "=== smoke tests ==="
./smoke-tests.sh
echo "=== check passed ==="
