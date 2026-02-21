package main

import (
	"os"
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

// comparePair stats both files; returns (true, reason, hash, size, mtime) if different, (false, ...) if same.
func comparePair(leftRoot, rightRoot, rel, hashAlg string, threshold int) (different bool, reason string, hashStr string, size int64, mtime time.Time) {
	leftPath := filepath.Join(leftRoot, rel)
	rightPath := filepath.Join(rightRoot, rel)
	leftInfo, err := os.Stat(leftPath)
	if err != nil {
		return true, "stat left: " + err.Error(), "", 0, time.Time{}
	}
	rightInfo, err := os.Stat(rightPath)
	if err != nil {
		return true, "stat right: " + err.Error(), "", 0, time.Time{}
	}
	if leftInfo.Size() != rightInfo.Size() {
		return true, "size changed", "", leftInfo.Size(), leftInfo.ModTime().Truncate(time.Second)
	}
	leftModTime := leftInfo.ModTime().Truncate(time.Second)
	rightModTime := rightInfo.ModTime().Truncate(time.Second)
	if leftModTime.Equal(rightModTime) {
		return false, "", "", 0, time.Time{}
	}
	leftHash, err := hashFile(leftPath, hashAlg, threshold)
	if err != nil {
		return true, "hash left: " + err.Error(), "", leftInfo.Size(), leftModTime
	}
	rightHash, err := hashFile(rightPath, hashAlg, threshold)
	if err != nil {
		return true, "hash right: " + err.Error(), "", leftInfo.Size(), leftModTime
	}
	if leftHash == rightHash {
		return false, "", "", 0, time.Time{}
	}
	return true, "content differs", leftHash, leftInfo.Size(), leftModTime
}

// runWorkers starts n workers that read from pairCh, compare each pair, and send to resultCh.
func runWorkers(leftRoot, rightRoot string, n int, hashAlg string, threshold int, pairCh <-chan string, resultCh chan<- DiffResult, progress *progressCounts) {
	workCh := make(chan string, n*2)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rel := range workCh {
				diff, reason, hashStr, size, mtime := comparePair(leftRoot, rightRoot, rel, hashAlg, threshold)
				if diff {
					resultCh <- DiffResult{Rel: rel, Reason: reason, Hash: hashStr, Size: size, Mtime: mtime}
				}
				if progress != nil {
					atomic.AddInt32(&progress.processed, 1)
				}
			}
		}()
	}
	go func() {
		for rel := range pairCh {
			if progress != nil {
				atomic.AddInt32(&progress.enqueued, 1)
				// Record start time when first pair is enqueued (for time-remaining estimate).
				atomic.CompareAndSwapInt64(&progress.startTimeUnixNano, 0, time.Now().UnixNano())
			}
			workCh <- rel
		}
		close(workCh)
		wg.Wait()
		close(resultCh)
	}()
}

// progressCounts holds counters and start time for the progress indicator and time-remaining estimate.
type progressCounts struct {
	enqueued          int32
	processed         int32
	startTimeUnixNano int64 // set when first pair is enqueued; 0 means not yet started
}
