package lib

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// DiffResult describes one differing file for output.
type DiffResult struct {
	Rel      string
	Reason   string
	Hash     string
	Size     int64
	Mtime    time.Time
	LeftOnly bool
}

// ProgressCounts holds counters and start time for the progress indicator.
// Exported fields so main can use atomic load for progress display.
// TotalPairs is the total number of pairs to compare (set before starting workers); 0 means unknown.
// If WorkerProcessed is non-nil and len(WorkerProcessed) >= numWorkers, each worker atomically increments WorkerProcessed[workerIdx] so you can see per-worker utilization (e.g. min/max in summary).
type ProgressCounts struct {
	Enqueued          int32
	Processed         int32
	StartTimeUnixNano int64
	TotalPairs        int32
	WorkerProcessed   []int32 // optional: per-worker compare count, one per worker
}

// comparePair hashes both files and compares hashes. Caller must have already checked size and mtime
// (same size, different mtime); only such pairs should be sent to workers.
func comparePair(leftRoot, rightRoot, relativePath string, hashAlg string, threshold int) (different bool, reason string, hashStr string) {
	leftPath := filepath.Join(leftRoot, relativePath)
	rightPath := filepath.Join(rightRoot, relativePath)
	leftHash, err := hashFile(leftPath, hashAlg, threshold)
	if err != nil {
		return true, "hash left: " + err.Error(), ""
	}
	rightHash, err := hashFile(rightPath, hashAlg, threshold)
	if err != nil {
		return true, "hash right: " + err.Error(), ""
	}
	if leftHash == rightHash {
		return false, "", ""
	}
	return true, "content differs", leftHash
}

// RunWorkers starts numWorkers workers that read from pairCh, compare each pair, and send to resultCh.
// progress and workerUtilization must be non-nil; workers update progress and call workerUtilization.Poke(workerIdx) after each pair.
func RunWorkers(leftRoot, rightRoot string, numWorkers int, hashAlg string, threshold int, pairCh <-chan PairJob, resultCh chan<- DiffResult, progress *ProgressCounts, workerUtilization *WorkerUtilization) {
	workCh := make(chan PairJob, numWorkers*2)
	var wg sync.WaitGroup
	for workerIdx := 0; workerIdx < numWorkers; workerIdx++ {
		idx := workerIdx
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range workCh {
				diff, reason, hashStr := comparePair(leftRoot, rightRoot, job.Rel, hashAlg, threshold)
				if diff {
					resultCh <- DiffResult{Rel: job.Rel, Reason: reason, Hash: hashStr, Size: job.Cached.LeftSize, Mtime: job.Cached.LeftMtime}
				}
				atomic.AddInt32(&progress.Processed, 1)
				if idx < len(progress.WorkerProcessed) {
					atomic.AddInt32(&progress.WorkerProcessed[idx], 1)
				}
				workerUtilization.Poke(idx)
			}
		}()
	}
	go func() {
		for job := range pairCh {
			atomic.AddInt32(&progress.Enqueued, 1)
			atomic.CompareAndSwapInt64(&progress.StartTimeUnixNano, 0, time.Now().UnixNano())
			workCh <- job
		}
		close(workCh)
		wg.Wait()
		close(resultCh)
	}()
}
