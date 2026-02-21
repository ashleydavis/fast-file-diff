package lib

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// WalkFileFunc is called for each file or directory: rel path, isDir, and for files only size and mtime (dirs pass 0, zero time).
type WalkFileFunc func(rel string, isDir bool, size int64, mtime time.Time)

// dirJob is a directory to scan for the worker-pool walk.
type dirJob struct {
	Root   string
	RelDir string
	Side   Side
}

// dirEntryInfo holds name and optional file metadata for one directory entry.
type dirEntryInfo struct {
	Name    string
	IsDir   bool
	Size    int64
	ModTime time.Time
}

// listDirEntries lists one directory and returns entries (name, isDir, and for files size/mtime). Skips . and .. and symlinks.
func listDirEntries(root, relDir string) ([]dirEntryInfo, error) {
	absPath := filepath.Join(root, relDir)
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}
	out := make([]dirEntryInfo, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if name == "." || name == ".." {
			continue
		}
		if e.IsDir() {
			out = append(out, dirEntryInfo{Name: name, IsDir: true})
			continue
		}
		if e.Type()&fs.ModeSymlink != 0 {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		out = append(out, dirEntryInfo{Name: name, IsDir: false, Size: info.Size(), ModTime: info.ModTime()})
	}
	return out, nil
}

func processDirJob(job dirJob, set *DiscoveredSet, log *Logger, dirCh chan dirJob, jobWg *sync.WaitGroup) {
	defer jobWg.Done()
	entries, err := listDirEntries(job.Root, job.RelDir)
	if err != nil {
		return
	}
	for _, ent := range entries {
		relPath := ent.Name
		if job.RelDir != "" {
			relPath = filepath.Join(job.RelDir, ent.Name)
		}
		if ent.IsDir {
			log.Log("dir: " + relPath)
			jobWg.Add(1)
			select {
			case dirCh <- dirJob{Root: job.Root, RelDir: relPath, Side: job.Side}:
			default:
				// Channel full: do this dir inline to avoid deadlock
				processDirJob(dirJob{Root: job.Root, RelDir: relPath, Side: job.Side}, set, log, dirCh, jobWg)
			}
		} else {
			log.Log("file: " + relPath)
			set.Add(relPath, job.Side, ent.Size, ent.ModTime)
		}
	}
}

// WalkBothTrees uses a worker pool to walk both trees in parallel. Queue is seeded with the two roots.
// Workers take directories from the queue, list them, add files to set, and enqueue subdirectories.
// When all directories are processed, doneCh is closed.
func WalkBothTrees(leftRoot, rightRoot string, dirBatchSize int, numWalkWorkers int, log *Logger, set *DiscoveredSet, doneCh chan struct{}) {
	if numWalkWorkers <= 0 {
		numWalkWorkers = 1
	}
	dirCh := make(chan dirJob, numWalkWorkers*4)
	var jobWg sync.WaitGroup
	jobWg.Add(2)
	go func() {
		dirCh <- dirJob{Root: leftRoot, RelDir: "", Side: SideLeft}
		dirCh <- dirJob{Root: rightRoot, RelDir: "", Side: SideRight}
	}()
	go func() {
		jobWg.Wait()
		close(dirCh)
	}()
	var workerWg sync.WaitGroup
	for i := 0; i < numWalkWorkers; i++ {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			for job := range dirCh {
				processDirJob(job, set, log, dirCh, &jobWg)
			}
		}()
	}
	workerWg.Wait()
	close(doneCh)
}

func walkTreePortable(root string, fn WalkFileFunc) {
	filepath.WalkDir(root, func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if dirEntry.IsDir() {
			fn(rel, true, 0, time.Time{})
			return nil
		}
		if dirEntry.Type() == fs.ModeSymlink {
			return nil
		}
		info, err := dirEntry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		fn(rel, false, info.Size(), info.ModTime())
		return nil
	})
}

func walkTree(root string, fn WalkFileFunc) {
	walkTreePortable(root, fn)
}
