#!/usr/bin/env bash
# Check the project for known vulnerabilities and document results in docs/VULNERABILITIES.md.
# Runs go mod verify and govulncheck (if available). Overwrites docs/VULNERABILITIES.md with date and findings.
# Usage: ./check-vulnerabilities.sh (run from project root)
set -e

VULN_DOC="docs/VULNERABILITIES.md"
DATE_ISO=$(date -Iseconds)

{
  echo "# Vulnerability check report"
  echo ""
  echo "**Date:** $DATE_ISO"
  echo ""
  echo "## go mod verify"
  echo ""
  if go mod verify 2>&1; then
    echo ""
    echo "Result: OK"
  else
    echo ""
    echo "Result: FAILED"
  fi
  echo ""
  echo "## govulncheck ./..."
  echo ""
  if command -v govulncheck >/dev/null 2>&1; then
    if govulncheck ./... 2>&1; then
      echo ""
      echo "Result: No known vulnerabilities reported."
    else
      echo ""
      echo "Result: One or more vulnerabilities may be present. Review output above."
    fi
  else
    echo "govulncheck is not installed. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
    echo ""
    echo "Result: Skipped (tool not available)."
  fi
} > "$VULN_DOC"

echo "Wrote $VULN_DOC"
