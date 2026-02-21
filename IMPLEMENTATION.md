# Implementation process for ffd

This document lists commits in order. Each commit is a single, buildable change that implements part of [SPEC.md](SPEC.md). Work through the list from top to bottom. **When implementing (including when an AI is implementing), follow this plan in order and verify that the resulting CLI conforms to SPEC.md.**

**Maintain [SUMMARY.md](SUMMARY.md):** Update it after each commit with what you did, how it went, and any problems. As a final step (after all commits), update SUMMARY.md to indicate how the completed work aligns with SPEC.md.

**For each commit:**

1. **Current commit** — Use the commit indicated by "Next:" below.
2. **TDD** — For each new function: write a **failing** unit test first, then implement until the test passes. If you cannot write a failing test first, stop and ask for directions.
3. **Smoke tests** — If this commit introduces behavior testable via the CLI, add (or extend) a smoke test in this commit; extend the harness if needed.
4. **Implement** — Do the work in that commit’s bullet list (and any smoke test bullet).
5. **Before committing** — Run these checks: (1) build (`./build.sh` or `go build -ldflags="-s -w" -o bin/ffd .`), (2) unit tests (`./test.sh`), (3) smoke tests (`./smoke-tests.sh`, once the harness and at least one test exist, from Commit 1 onward).
6. **Do not commit** if the code does not build or if unit or smoke tests fail.
7. **Commit** — Use the **Message:** line from this file as the commit subject and the **Description:** paragraph as the body.
8. **Update SUMMARY.md** — Add an entry for this commit: what you did, how it went, and any problems.
9. **Update plan** — Set "Next:" to the following commit and mark the completed commit with `[x]`.

**Progress:** Mark the current commit below (e.g. set "Next: Commit N") and check off commits as you complete them (`[x]`).

**Commit message and description:**
- **Message (first line):** Simple, descriptive, and state the overall intent of the commit.
- **Description (body):** Longer explanation of what the commit was for, what it achieves as a stepping stone, and how it relates to SPEC.md.
- **When making a commit:** Use the **Message** and **Description** from the current commit block in this file (the one indicated by "Next:"). Each commit block below has a **Message:** line (exact first line for `git commit -m`) and a **Description:** paragraph (use as the commit body, e.g. `git commit` then paste the description).

**Test-driven development (TDD):** For each function you create, write a **failing** test first, then implement the function so the test passes. If you cannot create a failing test before writing a function, stop and ask for directions.

**Smoke tests as you go:** Write smoke tests as you go where possible. For each commit that introduces behavior testable via the CLI (`bin/ffd`), add a smoke test in that commit. Introduce or extend the smoke test harness when needed so new tests can run. The "Before each commit" check runs smoke tests once the harness exists (from Commit 1 onward).

- **Next:** Commit 3

---

## - [x] Commit 1: Add Cobra and root command

**Message:** Add Cobra and root command

**Description:** This commit establishes the CLI skeleton required by SPEC (CLI / help, Cobra). It adds the root command with the two directory arguments (left, right), ensures no-args prints help, and validates that both paths exist and are directories. It also defines the exit code scheme (0/1/2/3) so all later code can exit consistently. No diff logic yet; the program exits after validation or after printing help.

- Add Cobra dependency; create root command with two positional args (left dir, right dir).
- No args → Cobra prints help and usage.
- Validate both args exist and are directories; exit 1 (usage) or 2 (fatal) with clear errors.
- Define exit code constants: 0 success, 1 usage, 2 fatal, 3 non-fatal errors occurred.
- Add minimal smoke test harness (e.g. `smoke-tests.sh` that runs a list of tests) and one smoke test: no args → help (exit 0, help text on stdout).

## - [x] Commit 2: Add logging layer (Logger)

**Message:** Add logging layer (Logger)

**Description:** Implements the logging and error-handling requirements from SPEC (Logger, two log files, fatal vs non-fatal, printing log paths). Introduces the Logger type and its API so every subsequent commit can log discoveries and errors without duplicating logic. This is the foundation for "never swallow an error" and for the user to inspect log files after a run.

- Introduce a **Logger** struct: temp dir path, two open log file paths (main + errors), non-fatal error count, mutex for concurrent use. Create temp dir with `os.MkdirTemp`; open two files with names like `ffd-YYYYMMDD-NNN-main.log` and `ffd-YYYYMMDD-NNN-errors.log` (date + sequence so old logs are not overwritten).
- **Logger methods:** `Log(msg string)` — write to main log only. `LogError(err error)` — write to both logs, increment non-fatal count. `Fatal(err error)` — write to both logs, print to stderr, `os.Exit(2)`. `PrintLogPaths()` — print the two log file paths to stderr (skip when stdout is not a TTY). `NonFatalCount() int` — return current non-fatal count.
- Create one Logger instance after validating args; pass it to all components that need to log. Call `PrintLogPaths()` (and close/flush) before exit.
- Use this Logger for all later work.
- Add smoke test: run with two existing empty dirs; expect exit 0 and log path message (or no diff output).

## - [ ] Commit 3: Path representation and safety

**Message:** Path representation and safety

**Description:** Implements the memory and security rules from SPEC: single path representation (two roots once, one relative path per pair), path interning to avoid duplicate strings, and safe resolution that rejects paths outside the two roots. This enables the rest of the pipeline to work with relative paths only and prevents path traversal.

- Single path representation: two roots once, one relative path per pair; intern/dedupe relative path strings.
- Resolve paths and reject any path outside the two roots.
- Unit tests for path normalization and safety.

## - [ ] Commit 4: Discovered-files set and pair formation

**Message:** Discovered-files set and pair formation

**Description:** Implements the discovery model from SPEC (Speed strategy): a set keyed by relative path so that when the same path is seen in both trees we form a pair and enqueue it for comparison. This is the bridge between the two directory walks and the worker queue; no comparison logic yet.

- Set keyed by relative path for discovered files; when same path appears in both trees, form a pair and enqueue.
- Unit tests for set and pair formation (e.g. testdata dirs).
- Add smoke test: run with two dirs where one path is invalid or missing; expect exit 1 or 2 and clear error.

## - [ ] Commit 5: Walk one tree and log entries

**Message:** Walk one tree and log entries

**Description:** Implements recursive directory walking for one tree per SPEC (Scope: recurse, include hidden/dotfiles, regular files only). Logs every directory and file discovered so the main log satisfies the spec. Portable implementation only; Linux batching comes in a later commit.

- Walk one directory tree recursively; include hidden/dotfiles; regular files only; log every dir and file.
- Portable implementation (no Linux-specific batch yet).
- Unit tests with small test trees.

## - [ ] Commit 6: Walk both trees in parallel and feed queue

**Message:** Walk both trees in parallel and feed queue

**Description:** Completes the Speed strategy for directory reading: two goroutines (one per tree), no back pressure, both feeding the same set/queue so pairs are discovered and enqueued as we go. Enables the progress indicator’s "pending" count to grow during the walk.

- Second goroutine for the other tree; no back pressure; both walks feed the same set/queue so pairs are emitted as discovered.
- Log every directory and file discovered (both trees).
- Add smoke test: two identical empty dirs → no diff output (or empty diff).

## - [ ] Commit 7: Linux batched directory read (optional)

**Message:** Linux batched directory read (optional)

**Description:** Optimizes directory reading on Linux per SPEC (Speed strategy: batch reads, --dir-batch-size). Optional: portable fallback remains for non-Linux; this commit reduces syscalls on Linux only.

- On Linux, use batched directory reads (e.g. getdents64) with configurable batch size; guard with build tag or runtime check.
- Keep portable fallback for non-Linux.

## - [ ] Commit 8: Worker pool and queue processing

**Message:** Worker pool and queue processing

**Description:** Implements the worker model from SPEC (Speed strategy, Memory strategy): a fixed number of workers N (default NumCPU, --workers) pulling from the queue and processing pairs. Comparison is still a stub (e.g. report all as different); this commit wires up concurrency and progress (processed vs pending on stderr when TTY).

- Queue of work items (relative path only); N workers (default `runtime.NumCPU`, `--workers` flag) pull and process pairs.
- Workers stat both files and call comparison (stub or minimal: e.g. report all as different).
- Progress: processed vs pending on stderr when stderr is a TTY.
- Add smoke test: two identical dirs (same files) → no differences reported; two dirs with one file different → difference reported.

## - [ ] Commit 9: Size and mtime comparison

**Message:** Size and mtime comparison

**Description:** Implements the first part of the comparison logic from SPEC (Speed strategy, Output reasons): same size and same mtime → skip; different size → record "size changed" without hashing. Normalizes mtime granularity for cross-filesystem consistency. Sets up the branch that will trigger hashing in the next commit.

- For each pair: stat size and mtime; normalize mtime granularity.
- Same size + same mtime → skip (same file).
- Different size → record “size changed”, no hash.
- Same size + different mtime → mark for hash (next commit).
- Unit tests for size/mtime logic.
- Add smoke test: two dirs with one file same size but different content → diff reported (after hash).

## - [ ] Commit 10: Hash selection and content comparison

**Message:** Hash selection and content comparison

**Description:** Completes content comparison per SPEC (Speed strategy, hash options): when size matches but mtime differs, compute the selected hash (default xxHash, multiple algorithms via --hash). Implements threshold for read-full vs stream, buffer reuse, and documents hash choices in help/usage. Delivers "content differs" and per-file hash in output.

- Support multiple hash algorithms; **default xxHash**. Add CLI argument (e.g. `--hash`) to select the algorithm; document it and list available names (e.g. xxhash, sha256, md5) in help and usage.
- When size matches but mtime differs, compute the selected hash (hex); threshold for read-full vs stream (default 10MB, `--threshold`); reuse buffers (e.g. sync.Pool).
- Report “content differs” when hashes differ.
- Unit tests for hashing and comparison (at least for default and one other algorithm).
- Add smoke test: run with `--hash xxhash` (and optionally `--hash sha256`); same content → no diff, different content → diff with hash in output.

## - [ ] Commit 11: Output format: text (ASCII tree)

**Message:** Output format: text (ASCII tree)

**Description:** Implements the default output format from SPEC (Output: text, ASCII tree, case-sensitive sort, per-file details). Streams to stdout with progress on stderr. Enables users to see differing files in a tree and understand why each is different (size, time, hash, left-only, right-only).

- Stream diff results to stdout; progress remains on stderr.
- Text format: ASCII tree, case-sensitive sort; per-file: path, size, mtime, hash (if any), reason, left-only/right-only.
- Wire `--format text` (or default).
- Add smoke test: run with default or `--format text`; diff output is tree-shaped and includes path/size/reason.

## - [ ] Commit 12: Output format: table

**Message:** Output format: table

**Description:** Adds the table output format per SPEC (Output formats). Same per-file fields as text but in tabular form without a tree, for scripting or quick scanning.

- Table format: no tree; same per-file fields in columns.
- Wire `--format table`.
- Add smoke test: run with `--format table`; output is tabular (e.g. columns), no tree.

## - [ ] Commit 13: Output format: JSON

**Message:** Output format: JSON

**Description:** Adds the JSON output format per SPEC (Output formats). Tree structure in JSON for integration with other tools or pipelines.

- JSON format: tree structure in JSON.
- Wire `--format json`.
- Add smoke test: run with `--format json`; stdout is valid JSON (e.g. parseable).

## - [ ] Commit 14: Output format: YAML

**Message:** Output format: YAML

**Description:** Adds the YAML output format per SPEC (Output formats). Tree structure in YAML for human-readable or tool-consumable output.

- YAML format: tree structure in YAML.
- Wire `--format yaml`.
- Add smoke test: run with `--format yaml`; stdout is valid YAML (e.g. parseable).

## - [ ] Commit 15: Remaining CLI and exit behavior

**Message:** Remaining CLI and exit behavior

**Description:** Finishes CLI behavior per SPEC (--quiet, exit 3 when non-fatal errors occurred, empty roots and only-on-one-side handled consistently). Ensures script-friendly and interactive use both behave as specified.

- `--quiet`: suppress progress and “check error log” message when piping.
- Empty roots and “only on one side” handled consistently.
- Exit 3 when non-fatal errors occurred; unless `--quiet`, tell user to check error log.
- Add smoke test: run with `--quiet` and pipe stdout; no progress or "check error log" in output.

## - [ ] Commit 16: smoke-tests.sh harness complete and remaining smoke tests

**Message:** smoke-tests.sh harness complete and remaining smoke tests

**Description:** Completes the smoke test harness per SPEC (Testing, Scripts): add `ls` and run-one-by-name to smoke-tests.sh so all tests from commits 1–15 are runnable by name and listable. Add any remaining test data under ./test (e.g. left-only, right-only scenarios) and smoke tests not yet covered. Ensures full coverage: help, identical dirs, one diff, left-only, right-only, each format.

- Extend smoke-tests.sh: add `ls` (list test names) and run one by name (e.g. `./smoke-tests.sh <test-name>`).
- Add test data under `./test` for left-only, right-only, and any missing scenarios.
- Add or consolidate smoke tests so all of: help, identical dirs, one file different, left-only, right-only, and each output format are covered.

## - [ ] Commit 17: perf-test.sh and perf-results.csv

**Message:** perf-test.sh and perf-results.csv

**Description:** Implements performance testing per SPEC (Testing: performance tests). Script builds optimized binary, generates data under ./test, runs scenarios at each file count, writes timing and appends to perf-results.csv (ISO date, columns per spec) for charting over time.

- perf-test.sh: build optimized binary, generate data under `./test`, run scenarios at 0, 1, 10, 100, 1K, 10K, 100K files.
- Human-readable timing output; append to perf-results.csv (ISO date, columns per spec).

## - [ ] Commit 18: CI workflow

**Message:** CI workflow

**Description:** Implements CI per SPEC (CI): GitHub Actions workflow using existing scripts (build, test, smoke-tests) and security checks (go mod verify, govulncheck). Ensures every push/PR is built and tested on Linux.

- `.github/workflows/ci.yml`: use existing scripts (build, test, smoke-tests); add security checks (go mod verify, govulncheck).
- Build and test on Linux.

## - [ ] Commit 19: Release workflow

**Message:** Release workflow

**Description:** Implements release automation per SPEC (Release): workflow on release/tag, use scripts, build for Windows/Linux/macOS (ffd.exe on Windows), run smoke tests, create GitHub Release with executables and checksums (Security: checksummed downloads). Bash used for Windows build/test in GHA where appropriate.

- `.github/workflows/release.yml`: trigger on release/tag; use scripts; build Windows/Linux/macOS; run smoke tests; create GitHub Release with executables and checksums.
- Use Bash for Windows build/test in GHA where appropriate.

---

**Final step (after all commits):** Update [SUMMARY.md](SUMMARY.md) to indicate how the completed work aligns with SPEC.md (which spec sections are satisfied, any gaps or deviations).

---

## Principles

- Use test-driven development: write a failing test for each function before implementing it; then write the code to make the test pass. If you cannot write a failing test first, stop and ask for directions.
- Write smoke tests as you go: for each commit that introduces CLI-testable behavior, add a smoke test; extend the harness when needed.
- Each commit should build and, where relevant, tests should pass.
- One clear theme per commit; keep diffs reviewable (~50–150 lines for larger commits).
- Log all errors and important events from Commit 2 onward.
