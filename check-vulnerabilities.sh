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
  run_govulncheck() {
    if command -v govulncheck >/dev/null 2>&1; then
      govulncheck -show verbose ./... 2>&1
    else
      go run golang.org/x/vuln/cmd/govulncheck@latest -show verbose ./... 2>&1
    fi
  }
  if run_govulncheck; then
    echo ""
    echo "Result: No known vulnerabilities reported."
  else
    echo ""
    echo "Result: One or more vulnerabilities may be present. Review output above."
  fi
  echo ""
  echo "## Remediation"
  echo ""
  echo "- **Go standard library:** If the report lists vulnerabilities in \`stdlib\` or packages like \`net/url\`, \`archive/zip\`, \`crypto/tls\`, upgrade the Go toolchain to the version shown in \"Fixed in\" (see \`go.mod\` \`toolchain\` directive). You cannot remove the standard library."
  echo "- **Third-party modules:** Run \`go get -u ./...\` or \`go get -u module@version\` to update to fixed versions, or remove the dependency if unused."
} > "$VULN_DOC"

echo "Wrote $VULN_DOC"
