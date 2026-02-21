# Implementation summary

This document is updated after each commit: what was done, how it went, and any problems. After all commits are complete, it is updated to describe how the completed work aligns with [SPEC.md](SPEC.md).

---

## Per commit

*(For each commit, add a short entry: what you did, how it went, problems if any.)*

### Commit 1: Add Cobra and root command
- **What I did:** Added Cobra dependency; root command with two positional args (left, right); no args prints help to stdout and exits 0; `ensureDir` validates both paths (exit 2 on invalid); exit code constants 0/1/2/3; smoke harness and `smoke-help.sh` (no args → help).
- **How it went:** Build, unit tests, and smoke test passed. TDD: wrote failing tests for `ensureDir` then implemented.
- **Problems:** None.

### Commit 2: Add logging layer (Logger)
- **What I did:** Added Logger struct (temp dir, main + error log files with date/sequence names), Log/LogError/Fatal/PrintLogPaths/NonFatalCount/Close; isTTY to skip PrintLogPaths when not TTY; created Logger after validation in root and call PrintLogPaths before exit; smoke test two-empty-dirs.
- **How it went:** Unit tests for NewLogger, Log, LogError/NonFatalCount; build and smoke tests passed.
- **Problems:** None.

### Commit 3: Path representation and safety
- **What I did:** Added path.go: resolvePath(root, rel) with pathUnder to reject escapes; pathPool with Intern() for deduping relative path strings. Unit tests for resolvePath (under root, empty, rejects ..) and pathPool.Intern.
- **How it went:** All tests passed. No smoke test required for this commit.
- **Problems:** None.

### Commit 4: Discovered-files set and pair formation
- **What I did:** Added discoveredSet (map left/right by rel path, pathPool for interning); Add(rel, side) returns true when pair forms. Unit tests for pair formation, left-only, right-only, multiple pairs. Smoke test invalid-dir (exit 2, clear error).
- **How it went:** Build, tests, smoke passed.
- **Problems:** None.

### Commit 5: Walk one tree and log entries
- **What I did:** Added walk.go: walkTree(root, fn) with filepath.WalkDir; regular files and dirs only, include dotfiles; callback(rel, isDir). Unit tests for path collection and regular-files-only. Wired walk of left tree in main with logger.Log for each dir/file.
- **How it went:** All tests and smoke passed.
- **Problems:** None.

### Commit 6: Walk both trees in parallel and feed queue
- **What I did:** Added walkBothTrees (two goroutines, WaitGroup, pairCh); each walk logs dir/file and for files calls set.Add(rel, side), sending rel to pairCh when pair forms; main drains pairCh and logs pairs. Smoke test identical-dirs (same files, exit 0, no stdout).
- **How it went:** Build, tests, smoke passed.
- **Problems:** None.

### Commit 7: Linux batched directory read (optional)
- **What I did:** Added --dir-batch-size flag (default 4096); walkTreeWithBatch with build-tagged walk_linux.go (Readdir(n) batched walk) and walk_nonlinux.go (portable); walkBothTrees passes batch size.
- **How it went:** Build, tests, smoke passed on Linux.
- **Problems:** None.

### Commit 8: Worker pool and queue processing
- **What I did:** Added --workers flag (default NumCPU); compare.go with comparePair (size+mtime), runWorkers (N workers from pairCh, resultCh), progressCounts; progress loop on stderr when TTY; main runs walkBothTrees, runWorkers, collects DiffResults and prints to stdout; smoke one-diff; unit tests for comparePair.
- **How it went:** Build, tests, smoke passed.
- **Problems:** None.

### Commit 9: Size and mtime comparison
- **What I did:** comparePair already did size+mtime; added unit test same size/different mtime; smoke same-size-diff-mtime (mtime differs → diff reported).
- **How it went:** All tests and smoke passed.
- **Problems:** None.

### Commit 10: Hash selection and content comparison
- **What I did:** Added hash.go (xxhash, sha256, md5; hashFile with threshold, sync.Pool); --hash and --threshold flags; comparePair hashes when same size + different mtime, reports \"content differs\" + hash; DiffResult.Hash; smoke hash-xxhash; unit tests for hash and compare.
- **How it went:** All tests and smoke passed.
- **Problems:** None.

### Commit 11: Output format text (ASCII tree)
- **What I did:** Extended DiffResult with Size, Mtime; formatTextTree in output.go (case-sensitive sort, ASCII tree, per-file size/mtime/reason/hash); --format flag (text default); smoke format-text.
- **How it went:** All tests and smoke passed.
- **Problems:** None.

### Commit 12: Output format table
- **What I did:** formatTable (tab-separated columns path, size, mtime, reason, hash); --format table; smoke format-table.
- **How it went:** Passed.
- **Problems:** None.

### Commit 13: Output format JSON
- **What I did:** formatJSON (array of objects); --format json; smoke format-json (python3 json.load).
- **How it went:** Passed.
- **Problems:** None.

### Commit 14: Output format YAML
- **What I did:** formatYAML (gopkg.in/yaml.v3); --format yaml; smoke format-yaml.
- **How it went:** Passed.
- **Problems:** None.

*(Continue for Commits 15–19; add entries as you complete each commit.)*

---

## Alignment with SPEC

*(After all 19 commits are done, update this section to summarize how the completed work aligns with SPEC.md: which spec sections are satisfied, and any gaps or deviations.)*
