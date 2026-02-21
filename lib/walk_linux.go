//go:build linux

package lib

import (
	"os"
	"path/filepath"
)

const defaultDirBatchSize = 4096

func walkTreeWithBatch(root string, batchSize int, fn func(rel string, isDir bool)) {
	if batchSize <= 0 {
		batchSize = defaultDirBatchSize
	}
	walkTreeBatched(root, "", root, batchSize, fn)
}

func walkTreeBatched(absRoot, relDir, root string, batchSize int, fn func(rel string, isDir bool)) {
	dirFile, err := os.Open(absRoot)
	if err != nil {
		return
	}
	defer dirFile.Close()
	for {
		entries, err := dirFile.Readdir(batchSize)
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
				fn(relPath, true)
				subAbs := filepath.Join(absRoot, name)
				walkTreeBatched(subAbs, relPath, root, batchSize, fn)
				continue
			}
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}
			if entry.Mode().IsRegular() {
				fn(relPath, false)
			}
		}
	}
}
