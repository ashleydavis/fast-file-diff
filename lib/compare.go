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

// CompareResult is the outcome of comparing one pair: identical (Diff == nil) or different (Diff != nil).
// Workers send one per pair to a single result channel.
type CompareResult struct {
	RelativePath string      // relative path
	Diff         *DiffResult // nil if identical, non-nil if different
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

// RunWorkers starts numWorkers workers that read from pairCh, compare each pair, and send one CompareResult per pair to resultCh (Diff set when different, nil when identical).
// progress and workerUtilization must be non-nil. RunWorkers closes resultCh when done.
func RunWorkers(leftRoot, rightRoot string, numWorkers int, hashAlg string, threshold int, pairCh <-chan PairJob, resultCh chan<- CompareResult, progress *ProgressCounts, workerUtilization *WorkerUtilization) {
	rec := NewProgressRecorder(progress, workerUtilization)
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
					resultCh <- CompareResult{RelativePath: job.Rel, Diff: &DiffResult{Rel: job.Rel, Reason: reason, Hash: hashStr, Size: job.Cached.LeftSize, Mtime: job.Cached.LeftMtime}}
				} else {
					resultCh <- CompareResult{RelativePath: job.Rel, Diff: nil}
				}
				rec.RecordCompletion(idx)
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
