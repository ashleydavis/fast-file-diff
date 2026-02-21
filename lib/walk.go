package lib

import (
	"io/fs"
	"path/filepath"
	"sync"
	"time"
)

// WalkFileFunc is called for each file or directory: rel path, isDir, and for files only size and mtime (dirs pass 0, zero time).
type WalkFileFunc func(rel string, isDir bool, size int64, mtime time.Time)

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

// WalkBothTrees walks left and right in parallel, logs every dir/file to logger,
// and records discovered paths and cached size/mtime in set. When both walks are done, it closes doneCh.
func WalkBothTrees(leftRoot, rightRoot string, dirBatchSize int, log *Logger, set *DiscoveredSet, doneCh chan struct{}) {
	var wg sync.WaitGroup
	walkOne := func(root string, side Side) {
		defer wg.Done()
		walkTreeWithBatch(root, dirBatchSize, func(rel string, isDir bool, size int64, mtime time.Time) {
			if isDir {
				log.Log("dir: " + rel)
			} else {
				log.Log("file: " + rel)
				set.Add(rel, side, size, mtime)
			}
		})
	}
	wg.Add(2)
	go walkOne(leftRoot, SideLeft)
	go walkOne(rightRoot, SideRight)
	wg.Wait()
	close(doneCh)
}
