#!/usr/bin/env bash
# Pre-commit gate: build, unit tests, smoke tests. Do not commit if this fails.
# Usage: ./check.sh (run from project root)
set -e
echo "=== build ==="
./build.sh
echo "=== unit tests ==="
./test.sh
echo "=== smoke tests ==="
./smoke-tests.sh
echo "=== check passed ==="
