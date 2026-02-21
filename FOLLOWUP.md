# Follow-up plan for ffd

This document lists follow-up commits in order. Each commit is a single, buildable change. Work through the list from top to bottom.

**Maintain [SUMMARY.md](SUMMARY.md):** Before every commit, append to SUMMARY.md what was done, why it was done, and what it accomplishes. **Append the SUMMARY.md entry for that commit before running `git commit`** — do not commit and then update SUMMARY; update SUMMARY first, then commit.

**For each commit:**

1. **Current commit** — Use the commit indicated by "Next:" below.
2. **Implement** — Do the work in that commit’s bullet list.
3. **Before committing** — Apply these checks:
   - **Names:** Are all variable, function, and file names long and descriptive? If not, fix them.
   - **Unit tests:** Does every function have one or more unit tests that exercise it? If not, add them.
   - **Smoke tests:** Does every feature have one or more smoke tests that exercise it? If not, add them.
   - **Build:** The code must build (`./build.sh` or equivalent).
   - **Tests:** All unit and smoke tests must pass. Run `./check.sh`; do not commit if it exits non-zero.
   - **README:** Update the README as necessary for new features.
   - **SUMMARY:** Append to SUMMARY.md what was done, why it was done, and what it accomplishes for *this* commit. Do this **before** committing.
4. **Commit** — Use the **Message:** line from this file as the commit subject and the **Description:** paragraph as the body. (SUMMARY.md must already be updated for this commit.)
5. **Update plan** — Set "Next:" to the following commit and mark the completed commit with `[x]`.

**Progress:** Mark the current commit below (e.g. set "Next: Commit N") and check off commits as you complete them (`[x]`).

- **Next:** Commit 8 (README quick start)

**Final step:** Commit 9 verifies build, all tests, and README completeness before closing the plan.

---

## - [x] Commit 1: Progress indicator — estimate time remaining

**Message:** Progress indicator: estimate time remaining from average time per file

**Description:** Update the progress indicator on stderr so it shows an estimate of time remaining, not only "processed N, pending M". Track the average time required to compare each file pair (e.g. rolling average or total elapsed / processed so far). Use that to estimate time remaining based on the number of pairs still pending. Display the estimate in the progress line (e.g. "processed N, pending M, ~Xs remaining" or similar). This improves UX for large trees.

- In the progress loop (or wherever processed/pending are updated), maintain a measure of average time per comparison (e.g. start time when first pair is enqueued or first result arrives; average = elapsed / processed).
- Compute estimated remaining time as (pending count × average time per pair).
- Update the progress line printed to stderr to include the estimate (e.g. "processed N, pending M, ~Xs remaining"). Handle edge cases: no pending, or processed == 0 (no estimate yet).
- Add or extend unit tests for any new helper that computes average/estimate.
- Add a smoke test if the progress output is observable (e.g. run with a small tree and capture stderr; expect "processed" and "pending"; optionally expect "remaining" or similar). If progress is only on TTY, document that the smoke test may only check exit and stdout.
- Ensure all names are long and descriptive; every function has unit tests; every new feature has smoke coverage; build and `./check.sh` pass; update README if needed; append to SUMMARY.md.

---

## - [x] Commit 2: Perf CSV format and directory layout

**Message:** Perf: CSV records as avg time per pair only; move CSV to perf/, temp files to test/perf/tmp/

**Description:** Change the performance test CSV so each record stores only the average time required to compare one file pair (not total_sec and time_per_file_sec; just one metric per row). Move the CSV output from project root to the `perf/` directory. Move temporary performance test files (e.g. generated left/right trees) to `test/perf/tmp/` so the perf script no longer clobbers or mixes with other test data under `test/perf/`.

- Create `perf/` directory for output. Create `test/perf/tmp/` for temporary perf data (or use a subdir under `test/perf/` for generated trees).
- Update `perf-test.sh`: write CSV to `perf/perf-results.csv` (or similar). Ensure CSV header and rows record only the average time per file pair (one column for that metric, plus any needed identifiers like date_iso, scenario, file_count). Remove redundant columns if the spec was to have only avg time per pair per record.
- Generate temporary trees under `test/perf/tmp/` (e.g. `test/perf/tmp/left`, `test/perf/tmp/right`) so they are not under the project root and are clearly temporary.
- Update any references to the old CSV path or temp paths (e.g. in README, SPEC, or docs).
- Add or adjust unit tests if any code parses or produces this CSV; add smoke test only if there is CLI-observable behavior. Ensure names, unit tests, smoke tests, build, `./check.sh`, README, SUMMARY.md as above.

---

## - [x] Commit 3: Vulnerability check script and VULNERABILITIES.md

**Message:** Add vulnerability check script and document findings in VULNERABILITIES.md

**Description:** Add a script (e.g. `check-vulnerabilities.sh` or `scripts/check-vuln.sh`) that runs programmatic checks for known vulnerability classes (e.g. `govulncheck`, `go mod verify`, and any other checks you choose). The script must write or append its findings (and date of run) to a file named `VULNERABILITIES.md` in the project root, so that vulnerabilities are documented in one place. The script may run `govulncheck` and parse or summarize output into `VULNERABILITIES.md`; if no issues are found, document that too.

- Create the script (run from project root). It should run at least `go mod verify` and `govulncheck` (or equivalent).
- Script writes or overwrites `VULNERABILITIES.md` with: date of run, tool(s) run, and findings (or "No known vulnerabilities reported").
- Document in README or Scripts section that this script exists and updates `VULNERABILITIES.md`.
- No new Go code required; ensure existing code still builds and tests pass. Append to SUMMARY.md.

---

## - [x] Commit 4: Long descriptive names audit and fixes

**Message:** Audit and fix names: ensure all variables, functions, and files are long and descriptive

**Description:** Review the entire codebase for variable names, function names, and file names. Rename any that are abbreviated, unclear, or short (e.g. single-letter or cryptic) to long, descriptive names. This improves readability and maintainability. Do not change behavior; only rename.

- Audit `*.go` and scripts: variables (e.g. in loops, parameters, locals), function names, and file names (e.g. `path.go` may stay if it is descriptive; `discover.go` vs `discovered_set.go` — choose consistently).
- Rename identifiers and files as needed. Update all references (imports, call sites, tests).
- After renames, ensure every function still has unit tests and every feature has smoke tests; run `./check.sh`; update README if any user-facing names changed; append to SUMMARY.md.

---

## - [x] Commit 5: Unit test coverage — every function exercised

**Message:** Unit tests: ensure every function has one or more unit tests that exercise it

**Description:** Review the codebase and ensure that every function (exported and unexported, in `main` and in any packages) has at least one unit test that exercises it. Add tests where missing. This may require exporting previously unexported functions for testing or using test files in the same package. Goal: no function without test coverage.

- List all functions in the project (main and any lib code). For each, verify at least one test in `*_test.go` calls or triggers it.
- Add unit tests for any function that lacks coverage. Prefer same-package tests; use subprocess or integration-style tests only where necessary.
- Run `./check.sh`; fix any regressions. Append to SUMMARY.md with a brief list of what was added or fixed.

---

## - [x] Commit 6: Smoke test coverage — every feature exercised

**Message:** Smoke tests: ensure every feature has one or more smoke tests that exercise it

**Description:** Review all CLI and user-facing features (flags, output formats, exit codes, progress, logging behavior, etc.) and ensure each is covered by at least one smoke test. Add smoke tests where missing (e.g. `--hash sha256`, `--hash md5`, exit code 1 for wrong args, or other gaps identified earlier). Each feature should be exercised by at least one scenario in `./smoke-tests.sh`.

- List features (from SPEC and README): help, formats, hashes, --quiet, --workers, exit 0/1/2/3, left-only/right-only, etc.
- For each feature, confirm a smoke test exists that runs the CLI in a way that would fail if that feature were broken. Add tests where missing.
- Run `./check.sh`; update README if new tests are added; append to SUMMARY.md.

---

## - [x] Commit 7: Move library code to lib/; only main.go in root

**Message:** Move all library code to lib/; keep only main.go in project root

**Description:** Refactor the project so that all library-style code (everything except the entrypoint) lives under a `lib/` directory (e.g. `lib/` package or `lib/ffd` or similar). Only `main.go` and the `main` package remain in the project root so that the root contains only the CLI entrypoint and the rest is clearly library code. Update imports, build, and tests accordingly.

- Create `lib/` (or `lib/ffd` if you use a Go module subpath). Move all Go files except `main.go` into `lib/` and adjust package declarations. If the current code is all `package main`, introduce a package (e.g. `package ffd`) for the lib code and have `main.go` import and call it.
- Move `logger.go`, `path.go`, `discover.go`, `walk.go`, `walk_linux.go`, `walk_nonlinux.go`, `compare.go`, `hash.go`, `output.go` (and their tests) into `lib/`. Keep `main.go` in root with `package main`.
- Update `main.go` to import the lib package and use its types and functions. Update build and test commands so that the lib is built and tested (e.g. `go build .` from root, `go test ./...`).
- Ensure all unit tests and smoke tests still pass; update README if build or layout changes; append to SUMMARY.md.

---

## - [ ] Commit 8: README quick start section

**Message:** Add quick start section to README

**Description:** Add a "Quick start" (or "Quick start guide") section near the top of README.md that shows how to use the tool as quickly as possible: build or run, then one or two example commands (e.g. compare two dirs, or help). Keep it short so new users can run a diff in under a minute.

- Add a "Quick start" section after the opening description (or after Scripts). Include: how to build (or `go run`), then one or two example commands (e.g. `./bin/ffd /path/left /path/right`, and optionally `./bin/ffd --help`). Optionally one line on expected output.
- Do not duplicate the full "Using the CLI" section; link to it or keep quick start to 3–5 lines.
- Append to SUMMARY.md.

---

## - [ ] Commit 9: Final verification — build, tests, and README

**Message:** Final verification: build, all tests pass, README complete for users and developers

**Description:** Final follow-up step: confirm the code compiles, all unit and smoke tests pass, and the README contains all information needed by users and by future developers working on the tool. Fix any gaps (e.g. missing sections, outdated scripts, or broken tests) and document completion in SUMMARY.md.

- **Build:** Run `./build.sh` (or equivalent). Confirm the code compiles with no errors.
- **Tests:** Run `./check.sh`. Confirm all unit tests and all smoke tests pass. If any fail, fix them before committing this step.
- **README:** Review README.md for completeness. Ensure it includes everything relevant for:
  - **Users:** how to build/run, quick start, CLI usage, arguments, options, output formats, scripts (build, test, smoke, perf), and where to find more (e.g. SPEC.md, `--help`).
  - **Developers:** how to run tests, where code lives (e.g. root vs lib/), how to add tests, key scripts (test.sh, smoke-tests.sh, check.sh), and pointers to SPEC, IMPLEMENTATION, FOLLOWUP, SUMMARY.
- Add or update README sections if anything is missing. Then append to SUMMARY.md: what was verified, any fixes made, and that the follow-up plan is complete.

---

## Completion

After Commit 9 is done, the follow-up plan is complete. SUMMARY.md should note that build and all tests pass and that the README is complete for users and developers.
