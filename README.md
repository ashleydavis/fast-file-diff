# fast-file-diff-go

A fast CLI that reports which files differ between two directories (by path and content), optimized for speed on large trees.

## Scripts

| Script | Purpose |
|--------|--------|
| `run.sh` | Run the program (`go run .`) |
| `build.sh` | Build optimized binary to `bin/ffd` |
| `test.sh` | Run unit tests (`go test .`) |
| `smoke-tests.sh` | Run all smoke tests |
| `smoke-tests.sh <test-name>` | Run one smoke test |
| `smoke-tests.sh ls` | List tests that can be run individually |
| `perf-test.sh` | Run performance tests (optimized build; generates data under `./test/perf/tmp/`, appends to `./perf/perf-results.csv`; may take a long time) |
| `check-vulnerabilities.sh` | Run vulnerability checks (go mod verify, govulncheck if installed) and write results to `VULNERABILITIES.md` |

See [SPEC.md](SPEC.md) for full details on each script.

## Run, build, and test

```bash
./run.sh    # run the program
./build.sh  # build binary to bin/ffd
./test.sh   # run tests
```

## Using the CLI

```bash
./bin/ffd <dir1> <dir2>
```

- **Arguments:** two directory paths (e.g. `./bin/ffd /path/to/a /path/to/b`).
- **Options:** include output format (`--format`), hash algorithm (`--hash`, default xxhash; multiple options available), and others (see `./bin/ffd --help`). Run with no arguments or `--help` to see full usage and the list of hash algorithms.
- **Output:** list of files that are "different" between the two trees. Exit code 0 when run completes; non-zero on usage or I/O errors. See [SPEC.md](SPEC.md) for how the diff works.

Implementation must follow [SPEC.md](SPEC.md). Keep this README updated with the actual functionality of the CLI (arguments, options, output formats, and usage). Create unit tests for every function that is created.

## Smoke tests

Create smoke tests implemented as shell scripts that run against the compiled executable (`bin/ffd`). Use a comprehensive set of small tests; put example data and scenarios under a `./test` directory. Each smoke test must be independent (so they can be parallelized later). Invocation:

- Run all: `./smoke-tests.sh` (no arguments)
- Run one: `./smoke-tests.sh <test-name>`
- List tests: `./smoke-tests.sh ls`

## Performance tests

Performance tests are in their own script (e.g. `./perf-test.sh`) and may take a long time. They must run against an optimized build: build with `go build -ldflags="-s -w" -o bin/ffd .` (do not use `-race` or `-gcflags="-N -l"`). The script generates temporary test trees under `./test/perf/tmp/`, runs scenarios (all same, left-only, right-only) at 0, 1, 10, 100, 1K, 10K, and 100K files, and writes timing output. Results are appended to `./perf/perf-results.csv` (each row: date_iso, scenario, file_count, avg_sec_per_pair). See [SPEC.md](SPEC.md) for full details.

## Adding modules

Add a dependency (and update `go.mod` / `go.sum`):

```bash
go get <module-path>@<version>   # e.g. go get github.com/foo/bar@v1.2.0
go get <module-path>             # latest version
```

Then run `go mod tidy` to drop unused deps and fix the module graph.
