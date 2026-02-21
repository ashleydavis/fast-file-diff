//go:build linux

package lib

import (
	"os"
	"path/filepath"
	"time"
)

// Fallback batch size when caller passes <= 0; Readdir(batchSize) uses fewer syscalls than reading one entry at a time.
const defaultDirBatchSize = 4096

// Entry point for Linux: uses batched Readdir for better performance on large directories; ignores batchSize on non-Linux (see walk_nonlinux.go).
func walkTreeWithBatch(root string, batchSize int, fn WalkFileFunc) {
	if batchSize <= 0 {
		batchSize = defaultDirBatchSize
	}
	walkTreeBatched(root, "", root, batchSize, fn)
}

// Recursively lists absRoot in batches via Readdir(batchSize), builds relative paths from relDir, and invokes fn for each file/dir; skips symlinks and non-regular files like the portable path.
func walkTreeBatched(absRoot, relDir, root string, batchSize int, fn WalkFileFunc) {
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
				fn(relPath, true, 0, time.Time{})
				subAbs := filepath.Join(absRoot, name)
				walkTreeBatched(subAbs, relPath, root, batchSize, fn)
				continue
			}
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}
			if entry.Mode().IsRegular() {
				fn(relPath, false, entry.Size(), entry.ModTime())
			}
		}
	}
}
