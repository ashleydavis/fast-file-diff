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
	Rel    string
	Reason string
}

// comparePair stats both files; returns (true, reason) if different, (false, "") if same.
// Uses size and mtime only (same size + same mtime → same).
func comparePair(leftRoot, rightRoot, rel string) (different bool, reason string) {
	leftPath := filepath.Join(leftRoot, rel)
	rightPath := filepath.Join(rightRoot, rel)
	li, err := os.Stat(leftPath)
	if err != nil {
		return true, "stat left: " + err.Error()
	}
	ri, err := os.Stat(rightPath)
	if err != nil {
		return true, "stat right: " + err.Error()
	}
	if li.Size() != ri.Size() {
		return true, "size changed"
	}
	// Normalize mtime to seconds for cross-filesystem consistency
	lm := li.ModTime().Truncate(time.Second)
	rm := ri.ModTime().Truncate(time.Second)
	if !lm.Equal(rm) {
		return true, "mtime differs" // will trigger hash in next commit
	}
	return false, ""
}

// runWorkers starts n workers that read from pairCh, compare each pair, and send to resultCh.
// When pairCh is closed and all work is done, resultCh is closed.
func runWorkers(leftRoot, rightRoot string, n int, pairCh <-chan string, resultCh chan<- DiffResult, progress *progressCounts) {
	workCh := make(chan string, n*2)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rel := range workCh {
				diff, reason := comparePair(leftRoot, rightRoot, rel)
				if diff {
					resultCh <- DiffResult{Rel: rel, Reason: reason}
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
			}
			workCh <- rel
		}
		close(workCh)
		wg.Wait()
		close(resultCh)
	}()
}

type progressCounts struct {
	enqueued  int32
	processed int32
}
