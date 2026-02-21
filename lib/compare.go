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
type ProgressCounts struct {
	Enqueued          int32
	Processed         int32
	StartTimeUnixNano int64
	TotalPairs        int32
}

func comparePair(leftRoot, rightRoot, rel, hashAlg string, threshold int, cached *PairInfo) (different bool, reason string, hashStr string, size int64, mtime time.Time) {
	leftPath := filepath.Join(leftRoot, rel)
	rightPath := filepath.Join(rightRoot, rel)
	if cached == nil {
		panic("comparePair: missing file info (cached is nil); discovery walk must provide PairInfo for every pair")
	}
	leftSize := cached.LeftSize
	rightSize := cached.RightSize
	leftModTime := cached.LeftMtime
	rightModTime := cached.RightMtime
	if leftSize != rightSize {
		return true, "size changed", "", leftSize, leftModTime
	}
	if leftModTime.Equal(rightModTime) {
		return false, "", "", 0, time.Time{}
	}
	leftHash, err := hashFile(leftPath, hashAlg, threshold)
	if err != nil {
		return true, "hash left: " + err.Error(), "", leftSize, leftModTime
	}
	rightHash, err := hashFile(rightPath, hashAlg, threshold)
	if err != nil {
		return true, "hash right: " + err.Error(), "", leftSize, leftModTime
	}
	if leftHash == rightHash {
		return false, "", "", 0, time.Time{}
	}
	return true, "content differs", leftHash, leftSize, leftModTime
}

// RunWorkers starts numWorkers workers that read from pairCh, compare each pair, and send to resultCh.
func RunWorkers(leftRoot, rightRoot string, numWorkers int, hashAlg string, threshold int, pairCh <-chan PairJob, resultCh chan<- DiffResult, progress *ProgressCounts) {
	workCh := make(chan PairJob, numWorkers*2)
	var wg sync.WaitGroup
	for workerIdx := 0; workerIdx < numWorkers; workerIdx++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range workCh {
				diff, reason, hashStr, size, mtime := comparePair(leftRoot, rightRoot, job.Rel, hashAlg, threshold, job.Cached)
				if diff {
					resultCh <- DiffResult{Rel: job.Rel, Reason: reason, Hash: hashStr, Size: size, Mtime: mtime}
				}
				if progress != nil {
					atomic.AddInt32(&progress.Processed, 1)
				}
			}
		}()
	}
	go func() {
		for job := range pairCh {
			if progress != nil {
				atomic.AddInt32(&progress.Enqueued, 1)
				atomic.CompareAndSwapInt64(&progress.StartTimeUnixNano, 0, time.Now().UnixNano())
			}
			workCh <- job
		}
		close(workCh)
		wg.Wait()
		close(resultCh)
	}()
}
