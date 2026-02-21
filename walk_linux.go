//go:build linux

package main

import (
	"os"
	"path/filepath"
)

const defaultDirBatchSize = 4096

// walkTreeWithBatch on Linux uses batched directory reads (Readdir(n)) to reduce syscalls.
func walkTreeWithBatch(root string, batchSize int, fn func(rel string, isDir bool)) {
	if batchSize <= 0 {
		batchSize = defaultDirBatchSize
	}
	walkTreeBatched(root, "", root, batchSize, fn)
}

func walkTreeBatched(absRoot, relDir, root string, batchSize int, fn func(rel string, isDir bool)) {
	f, err := os.Open(absRoot)
	if err != nil {
		return
	}
	defer f.Close()
	for {
		entries, err := f.Readdir(batchSize)
		if err != nil {
			return
		}
		if len(entries) == 0 {
			break
		}
		for _, e := range entries {
			name := e.Name()
			if name == "." || name == ".." {
				continue
			}
			relPath := name
			if relDir != "" {
				relPath = filepath.Join(relDir, name)
			}
			if e.IsDir() {
				fn(relPath, true)
				subAbs := filepath.Join(absRoot, name)
				walkTreeBatched(subAbs, relPath, root, batchSize, fn)
				continue
			}
			if e.Mode()&os.ModeSymlink != 0 {
				continue
			}
			if e.Mode().IsRegular() {
				fn(relPath, false)
			}
		}
	}
}
