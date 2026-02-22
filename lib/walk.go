package lib

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// WalkFileFunc is called for each file or directory: relative path and whether it is a directory.
type WalkFileFunc func(rel string, isDir bool)

// dirJob is a directory to scan for the worker-pool walk.
type dirJob struct {
	Root   string
	RelDir string
	Side   Side
}

// dirEntryInfo holds name and isDir for one directory entry (no size/mtime; compare phase stats when needed).
type dirEntryInfo struct {
	Name  string
	IsDir bool
}

// listDirEntries lists one directory and returns entries (name, isDir). Skips . and .. and symlinks. Does not call Info().
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
		if e.Type()&fs.ModeType != 0 {
			continue
		}
		out = append(out, dirEntryInfo{Name: name, IsDir: false})
	}
	return out, nil
}

// Lists one directory, adds files to set with their side/size/mtime, and enqueues subdirs (or runs them inline if dirCh is full to avoid deadlock).
// onDirProcessed must be non-nil; it is called at the start of the directory and once per entry so utilization reflects ongoing work.
func processDirJob(job dirJob, set *DiscoveredSet, log *Logger, dirCh chan dirJob, jobWg *sync.WaitGroup, onDirProcessed func()) {
	defer jobWg.Done()
	onDirProcessed()
	entries, err := listDirEntries(job.Root, job.RelDir)
	if err != nil {
		return
	}
	for _, ent := range entries {
		onDirProcessed()
		relPath := ent.Name
		if job.RelDir != "" {
			relPath = filepath.Join(job.RelDir, ent.Name)
		}
		if ent.IsDir {
			jobWg.Add(1)
			select {
			case dirCh <- dirJob{Root: job.Root, RelDir: relPath, Side: job.Side}:
			default:
				// Channel full: do this dir inline to avoid deadlock
				processDirJob(dirJob{Root: job.Root, RelDir: relPath, Side: job.Side}, set, log, dirCh, jobWg, onDirProcessed)
			}
		} else {
			set.Add(relPath, job.Side)
		}
	}
}

// WalkBothTrees uses a worker pool to walk both trees in parallel. Queue is seeded with the two roots.
// Workers take directories from the queue, list them, add files to set, and enqueue subdirectories.
// When all directories are processed, doneCh is closed.
// workerUtilization must be non-nil; workers Poke at the start of each directory and once per entry while listing, so utilization stays meaningful when directories have many files.
func WalkBothTrees(leftRoot, rightRoot string, dirBatchSize int, numWalkWorkers int, log *Logger, set *DiscoveredSet, doneCh chan struct{}, workerUtilization *WorkerUtilization) {
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
		idx := i
		workerWg.Add(1)
		onDir := func() { workerUtilization.Poke(idx) }
		go func() {
			defer workerWg.Done()
			for job := range dirCh {
				processDirJob(job, set, log, dirCh, &jobWg, onDir)
			}
		}()
	}
	workerWg.Wait()
	close(doneCh)
}

// defaultDirBatchSize is used when caller passes <= 0; ReadDir(batchSize) uses fewer syscalls than reading one entry at a time.
const defaultDirBatchSize = 4096

// walkTreeWithBatch walks root using batched ReadDir(batchSize), invokes walkFileFunc for each file and directory.
func walkTreeWithBatch(root string, batchSize int, walkFileFunc WalkFileFunc) {
	if batchSize <= 0 {
		batchSize = defaultDirBatchSize
	}
	walkTreeBatched(root, "", root, batchSize, walkFileFunc)
}

// walkTreeBatched recursively lists absRoot in batches via File.ReadDir(batchSize), builds relative paths from relDir, and invokes walkFileFunc for each file/dir; skips symlinks and non-regular files. Does not call Info(); compare phase stats when needed.
func walkTreeBatched(absRoot, relDir, root string, batchSize int, walkFileFunc WalkFileFunc) {
	dirFile, err := os.Open(absRoot)
	if err != nil {
		return
	}
	defer dirFile.Close()
	for {
		entries, err := dirFile.ReadDir(batchSize)
		if err != nil {
			return
		}
		if len(entries) == 0 {
			break
		}
		for _, entry := range entries {
			name := entry.Name()
			if name == "." || name == ".." {
				continue
			}
			relPath := name
			if relDir != "" {
				relPath = filepath.Join(relDir, name)
			}
			if entry.IsDir() {
				walkFileFunc(relPath, true)
				subAbs := filepath.Join(absRoot, name)
				walkTreeBatched(subAbs, relPath, root, batchSize, walkFileFunc)
				continue
			}
			if entry.Type()&fs.ModeSymlink != 0 {
				continue
			}
			if entry.Type()&fs.ModeType != 0 {
				continue
			}
			walkFileFunc(relPath, false)
		}
	}
}

// walkTree is the default entry for a single-tree walk; uses batched ReadDir via walkTreeWithBatch.
func walkTree(root string, walkFileFunc WalkFileFunc) {
	walkTreeWithBatch(root, 0, walkFileFunc)
}

// WalkTree traverses root recursively and calls walkFileFunc for each file and directory (relative path, isDir). Skips symlinks and non-regular files. Exported for testbeds and tools that need a single-tree walk.
func WalkTree(root string, walkFileFunc WalkFileFunc) {
	walkTree(root, walkFileFunc)
}
