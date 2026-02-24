# Implementation plan: phased restructure

This document lists commits in order to implement [PHASED-RESTRUCTURE-SPEC.md](PHASED-RESTRUCTURE-SPEC.md). Each commit is a single, buildable change. Work through the list from top to bottom.

**Execution:** We remove all workers and run the whole diff sequentially in one thread (no goroutine pool, no worker queue). The seven phases (walk-left, walk-right, build-pairs, classify-pairs, hash-left, hash-right, compare-hashes) run one after the other in a single flow so we can measure baseline speed before adding concurrency.

**Phase names:** walk-left, walk-right, build-pairs, classify-pairs, hash-left, hash-right, compare-hashes (in that order). The `--phase` flag accepts **the phase name only** (e.g. `--phase walk-right`); do not accept numeric arguments (1–7).

**For each commit:**

1. **Current commit** — Use the commit indicated by "Next:" below.
2. **TDD** — For each new function: write a **failing** unit test first, then implement until the test passes.
3. **Tests** — **Turn existing smoke tests off** during the refactor (Commits 2–7); **re-enable them only on the final commit** (Commit 8). **Only commit the final commit if all smoke tests pass.** With smoke tests off, you only need to worry about **unit tests** for the commit gate (build + unit tests must pass). **Feel free to delete** unit tests for the old path (discover, compare, etc.) that will no longer be used. **Create or update** unit tests as you go so that **all code—new and any that remains—is covered by tests.**
4. **Implement** — Do the work in that commit’s bullet list.
5. **Before committing** — Run `./check.sh`. For Commits 1–7, smoke tests are off so only build and unit tests must pass. For Commit 8 (final), re-enable smoke tests; **do not commit until all smoke tests pass.** Do not commit if `./check.sh` fails.
6. **Commit** — You may commit each change to Git yourself as you go, **provided `./check.sh` has passed**. Use a **short** line for the commit message (subject) and **more detailed** text for the commit description (body) that spells out the reason and intent for the change. The **Message:** line in each commit block is the short subject; the **Description:** paragraph is the detailed body.
7. **Update plan** — Set "Next:" to the following commit and mark the completed commit with `[x]`.

**Progress:** Set "Next:" to the commit you are about to do. Mark completed commits with `[x]`.

**Order:** The **--phase** argument is added **first** (Commit 2) so that each phase can be smoke-tested as soon as it is implemented. Commit 2 adds the flag and stub implementations for all seven phases; later commits replace stubs with real implementations one phase at a time.

**Smoke tests during the refactor:** **Turn the existing smoke tests off** for Commits 2–7 (e.g. skip them in `./check.sh` or the harness). **Re-enable them only on the final commit** (Commit 8). **Only commit the final commit if all smoke tests pass.** Do not modify or delete existing smoke test scripts; only skip running them until the end. New smoke tests for each phase can still run every commit if desired. **Because smoke tests are off, you can remove the old code path (discover + compare) during the refactor**—you do not have to keep it working until the final commit; remove it as soon as the phased pipeline is wired as the default (e.g. in the same commit that makes the phased pipeline the default, or earlier if convenient).

- **Next:** Commit 4

---

## - [x] Commit 1: Add FileInfo and phased pipeline types

**Message:** Add FileInfo and phased pipeline types

**Description:** Introduces the data structures required by the phased spec: FileInfo (relative path, size, mtime, hash) and types for pair classification (left-only, right-only, pairs, content-check queue). No pipeline behavior yet; this commit only adds the types and any helpers so later commits can use them. Aligns with PHASED-RESTRUCTURE-SPEC “Data structures”.

- Define **FileInfo** struct: Rel (string), Size (int64), Mtime (time.Time, normalized to second), Hash (string).
- Define **Pair** (or equivalent): relative path plus references to left and right FileInfo (e.g. indices or pointers).
- Define types for phase 3 output: left-only list, right-only list, and a collection of pairs (path + left info + right info). Use descriptive names (e.g. LeftOnlyPaths, RightOnlyPaths, Pairs).
- Define type for phase 4 output: pairs that differ by size, content-check queue (pairs needing hash), pairs same by size+mtime.
- Unit tests for any pure functions (e.g. mtime normalization if factored out). If FileInfo is just a struct, test that it can be constructed and used in a slice.

---

## - [x] Commit 2: Add --phase CLI flag and stub implementations for all phases

**Message:** Add --phase CLI flag and stub implementations for all phases

**Description:** Adds the **--phase** argument first so each phase can be smoke-tested as it is implemented. Accept the phase **by name only** (walk-left, walk-right, build-pairs, classify-pairs, hash-left, hash-right, compare-hashes); do not accept numbers (1–7). When set, run **only the phases needed** to produce that phase’s inputs (e.g. walk-left or walk-right runs only that phase; build-pairs runs walk-left, walk-right, build-pairs; etc.). **Report only the requested phase’s time:** start a timer immediately before that phase runs, stop it immediately after, and print that elapsed time to stderr (e.g. “walk-right: 0.012s”). Do not include earlier phases in the reported duration. Then exit. Provide **stub** implementations for all seven phases. When --phase is not set (or empty), keep using the existing discover+compare path. Document --phase in help and README (list the seven phase names). After this commit, smoke tests for each phase by name can run and pass (stubs run and report that phase’s duration only).

- Add **--phase** flag: accept **phase name only** (walk-left, walk-right, build-pairs, classify-pairs, hash-left, hash-right, compare-hashes). Unset or empty = run all / use current path. Do not accept numeric arguments (1–7). Validate and exit with usage if invalid.
- In runRoot: when --phase is set to a phase name, run **only the phases required** for that phase’s inputs (walk-left and walk-right need none; build-pairs needs walk-left, walk-right; classify-pairs needs the first three; hash-left and hash-right need the first four; compare-hashes needs all six before it). Use **stub** phase functions. **Time only the requested phase:** start a timer immediately before invoking it, stop it immediately after; print that elapsed time to stderr (e.g. “walk-right: 0.001s”). Then exit. Do not run diff output formatting when --phase is set (except optionally for compare-hashes).
- When --phase is unset or empty, use the existing discover + compare flow for now (smoke tests are off during the refactor, so the old path can be removed in a later commit once the phased pipeline is the default).
- Define stub functions for all seven phases (same signatures as real implementations; return empty/minimal data). Name them or map them to the phase names.
- Document --phase in help and README (list the seven phase names; phase is set by name only).
- **README:** Update the README to describe the **phases** and **how to invoke them individually by name**: e.g. `ffd --phase walk-right left right` runs only the phases needed for that phase’s inputs, then runs that phase and prints its duration to stderr (e.g. `walk-right: 0.012s`) and exits. No diff output on stdout when --phase is set. Set the phase by name only. Keep README in sync with actual behavior.
- **Smoke tests:** Add smoke tests for each phase **by name**: run `bin/ffd --phase walk-left <left> <right>`, `--phase walk-right`, …, `--phase compare-hashes` with test dirs; expect exit 0 and that phase’s duration on stderr. Register all seven in the smoke test harness.

---

## - [x] Commit 3: Implement walk-left and walk-right with shared implementation

**Message:** Implement walk-left and walk-right with shared implementation

**Description:** Replaces the walk-left and walk-right stubs with a **single shared** walk-and-collect implementation: one function (e.g. WalkTreeCollectFileInfo(root, batchSize) that walks the tree at root, uses DirEntry.Info() for each file to get size and mtime, and returns []FileInfo. walk-left calls it with the left root; walk-right calls it with the right root. Do not duplicate the walk logic. Do not modify walk.go; use ReadDir and entry.Info() in modifiable code (e.g. discover.go or a new file).

- Implement **shared** walk-and-collect function: takes root and directory batch size; walks the tree; for each **file** use **Info() on the DirEntry** (from ReadDir)—**do not call os.Stat**; normalize mtime to second; append FileInfo to a slice; return the slice.
- **walk-left** phase: call the shared function with the left root; return the slice as the left array.
- **walk-right** phase: call the same shared function with the right root; return the slice as the right array.
- TDD: write failing unit test (e.g. testdata with a few files; call the shared function; assert slice length and one entry has expected path, size, mtime). Then implement.
- Unit test: empty dir returns empty slice; dir with one or more files returns correct count and attributes. Test both phases (left root and right root) use the same behavior.
- **Smoke test:** Run with `--phase walk-left` and test dirs; expect exit 0 and walk-left duration on stderr. Run with `--phase walk-right`; expect exit 0 and walk-right duration on stderr.

---

## - [ ] Commit 4: Implement build-pairs — build pairs from left and right arrays

**Message:** Implement build-pairs — build pairs from left and right arrays

**Description:** Replaces the build-pairs stub with the real implementation: build a map keyed by relative path from the left and right FileInfo arrays, then derive left-only paths, right-only paths, and pairs (path + left FileInfo + right FileInfo).

- Replace **build-pairs** stub with real implementation: accepts left []FileInfo and right []FileInfo; builds map path → {left present, right present, left info, right info}; returns left-only list, right-only list, and slice of pairs. Use descriptive names for return types.
- TDD: failing unit test. Example: left has ["a","b"], right has ["b","c"] → left-only ["a"], right-only ["c"], pairs [("b", leftB, rightB)].
- Unit tests: both sides empty; one side empty; disjoint; all paired; mixed left-only, right-only, paired. Ensure path representation is not duplicated.
- **Smoke test:** Run with `--phase build-pairs` and test dirs; expect exit 0 and build-pairs duration on stderr.

---

## - [ ] Commit 5: Implement classify-pairs — classify pairs by size/mtime and build content-check queue

**Message:** Implement classify-pairs — classify pairs by size/mtime and build content-check queue

**Description:** Replaces the classify-pairs stub with the real implementation: for each pair, compare size and mtime; different size → differing (size changed); same size and same mtime → same; same size and different mtime → add to content-check queue.

- Replace **classify-pairs** stub with real implementation: accepts the pairs from build-pairs; for each pair compares left and right Size and Mtime; returns (differingBySize, contentCheckQueue, sameBySizeMtime). Mtime comparison uses normalized values (e.g. Equal).
- TDD: failing unit test. Example: pair with same size same mtime → sameBySizeMtime; different size → differingBySize; same size different mtime → contentCheckQueue.
- Unit tests: all same size/mtime; all different size; mix; content-check queue only contains same-size pairs.
- **Smoke test:** Run with `--phase classify-pairs` and test dirs; expect exit 0 and classify-pairs duration on stderr.

---

## - [ ] Commit 6: Implement hash-left and hash-right with shared implementation

**Message:** Implement hash-left and hash-right with shared implementation

**Description:** Replaces the hash-left and hash-right stubs with a **single shared** load-and-hash implementation: one function that, for each pair in the content-check queue, loads the file at (root + relative path), computes the hash via HashFile, and assigns it to the designated FileInfo (left or right). hash-left calls it with the left root and assigns to left FileInfo.Hash; hash-right calls it with the right root and assigns to right FileInfo.Hash. Do not duplicate the load-and-hash logic. Handle errors per spec (log, non-fatal count).

- Implement **shared** load-and-hash function: accepts root, content-check queue, hash algorithm, size threshold, and a way to assign the computed hash to the left or right FileInfo of each pair (e.g. callback or which side). For each pair, call HashFile(root, rel, …) and assign result to the appropriate FileInfo.Hash.
- **hash-left** phase: call the shared function with the left root and assign hashes to left file info.
- **hash-right** phase: call the same shared function with the right root and assign hashes to right file info.
- TDD: failing unit test with testdata (temp dir, minimal pair and content-check queue; call shared function for left then right; assert both left and right FileInfo.Hash set and match HashFile).
- Unit test: empty content-check queue leaves hashes empty; one or more pairs get hashes set for both sides when both phases are used.
- **Smoke test:** Run with `--phase hash-left` and test dirs; expect exit 0 and hash-left duration on stderr. Run with `--phase hash-right`; expect exit 0 and hash-right duration on stderr.

---

## - [ ] Commit 7: Implement compare-hashes — compare hashes and produce final diff result

**Message:** Implement compare-hashes — compare hashes and produce final diff result

**Description:** Replaces the compare-hashes stub with the real implementation: for each pair in the content-check queue, compare hashes; combine with classify-pairs results into a single list of DiffResults suitable for existing output formatting.

- Replace **compare-hashes** stub with real implementation: accepts content-check queue (with hashes filled), differing-by-size, left-only, right-only; for each content-check pair compares hashes and produces DiffResult when different; returns combined []DiffResult (and optionally same count) matching existing output shape (lib.DiffResult).
- TDD: failing unit test: content-check pair same hash → not in diff result; different hash → in diff result with reason "content differs". Combine with left-only, right-only, size-changed.
- Unit tests: all same hash; all different hash; mix; left-only and right-only included in output.
- **Smoke test:** Run with `--phase compare-hashes` and test dirs; expect exit 0 and diff output on stdout (and/or compare-hashes duration on stderr).

---

## - [ ] Commit 8: Use phased pipeline as default; remove old path; re-enable smoke tests

**Message:** Use phased pipeline as default; remove old path; re-enable smoke tests

**Description:** When --phase is not set, run all seven phases in sequence and use the phased pipeline output for diff results. **Remove the old discover + compare path** (no need to keep it; smoke tests were off during the refactor). Re-enable all existing smoke tests in `./check.sh` or the harness. **Do not commit this commit until all smoke tests pass.** Ensure progress and summary still work.

- In runRoot: when --phase is unset or empty, run walk-left, walk-right, build-pairs, classify-pairs, hash-left, hash-right, compare-hashes in order; collect DiffResults and feed existing output/formatting and summary. All diff results come from the seven-phase pipeline.
- **Remove** calls to Discover and Compare from the main diff flow (delete the old code path).
- Ensure progress and summary still work (e.g. total files from walk-left+walk-right or build-pairs; compared count from classify-pairs and compare-hashes). Update runRoot summary if needed.
- **Re-enable** all existing smoke tests (undo the skip added for Commits 2–7). Run `./check.sh`; fix any failing tests or smoke tests. **Only commit when all smoke tests pass.**

---

## Principles

- **Do not modify `lib/walk.go`.** The existing walk code is good; leave it unchanged. When implementing walk-left and walk-right, use **Info() on the DirEntry** (from ReadDir) to get file size and mtime; do not call os.Stat.
- **walk-left and walk-right use a shared implementation:** One function (e.g. WalkTreeCollectFileInfo(root, batchSize)) that both phases call with the appropriate root; do not duplicate the walk-and-collect logic.
- **hash-left and hash-right use a shared implementation:** One function that loads and hashes files for pairs in the content-check queue; both phases call it with the appropriate root and assign the hash to left or right FileInfo. Do not duplicate the load-and-hash logic.
- **You may change `lib/discover.go`, `main.go`, and `lib/compare.go`** as needed (e.g. replace discover/compare with phased pipeline, add phase orchestration in main, move or remove code in discover/compare).
- Use TDD: write a failing test for each new function, then implement.
- **Smoke tests:** Turn existing smoke tests off during the refactor; re-enable on the final commit. **Only commit the final commit if all smoke tests pass.** Do not modify or delete existing smoke test scripts. Because they are off, you can remove the old code path during the refactor. **Unit tests:** With smoke tests off, the commit gate is build + unit tests only. Delete unit tests for the old path that will no longer be used. Create or update unit tests as you go so all code (new and remaining) is covered by tests.
- One clear theme per commit; keep diffs reviewable.
- **README:** Update the README to document the phases (walk-left through compare-hashes) and how to invoke them individually **by name** (--phase &lt;name&gt;); keep it in sync with CLI behavior. The phase is set by name only, not by number.
- Before each commit, run `./check.sh`; do not commit if it fails. You may commit each change yourself once `./check.sh` passes. Use a short commit message (subject) and a detailed description (body) for reason and intent.
