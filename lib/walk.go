package lib

import (
	"io/fs"
	"path/filepath"
	"sync"
)

func walkTreePortable(root string, fn func(rel string, isDir bool)) {
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
			fn(rel, true)
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
		fn(rel, false)
		return nil
	})
}

func walkTree(root string, fn func(rel string, isDir bool)) {
	walkTreePortable(root, fn)
}

// WalkBothTrees walks left and right in parallel, logs every dir/file to logger,
// feeds the discovered set and sends pair relative paths to pairCh when both sides have the file.
func WalkBothTrees(leftRoot, rightRoot string, dirBatchSize int, log *Logger, set *DiscoveredSet, pairCh chan<- string) {
	var wg sync.WaitGroup
	walkOne := func(root string, sd Side) {
		defer wg.Done()
		walkTreeWithBatch(root, dirBatchSize, func(rel string, isDir bool) {
			if isDir {
				log.Log("dir: " + rel)
			} else {
				log.Log("file: " + rel)
				if set.Add(rel, sd) {
					pairCh <- rel
				}
			}
		})
	}
	wg.Add(2)
	go walkOne(leftRoot, SideLeft)
	go walkOne(rightRoot, SideRight)
	wg.Wait()
	close(pairCh)
}
