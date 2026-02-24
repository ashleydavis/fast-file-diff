package lib

import (
	"path/filepath"
	"sync"
)

// Side indicates which tree (left or right) a path was seen on.
type Side int

const (
	SideLeft Side = iota
	SideRight
)

// DirJob is a directory to process; Root and RelDir identify the directory, Side is the label for the discovery set (e.g. left or right).
type DirJob struct {
	Root   string
	RelDir string
	Side   Side
}

// DiscoveredFile is sent on fileCh for each discovered regular file (relative path and which side it was seen on).
type DiscoveredFile struct {
	Rel  string
	Side Side
}

// Discover walks one or more directory trees in parallel using numWorkers. initialJobs seeds the queue (e.g. one job per root with RelDir ""). Each worker pulls directories from the queue, calls WalkTree for each; directories are pushed to the queue, and each discovered file is sent on fileCh. Discover closes fileCh when all directories are processed. The caller typically drains fileCh (e.g. into a DiscoveredSet) and closes a doneCh when the drain finishes. workerUtilization is Poked for progress; pass a positive dirBatchSize and numWorkers.
func Discover(initialJobs []DirJob, fileCh chan<- DiscoveredFile, dirBatchSize int, numWorkers int, workerUtilization *WorkerUtilization) {
	dirCh := make(chan DirJob, numWorkers*4)
	var jobWg sync.WaitGroup
	jobWg.Add(len(initialJobs))

	// Seed the queue in a goroutine so we don't block if there are many initial jobs or the buffer fills.
	go func() {
		for _, j := range initialJobs {
			dirCh <- j
		}
	}()

	// Close dirCh only after every job is done (jobWg reaches zero). Workers are the only other senders,
	// so once they've all finished their current jobs, no new jobs will be sent. This must run in its own
	// goroutine because main blocks on workerWg.Wait(); workers exit only after dirCh is closed.
	go func() {
		jobWg.Wait()
		close(dirCh)
	}()
	var workerWg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		idx := i
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			for job := range dirCh {
				dirPath := job.Root
				if job.RelDir != "" {
					dirPath = filepath.Join(job.Root, job.RelDir)
				}
				WalkTree(dirPath, dirBatchSize, func(rel string, isDir bool) {
					workerUtilization.Poke(idx)
					fullRel := rel
					if job.RelDir != "" {
						fullRel = filepath.Join(job.RelDir, rel)
					}
					if isDir {
						jobWg.Add(1)
						dirCh <- DirJob{Root: job.Root, RelDir: fullRel, Side: job.Side}
					} else {
						fileCh <- DiscoveredFile{Rel: fullRel, Side: job.Side}
					}
				})
				jobWg.Done()
			}
		}()
	}
	workerWg.Wait()
	close(fileCh)
}

// PairJob is a single pair to compare (relative path); compare phase stats both files.
type PairJob struct {
	Rel string
}

// DiscoveredSet tracks which relative paths have been seen on left and right.
// leftOnlyCount and rightOnlyCount are maintained in Add() so counts are O(1).
type DiscoveredSet struct {
	mu             sync.Mutex
	pool           *PathPool
	left           map[string]bool
	right          map[string]bool
	pairPaths      []string
	leftOnlyCount  int
	rightOnlyCount int
}

// NewDiscoveredSet returns a new discovered set using the given path pool.
func NewDiscoveredSet(pool *PathPool) *DiscoveredSet {
	return &DiscoveredSet{
		pool:  pool,
		left:  make(map[string]bool),
		right: make(map[string]bool),
	}
}

// Add records that rel was seen on the given side. Returns true when this completes a pair (the other side had already been seen for rel).
func (discoveredSet *DiscoveredSet) Add(rel string, side Side) bool {
	rel = discoveredSet.pool.Intern(filepath.Clean(filepath.ToSlash(rel)))
	discoveredSet.mu.Lock()
	defer discoveredSet.mu.Unlock()
	switch side {
	case SideLeft:
		if discoveredSet.right[rel] {
			firstTime := !discoveredSet.left[rel]
			discoveredSet.left[rel] = true
			if firstTime {
				discoveredSet.pairPaths = append(discoveredSet.pairPaths, rel)
				discoveredSet.rightOnlyCount-- // was right-only, now a pair
			}
			return firstTime
		}
		discoveredSet.left[rel] = true
		discoveredSet.leftOnlyCount++
		return false
	case SideRight:
		if discoveredSet.left[rel] {
			firstTime := !discoveredSet.right[rel]
			discoveredSet.right[rel] = true
			if firstTime {
				discoveredSet.pairPaths = append(discoveredSet.pairPaths, rel)
				discoveredSet.leftOnlyCount-- // was left-only, now a pair
			}
			return firstTime
		}
		discoveredSet.right[rel] = true
		discoveredSet.rightOnlyCount++
		return false
	default:
		return false
	}
}

// PairsCount returns the number of file pairs discovered so far (both sides seen).
func (discoveredSet *DiscoveredSet) PairsCount() int {
	return len(discoveredSet.pairPaths)
}

// LeftOnlyCount returns the number of paths seen on left but not on right. O(1).
func (discoveredSet *DiscoveredSet) LeftOnlyCount() int {
	return discoveredSet.leftOnlyCount
}

// RightOnlyCount returns the number of paths seen on right but not on left. O(1).
func (discoveredSet *DiscoveredSet) RightOnlyCount() int {
	return discoveredSet.rightOnlyCount
}

// PairPaths returns a copy of the relative paths that form pairs (seen on both sides), in discovery order.
func (discoveredSet *DiscoveredSet) PairPaths() []string {
	out := make([]string, len(discoveredSet.pairPaths))
	copy(out, discoveredSet.pairPaths)
	return out
}

// LeftOnlyPaths returns relative paths that were seen on left but not on right.
func (discoveredSet *DiscoveredSet) LeftOnlyPaths() []string {
	var out []string
	for rel := range discoveredSet.left {
		if !discoveredSet.right[rel] {
			out = append(out, rel)
		}
	}
	return out
}

// RightOnlyPaths returns relative paths that were seen on right but not on left.
func (discoveredSet *DiscoveredSet) RightOnlyPaths() []string {
	var out []string
	for rel := range discoveredSet.right {
		if !discoveredSet.left[rel] {
			out = append(out, rel)
		}
	}
	return out
}
