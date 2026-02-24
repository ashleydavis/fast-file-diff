# fast-file-diff

A fast CLI that reports which files differ between two directories (by path and content), optimized for speed on large trees.

## Quick start (latest release)

Pre-built binaries are on the [Releases](https://github.com/ashleydavis/fast-file-diff/releases/latest) page. Download the file for your platform:

| Platform        | File                |
|-----------------|---------------------|
| Linux (amd64)   | ffd-linux-amd64     |
| Linux (arm64)   | ffd-linux-arm64     |
| Windows (amd64) | ffd.exe             |
| macOS (Intel)   | ffd-darwin-amd64    |
| macOS (Apple Silicon) | ffd-darwin-arm64 |

Then run a diff:

```bash
# Linux/macOS: make executable, then run
chmod +x ffd-linux-amd64
./ffd-linux-amd64 /path/to/left /path/to/right
```

Windows: `ffd.exe C:\path\to\left C:\path\to\right`. Verify downloads with the SHA-256 checksums on the release page.

## Quick start (from source)

Build and run a diff from source code:

```bash
./build.sh
./bin/ffd /path/to/left /path/to/right
```

Run `./bin/ffd --help` for options (formats, hash algorithm, workers). See [Using the CLI](#using-the-cli) for details.

## Scripts

| Script | Purpose |
|--------|--------|
| `run.sh` | Run the program (`go run .`) |
| `build.sh` | Build optimized binary to `bin/ffd` |
| `test.sh` | Run unit tests (`go test .`) |
| `check.sh` | Build, run all unit tests, then all smoke tests (full gate) |
| `smoke-tests.sh` | Run all smoke tests |
| `smoke-tests.sh <test-name>` | Run one smoke test |
| `smoke-tests.sh ls` | List tests that can be run individually |
| `perf-test.sh` | Run performance tests (optimized build; generates data under `./test/perf/tmp/`, appends to `./perf/perf-results.csv`; may take a long time) |
| `check-vulnerabilities.sh` | Run vulnerability checks (go mod verify, govulncheck if installed) and write results to `docs/VULNERABILITIES.md` |

See [docs/SPEC.md](docs/SPEC.md) for full details on each script.

## Run, build, and test

```bash
./run.sh    # run the program
./build.sh  # build binary to bin/ffd
./test.sh   # run tests
```

## Using the CLI

Run `./bin/ffd --help` for full usage. Summary of commands and options:

### Commands

| Command | Description |
|---------|--------------|
| `ffd <left-dir> <right-dir>` | Compare two directory trees (default). |
| `ffd ls <directory>` | List files recursively (one path per line). |
| `ffd version` | Print version to stdout. Same as `--version`. |

### Options (flags)

These apply to the diff command; `ls` also accepts `--dir-batch-size`.

| Option | Default | Description |
|--------|---------|--------------|
| `--dir-batch-size` | 4096 | Directory read batch size (entries per syscall). |
| `--workers` | number of CPUs | Worker goroutines for comparing file pairs. |
| `--hash` | xxhash | Hash for content comparison: `xxhash`, `sha256`, `md5`. |
| `--threshold` | 10485760 (10 MiB) | Size in bytes: files smaller are read in full to hash, larger are streamed. |
| `--format` | text | Output format: `text`, `table`, `json`, `yaml`. |
| `--quiet` | false | Suppress progress, left/right directory lines, summary on stderr, and final error-log message (for scripting). |
| `--full` | false | Always hash file contents for every pair; do not skip hashing when size and mtime already match (slower but verifies content). |

### Diff (default)

```bash
./bin/ffd <dir1> <dir2>
```

- **Arguments:** two directory paths (e.g. `./bin/ffd /path/to/a /path/to/b`).
- **Output:** list of files that are "different" between the two trees. At start (unless `--quiet`) the left and right directories are printed to stderr; at the end a summary is printed to stderr including those paths again, plus counts and timings. Exit code 0 when run completes; non-zero on usage or I/O errors. See [docs/SPEC.md](docs/SPEC.md) for how the diff works.

### ls — list files recursively

```bash
./bin/ffd ls <directory>
```

- **Arguments:** one directory path. Walks the tree and prints the relative path of every file (one per line) to stdout.
- **Output:** one path per line; progress and summary (file count, duration) to stderr. Uses batched directory reads; batch size is configurable with `--dir-batch-size` (default 4096).
- **Example:** `./bin/ffd ls /media/backup/photos` or `./bin/ffd --dir-batch-size 8192 ls /media/backup/photos`.

### version — print version number

```bash
./bin/ffd version
./bin/ffd --version
```

- **Output:** version string to stdout (e.g. `dev` for local builds, or the release tag such as `v1.0.0` when built with `-ldflags "-X main.Version=..."`). Exit 0. Script-friendly.

Implementation must follow [docs/SPEC.md](docs/SPEC.md). Keep this README updated with the actual functionality of the CLI (arguments, options, output formats, and usage). Create unit tests for every function that is created.

## Smoke tests

Create smoke tests implemented as shell scripts that run against the compiled executable (`bin/ffd`). Use a comprehensive set of small tests; put example data and scenarios under a `./test` directory. Each smoke test must be independent (so they can be parallelized later). Invocation:

- Run all: `./smoke-tests.sh` (no arguments)
- Run one: `./smoke-tests.sh <test-name>`
- List tests: `./smoke-tests.sh ls`

## Performance tests

Performance tests are in their own script (e.g. `./perf-test.sh`) and may take a long time. They must run against an optimized build: build with `go build -ldflags="-s -w" -o bin/ffd .` (do not use `-race` or `-gcflags="-N -l"`). The script generates temporary test trees under `./test/perf/tmp/`, runs scenarios at 0, 1, 10, 100, 1K, 10K, and 100K files: **all_same_metadata** (same size and mtime → content never read), **all_same_content** (same content, different mtime → every file read), **all_different_size** (different sizes → no read), **all_different_mtime** (same size, different mtime/content → read), **left_only**, **right_only**, and writes timing output. Results are appended to `./perf/perf-results.csv` (one row per run: date_iso, machine, workers, min_sec_per_pair, max_sec_per_pair, avg_sec_per_pair, longest_test_total_sec, longest_test). The machine column records host OS, arch, core count, and CPU model; min/max/avg are across all scenario/file_count tests; longest_test_total_sec is the total time of the slowest test; longest_test is that test’s identifier (e.g. all_same_content_100000). See [docs/SPEC.md](docs/SPEC.md) for full details.

## For developers

- **Tests:** Run unit tests with `./test.sh` or `go test ./...`. Run the full gate (build + unit + smoke) with `./check.sh`.
- **Code layout:** The project root contains only `main.go` (CLI entrypoint); all library code lives under `lib/`.
- **Docs:** [docs/SPEC.md](docs/SPEC.md) (behavior and scripts), [docs/IMPLEMENTATION.md](docs/IMPLEMENTATION.md), [docs/FOLLOWUP.md](docs/FOLLOWUP.md), [docs/SUMMARY.md](docs/SUMMARY.md).

## Adding modules

Add a dependency (and update `go.mod` / `go.sum`):

```bash
go get <module-path>@<version>   # e.g. go get github.com/foo/bar@v1.2.0
go get <module-path>             # latest version
```

Then run `go mod tidy` to drop unused deps and fix the module graph.
