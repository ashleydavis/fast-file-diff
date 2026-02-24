# Phased restructure spec: sequential directory comparison

This document specifies a restructure of the directory comparison pipeline so that left and right trees are processed in **multiple sequential phases** instead of interleaved discovery and comparison. **We get rid of all workers** and do the whole diff **sequentially in one thread**: phases run one after the other, with no goroutine pool, no worker queue, and no interleaving. The goal is to establish a clear phased pipeline and measure how fast it runs as a baseline before adding concurrency later.

The restructure applies to the core diff logic (comparing two directory trees). CLI, logging, output formats, and other behavior remain as in [SPEC.md](SPEC.md). This spec only describes the comparison pipeline. The commit-by-commit implementation plan is in [PHASED-IMPLEMENTATION.md](PHASED-IMPLEMENTATION.md).

---

## Execution model

- **Sequential phases:** The seven phases below run in order; each phase has a **name** (see table). Phase N+1 starts only after phase N has completed.
- **No workers; one thread:** All workers (e.g. discover workers, compare workers) are removed. The entire diff runs sequentially in one thread: no goroutine pool, no worker queue, no concurrent phase. Each phase does its work in a single flow (e.g. one loop over the data). The intent is to measure baseline speed without concurrency.

- **Runnable phases:** Each phase must be runnable **separately** so that the time for **that one phase only** can be measured. The phase is selected via a **CLI argument** by **name only**: `--phase <name>` where &lt;name&gt; is one of `walk-left`, `walk-right`, `build-pairs`, `classify-pairs`, `hash-left`, `hash-right`, `compare-hashes`. Do not accept numeric phase arguments (e.g. 1–7); the user sets the phase by name. When set, the program runs **only the phases needed** to produce that phase’s inputs, then **reports only that phase’s duration** (timer starts just before the phase, stops just after). When the argument is not set, all phases run in order as normal.

**Phase names (in order):**

| # | Name            | Description |
|---|-----------------|-------------|
| 1 | **walk-left**   | Walk left tree; collect file info (path, size, mtime) into left array. |
| 2 | **walk-right**  | Walk right tree; collect file info into right array. |
| 3 | **build-pairs** | Build map from path → left/right; derive left-only, right-only, pairs. |
| 4 | **classify-pairs** | For each pair: compare size/mtime; differing by size, content-check queue, or same. |
| 5 | **hash-left**   | For each pair in content-check queue: hash left file, store in left file info. |
| 6 | **hash-right**  | For each pair in content-check queue: hash right file, store in right file info. |
| 7 | **compare-hashes** | Compare hashes for content-check pairs; produce final diff result set. |

---

## Data structures

- **File info (per file):** For each discovered file we store:
  - **Relative path** (string)
  - **Size** (int64)
  - **Modification time** (mtime, normalized e.g. to seconds for cross-filesystem consistency)
  - **Hash** (string, hex; initially empty, filled in hash-left and hash-right only for files that need content comparison)

- **Left/right arrays:** Each is an array (or slice) of file-info records: one entry per regular file discovered on that side.

- **Pair:** A pair is a relative path that exists on both sides. We need to know:
  - The relative path
  - Pointer or index to left file info and right file info (or embed path and both infos)

- **Pair classification:** After building pairs we have:
  - **Left-only:** paths that appear only in the left array
  - **Right-only:** paths that appear only in the right array
  - **Paired:** paths that appear in both arrays (each pair has left info + right info)

- **Content-check queue:** A list (or array) of pairs that need content comparison: same path on both sides but size or mtime differs, so we must compare by hash.

---

**Shared implementation:** walk-left and walk-right **share a single implementation**: one function (e.g. walk a tree at a given root, collect file info via DirEntry.Info(), return []FileInfo). walk-left calls it with the left root; walk-right calls it with the right root. Do not duplicate the walk-and-collect logic.

---

## Phase 1: walk-left — Walk left tree and collect file info

1. Walk the **left** directory tree recursively (same rules as current: regular files only, include hidden/dotfiles, no symlink following). Use the **shared** walk-and-collect implementation with the left root.
2. For each discovered **file** (not directory), obtain size and mtime by calling **Info() on the DirEntry** (the entry returned from ReadDir), not by calling Stat on the path. Use the DirEntry’s Info() to get file size and modification time.
3. Store for each file: **relative path**, **size**, **mtime**. Append to a **left array** (e.g. `[]FileInfo`). Do not compute hash yet.
4. When the walk finishes, the left array holds one entry per regular file in the left tree.

**Inputs:** Left root path, directory batch size (for ReadDir batching).  
**Output:** Left array of file info (path, size, mtime; hash empty).

---

## Phase 2: walk-right — Walk right tree and collect file info

1. Walk the **right** directory tree recursively (same rules as left). Use the **same shared** walk-and-collect implementation with the right root.
2. For each discovered file, obtain size and mtime by calling **Info() on the DirEntry** (from ReadDir), not Stat.
3. Store for each file: **relative path**, **size**, **mtime**. Append to a **right array**.
4. When the walk finishes, the right array holds one entry per regular file in the right tree.

**Inputs:** Right root path, directory batch size.  
**Output:** Right array of file info (path, size, mtime; hash empty).

---

## Phase 3: build-pairs — Build pairs from left and right arrays

1. Build a **map** keyed by relative path:
   - For each entry in the left array, record that this path exists on the left (and keep a reference to its file info).
   - For each entry in the right array, record that this path exists on the right (and keep a reference to its file info).
2. From this map derive:
   - **Left-only:** paths that appear only on the left.
   - **Right-only:** paths that appear only on the right.
   - **Pairs:** paths that appear on both sides; each pair has one left file info and one right file info.

**Inputs:** Left array, right array.  
**Output:** Left-only list, right-only list, and a collection of pairs (each pair = path + left info + right info).

---

## Phase 4: classify-pairs — Decide which pairs need content comparison

1. For **each pair** (path exists on both sides), compare the two file infos:
   - If **size** differs → treat as different (no content check needed). Record as differing and do **not** add to the content-check queue.
   - If **mtime** differs (and size is equal) → add this pair to the **content-check queue** (we will hash both files).
   - If **size and mtime** are both equal → treat as same; do not add to the content-check queue.
2. The content-check queue is therefore the set of pairs where size is the same but mtime differs (or where we choose to confirm by hash). Pairs with different size are already decided as "different" and need no hash.

**Inputs:** The list of pairs from build-pairs.  
**Output:**  
- List of pairs that are already known to differ (e.g. size changed).  
- **Content-check queue:** list of pairs that need content comparison (same size, mtime may differ).  
- List of pairs that are same by size+mtime (no further work).

---

**Shared implementation:** hash-left and hash-right **share a single implementation**: one function that, for each pair in the content-check queue, loads the file at (root + relative path), computes the selected hash, and stores it in the designated FileInfo (left or right). hash-left calls it with the left root and assigns the hash to the left file info; hash-right calls it with the right root and assigns to the right file info. Do not duplicate the load-and-hash logic.

---

## Phase 5: hash-left — Hash each left file that is part of a content-check pair

1. For **each pair** in the content-check queue, use the **shared** load-and-hash implementation with the **left** root; store the resulting hash in the **left file info** for that path.
2. Process files sequentially; no workers.

**Inputs:** Left root, content-check queue, hash algorithm, size threshold (read-full vs stream).  
**Output:** Left file infos for content-check pairs now have the hash field set.

---

## Phase 6: hash-right — Hash each right file that is part of a content-check pair

1. For **each pair** in the content-check queue, use the **same shared** load-and-hash implementation with the **right** root; store the resulting hash in the **right file info** for that path.
2. Process files sequentially; no workers.

**Inputs:** Right root, content-check queue, hash algorithm, size threshold.  
**Output:** Right file infos for content-check pairs now have the hash field set.

---

## Phase 7: compare-hashes — Compare hashes and finalize diff result

1. For **each pair** in the content-check queue, compare the **left hash** and **right hash** (both now set from hash-left and hash-right).
2. If hashes are equal → file pair is **same** (content identical).
3. If hashes differ → file pair is **different** (content differs); record for output with reason "content differs" and the two hashes.
4. Combine with results from classify-pairs:
   - Pairs already marked different (e.g. size changed) are differing.
   - Left-only and right-only from build-pairs are differing (with "left only" / "right only").
   - Pairs same by size+mtime are same.
   - Pairs in content-check queue are same or different based on hash comparison here.

**Inputs:** Content-check queue (with hashes filled in), and the "already different" list from classify-pairs.  
**Output:** Complete list of differing items (left-only, right-only, size changed, content differs) and same items, ready for output formatting (text, table, json, yaml) and progress/summary.

---

## Summary of phases (order)

| # | Name | Description |
|---|------|-------------|
| 1 | walk-left | Walk left tree; for each file use DirEntry.Info() (not Stat) for size, mtime; collect into left array. |
| 2 | walk-right | Walk right tree; for each file use DirEntry.Info() (not Stat) for size, mtime; collect into right array. |
| 3 | build-pairs | Build map from path → left/right; derive left-only, right-only, pairs. |
| 4 | classify-pairs | For each pair: compare size/mtime; if different size → differing; if same size, different mtime → add to content-check queue; if same size and mtime → same. |
| 5 | hash-left | For each pair in content-check queue: hash left file and store hash in left file info. |
| 6 | hash-right | For each pair in content-check queue: hash right file and store hash in right file info. |
| 7 | compare-hashes | For each pair in content-check queue: compare hashes; same or "content differs". Produce final diff result set. |

---

## Relationship to existing spec

- **User-visible behavior** (output format, reasons, per-file details, progress, logging, CLI, exit codes) stays as in [SPEC.md](SPEC.md). Only the **internal pipeline** (how we walk, collect info, build pairs, and compare) changes to this sequential phased model.
- **Info from DirEntry:** When walking, use the **Info() method on the fs.DirEntry** returned by ReadDir to get file size and mtime (mtime normalized to seconds). Do not call os.Stat on the path; using DirEntry.Info() avoids an extra syscall per file.
- **Hash algorithm and threshold:** Unchanged (e.g. `--hash`, `--threshold`); hash-left and hash-right use the same hash and read-full vs stream rules as the current implementation.
- **Directory batching:** walk-left and walk-right can still use batched directory reads (e.g. `ReadDir(batchSize)`) and the existing walk logic; only the **ordering** (left walk fully, then right walk fully) and the **collection** (into arrays with size/mtime) are specified here.

---

## Out of scope for this spec

- **Concurrency:** This spec does not require workers or goroutines. A later change may introduce workers (e.g. in hash-left and hash-right) once the sequential baseline is measured.
- **Interleaving:** We do not interleave left and right walks or interleave hashing with pair building; phases are strictly sequential.
- **Progress:** How progress is reported (e.g. "phase 1 done", "phase 2 done", or file counts) can follow existing progress rules in SPEC.md; this doc does not change the progress contract.

---

## Implementation notes

- **Path representation:** Keep a single representation for relative paths (e.g. one string per path, shared between left/right structures) to avoid duplication, per existing memory strategy.
- **Reuse:** Existing packages (e.g. `lib/walk`, `lib/hash`) can be reused; the main change is the **orchestration** in the diff command: call walk twice (left then right), collect into arrays, then run build-pairs, classify-pairs, hash-left, hash-right, compare-hashes in order.
- **Do not modify `lib/walk.go`:** The existing walk implementation is good and must not be changed. walk-left and walk-right should call it as-is (e.g. `WalkTree` or `walkTreeBatched`) and collect file info in the caller; do not add new entry points or change behavior inside `walk.go`.
- **Shared implementation for walk-left and walk-right:** The first two phases must use a **single shared** walk-and-collect function (e.g. one that takes root and batch size, returns []FileInfo). walk-left invokes it with the left root; walk-right invokes it with the right root. Do not implement two separate walk-and-collect code paths.
- **Shared implementation for hash-left and hash-right:** The two phases that load and hash files must use a **single shared** load-and-hash function (e.g. one that takes root, content-check queue, hash algorithm, threshold, and a way to assign the computed hash to the left or right FileInfo of each pair). hash-left invokes it with the left root and assigns to left file info; hash-right invokes it with the right root and assigns to right file info. Do not duplicate the load-and-hash logic.
- **Modifiable files:** `lib/discover.go`, `main.go`, and `lib/compare.go` may be changed as needed to implement the phased pipeline (e.g. replace or remove discover/compare logic, wire phases in main, add new phase code in lib).
- **Running a phase via CLI:** When the user sets the phase via the CLI argument **by name** (e.g. `--phase walk-right`), the program runs that phase so its duration can be measured. Run **only the phases required** to produce that phase’s inputs: walk-left and walk-right need no prior phases; build-pairs needs walk-left and walk-right; classify-pairs needs walk-left, walk-right, build-pairs; hash-left and hash-right need the first four; compare-hashes needs all six before it. **Only the requested phase’s time is reported:** start a timer immediately before that phase runs, stop it immediately after it completes, and print that elapsed time to stderr (e.g. `walk-right: 0.012s`). Earlier phases (if any) run only to build inputs; their time is **not** included in the reported duration.
- **Testing:** **Unit tests:** With smoke tests off during the refactor, the commit gate is build + unit tests only. Feel free to **delete** unit tests for the old path (discover, compare, etc.) that will no longer be used. **Create or update** unit tests as you go so that **all code—new and any that remains—is covered by tests.** Unit tests should cover the phased pipeline: build-pairs, classify-pairs, hash-left, hash-right, compare-hashes, and the shared walk and hash implementations. **Smoke tests:** Turn existing smoke tests off during the refactor; re-enable only on the final commit. Only commit the final commit if all smoke tests pass. Do not modify or delete existing smoke test scripts. Because smoke tests are off, the old code path can be removed during the refactor.
- **README:** Update the README to describe the **phases** (walk-left, walk-right, build-pairs, classify-pairs, hash-left, hash-right, compare-hashes) and **how to invoke them individually by name** (e.g. `--phase walk-right` to run only that phase and print its duration to stderr). The phase is set by name only. Keep the README in sync with actual CLI behavior per SPEC.md.
