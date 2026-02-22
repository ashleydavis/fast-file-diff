//go:build linux

package lib

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Fallback batch size when caller passes <= 0; ReadDir(batchSize) uses fewer syscalls than reading one entry at a time.
const defaultDirBatchSize = 4096

// Entry point for Linux: uses batched ReadDir for better performance on large directories; ignores batchSize on non-Linux (see walk_nonlinux.go).
func walkTreeWithBatch(root string, batchSize int, walkFileFunc WalkFileFunc) {
	if batchSize <= 0 {
		batchSize = defaultDirBatchSize
	}
	walkTreeBatched(root, "", root, batchSize, walkFileFunc)
}

// Recursively lists absRoot in batches via File.ReadDir(batchSize), builds relative paths from relDir, and invokes walkFileFunc for each file/dir; skips symlinks and non-regular files like the portable path. Uses ReadDir (not Readdir) so lstat is only called for regular files we report, not for every directory entry.
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
				walkFileFunc(relPath, true, 0, time.Time{})
				subAbs := filepath.Join(absRoot, name)
				walkTreeBatched(subAbs, relPath, root, batchSize, walkFileFunc)
				continue
			}
			if entry.Type()&fs.ModeSymlink != 0 {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.Mode().IsRegular() {
				walkFileFunc(relPath, false, info.Size(), info.ModTime())
			}
		}
	}
}
