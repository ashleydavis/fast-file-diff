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
