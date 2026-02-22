# Lock usage audit

This document lists every place locks (mutexes) are used, why they are used, and what would happen if the lock were removed.

---

## 1. lib/discover.go – `DiscoveredSet.mu`

**Type:** `sync.Mutex` on `DiscoveredSet`.

**Used in:**

| Location        | Method            | Why the lock is used |
|----------------|-------------------|-----------------------|
| Add()          | Lock at start, defer Unlock | Protects concurrent updates to `left`, `right`, `leftFileInfo`, `rightFileInfo`, `pairPaths`, and the two count fields. Many walk workers call Add() from different goroutines. |
| PairCachedInfo() | Lock, read maps, Unlock | Reads `leftFileInfo` and `rightFileInfo`. Must be consistent with Add(); map reads are not safe concurrent with map writes. |
| PairsCount()   | Lock, read len(pairPaths), Unlock | Reads slice length. Slice header is updated in Add() (append); reading without lock is a data race. |
| LeftOnlyCount()  | Lock, read leftOnlyCount, Unlock | Reads a counter that Add() writes. Without sync (lock or atomic), read-vs-write is a data race. |
| RightOnlyCount() | Lock, read rightOnlyCount, Unlock | Same as LeftOnlyCount(). |
| PairPaths()    | Lock, copy pairPaths, Unlock | Copies the slice; must not race with append in Add(). |
| LeftOnlyPaths()  | Lock, iterate left/right maps, Unlock | Iterates maps that Add() mutates; iteration concurrent with write can panic or corrupt state. |
| RightOnlyPaths() | Lock, iterate right/left maps, Unlock | Same as LeftOnlyPaths(). |

**Repercussions of removing the lock:**

- **Add():** Without the lock, concurrent Add() would cause concurrent map writes and slice append. Result: **runtime panic** (concurrent map write) or **memory corruption** (slice append race). **Lock is required.**
- **PairCachedInfo(), PairPaths(), LeftOnlyPaths(), RightOnlyPaths():** Concurrent read of maps/slice while Add() writes: **panic or undefined behavior**. **Lock (or another safe protocol) is required.**
- **PairsCount(), LeftOnlyCount(), RightOnlyCount():** If you only remove the lock from these getters (and keep it in Add()):
  - Reading the counters or len(pairPaths) without synchronization is a **data race** (undefined behavior; `go test -race` will report it).
  - You might see stale or wrong counts in the progress line.
  - **Mitigation:** Keep the lock, or make the two counts atomic (e.g. `int32` + `atomic.AddInt32` / `atomic.LoadInt32`) and keep the lock only where you read/write `pairPaths` and the maps. Then the count getters can be lock-free.

---

## 2. lib/logger.go – `Logger.mu`

**Type:** `sync.Mutex` on `Logger`.

**Used in:**

| Location        | Method            | Why the lock is used |
|----------------|-------------------|-----------------------|
| Log()          | Lock, write mainFile, Sync(), Unlock | Multiple goroutines (walk/compare workers) can call Log() at once. Prevents interleaved lines and ensures each line + Sync is atomic. |
| LogError()     | Lock, increment nonFatal, write both files, Unlock | Same as Log(); also protects nonFatal read-modify-write. |
| Fatal()        | Lock, write both files, Unlock, then stderr + Exit | Ensures log writes complete before process exits; protects file writes. |
| PrintLogPaths()| Lock, read mainPath/errorPath, Unlock | Protects read of path strings (written at creation; could be observed during Close). |
| NonFatalCount()| Lock, read nonFatal, Unlock | nonFatal is written in LogError(); read without sync is a data race. |
| Close()        | Lock, close files, set nil, Unlock | Prevents races with in-flight Log/LogError and ensures no use-after-close. |

**Repercussions of removing the lock:**

- **Log() / LogError():** Concurrent writes to the same file from multiple goroutines can **interleave output** (e.g. half of one line, then half of another). Sync() and other file operations are not atomic across goroutines. **Lock is required** for correct, readable logs unless you switch to a single writer goroutine (e.g. channel).
- **NonFatalCount():** Data race on read; count could be wrong. **Lock or atomic required.**
- **Close():** Without lock, a goroutine could Log() after the file is closed (use-after-close or write to closed file). **Lock (or a strict close protocol) is required.**

---

## 3. lib/path.go – `PathPool.mu`

**Type:** `sync.Mutex` on `PathPool`.

**Used in:**

| Location | Method  | Why the lock is used |
|----------|--------|-----------------------|
| Intern() | Lock, map lookup/insert, Unlock | Reads and possibly writes `seen` map. Called from DiscoveredSet.Add() from many walk workers. Map must not be written concurrently. |

**Repercussions of removing the lock:**

- **Intern():** Concurrent map read and write (or multiple writes) in Go causes **panic: concurrent map writes** (or concurrent map read and map write). **Lock is required** for the current map-based design. To reduce contention you could use a concurrent structure (e.g. `sync.Map`) instead of a plain map + mutex, but some form of synchronization is still required.

---

## Summary

| Package / type     | Lock required? | Notes |
|--------------------|----------------|--------|
| DiscoveredSet.Add() and all map/slice access | **Yes** | Concurrency is fundamental; removing lock causes panic or corruption. Count getters could use atomics to avoid taking the lock. |
| Logger (Log, LogError, Close, etc.)         | **Yes**        | Needed for correct, non-interleaved log output and safe close. |
| PathPool.Intern()                            | **Yes**        | Map is shared across goroutines; no lock ⇒ concurrent map panic. |

All current locks protect real shared state used from multiple goroutines. Removing them without a replacement (atomics, concurrent data structures, or a different design) leads to data races, panics, or wrong results.
