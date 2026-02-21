# Vulnerability check report

**Date:** 2026-02-22T08:47:37+10:00

## go mod verify

all modules verified

Result: OK

## govulncheck ./...

Fetching vulnerabilities from the database...

Checking the code against the vulnerabilities...

The package pattern matched the following 2 root packages:
  github.com/photosphere/fast-file-diff-go/lib
  github.com/photosphere/fast-file-diff-go
Govulncheck scanned the following 5 modules and the go1.25.7 standard library:
  github.com/photosphere/fast-file-diff-go
  github.com/cespare/xxhash/v2@v2.3.0
  github.com/spf13/cobra@v1.10.2
  github.com/spf13/pflag@v1.0.9
  gopkg.in/yaml.v3@v3.0.1

No vulnerabilities found.

Result: No known vulnerabilities reported.

## Remediation

- **Go standard library:** If the report lists vulnerabilities in `stdlib` or packages like `net/url`, `archive/zip`, `crypto/tls`, upgrade the Go toolchain to the version shown in "Fixed in" (see `go.mod` `toolchain` directive). You cannot remove the standard library.
- **Third-party modules:** Run `go get -u ./...` or `go get -u module@version` to update to fixed versions, or remove the dependency if unused.
