#!/usr/bin/env bash
# Pre-commit gate: format, vet, build, unit tests, smoke tests. Do not commit if this fails.
# Usage: ./check.sh (run from project root)
set -e
echo "=== format check ==="
unformatted=$(gofmt -l . 2>/dev/null | grep -v '^$' || true)
if [ -n "$unformatted" ]; then
	echo "Files need 'gofmt -w':"
	echo "$unformatted"
	exit 1
fi
echo "=== vet ==="
go vet ./...
echo "=== build ==="
./build.sh
echo "=== unit tests ==="
./test.sh
echo "=== smoke tests ==="
./smoke-tests.sh
echo "=== check passed ==="
