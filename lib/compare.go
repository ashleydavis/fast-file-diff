package lib

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// DiffResult describes one differing file for output. Left/Right fields hold both sides when comparing a pair.
type DiffResult struct {
	Rel        string
	Reason     string
	LeftHash   string
	RightHash  string
	LeftSize   int64
	RightSize  int64
	LeftMtime  time.Time
	RightMtime time.Time
	LeftOnly   bool
}

// CompareResult is the outcome of comparing one pair: identical (Diff == nil) or different (Diff != nil). When identical, Reason explains why.
// Workers send one per pair to a single result channel.
type CompareResult struct {
	RelativePath string      // relative path
	Diff         *DiffResult // nil if identical, non-nil if different
	Reason       string      // when identical, reason (e.g. "same size and mtime", "same hash")
}

// PairJob is a single pair to compare (relative path); compare phase stats both files.
type PairJob struct {
	Rel string
}

// comparePair stats both files. If size differs, reports "size changed" without hashing. If size same and mtime same, reports identical without hashing. Otherwise (same size, different mtime) hashes and compares. Returns different, reason, and left/right hash, size, mtime for DiffResult.
func comparePair(leftRoot, rightRoot, relativePath string, hashAlg string, threshold int) (different bool, reason string, leftHash, rightHash string, leftSize, rightSize int64, leftMtime, rightMtime time.Time) {
	leftPath := filepath.Join(leftRoot, relativePath)
	rightPath := filepath.Join(rightRoot, relativePath)
	leftInfo, err := os.Stat(leftPath)
	if err != nil {
		return true, "stat left: " + err.Error(), "", "", 0, 0, time.Time{}, time.Time{}
	}
	rightInfo, err := os.Stat(rightPath)
	if err != nil {
		return true, "stat right: " + err.Error(), "", "", leftInfo.Size(), 0, leftInfo.ModTime().Truncate(time.Second), time.Time{}
	}
	if !leftInfo.Mode().IsRegular() || !rightInfo.Mode().IsRegular() {
		return false, "", "", "", 0, 0, time.Time{}, time.Time{}
	}
	leftSize, leftMtime = leftInfo.Size(), leftInfo.ModTime().Truncate(time.Second)
	rightSize, rightMtime = rightInfo.Size(), rightInfo.ModTime().Truncate(time.Second)
	if leftSize != rightSize {
		return true, "size changed", "", "", leftSize, rightSize, leftMtime, rightMtime
	}
	if leftMtime.Equal(rightMtime) {
		return false, "same size and mtime", "", "", 0, 0, time.Time{}, time.Time{}
	}
	leftHash, err = hashFile(leftPath, hashAlg, threshold)
	if err != nil {
		return true, "hash left: " + err.Error(), "", "", leftSize, rightSize, leftMtime, rightMtime
	}
	rightHash, err = hashFile(rightPath, hashAlg, threshold)
	if err != nil {
		return true, "hash right: " + err.Error(), "", "", leftSize, rightSize, leftMtime, rightMtime
	}
	if leftHash == rightHash {
		return false, "same hash", "", "", 0, 0, time.Time{}, time.Time{}
	}
	return true, "content differs", leftHash, rightHash, leftSize, rightSize, leftMtime, rightMtime
}

// Compare runs the comparison phase: numWorkers workers compare each path in pairPaths (size/mtime/hash as needed) and send one CompareResult per pair to resultCh (Diff set when different, nil when identical).
// progress and workerUtilization must be non-nil. Compare sets progress.TotalPairs and progress.WorkerProcessed, feeds pairs internally, and closes resultCh when done.
func Compare(leftRoot, rightRoot string, pairPaths []string, numWorkers int, hashAlg string, threshold int, resultCh chan<- CompareResult, progress *ProgressCounts, workerUtilization *WorkerUtilization) {
	progress.TotalPairs = int32(len(pairPaths))
	if len(pairPaths) > 0 {
		progress.WorkerProcessed = make([]int32, numWorkers)
	}
	rec := NewProgressRecorder(progress, workerUtilization)
	workCh := make(chan PairJob, numWorkers*2)
	var wg sync.WaitGroup
	for workerIdx := 0; workerIdx < numWorkers; workerIdx++ {
		idx := workerIdx
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range workCh {
				diff, reason, lHash, rHash, lSize, rSize, lMtime, rMtime := comparePair(leftRoot, rightRoot, job.Rel, hashAlg, threshold)
				if diff {
					resultCh <- CompareResult{RelativePath: job.Rel, Diff: &DiffResult{Rel: job.Rel, Reason: reason, LeftHash: lHash, RightHash: rHash, LeftSize: lSize, RightSize: rSize, LeftMtime: lMtime, RightMtime: rMtime}}
				} else {
					resultCh <- CompareResult{RelativePath: job.Rel, Diff: nil, Reason: reason}
				}
				rec.RecordCompletion(idx)
			}
		}()
	}
	pairCh := make(chan PairJob, len(pairPaths)+1)
	go func() {
		for _, rel := range pairPaths {
			atomic.CompareAndSwapInt64(&progress.StartTimeUnixNano, 0, time.Now().UnixNano())
			atomic.AddInt32(&progress.Enqueued, 1)
			pairCh <- PairJob{Rel: rel}
		}
		close(pairCh)
	}()
	go func() {
		for job := range pairCh {
			workCh <- job
		}
		close(workCh)
		wg.Wait()
		close(resultCh)
	}()
}
