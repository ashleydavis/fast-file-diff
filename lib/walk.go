package lib

import (
	"io/fs"
	"os"
	"path/filepath"
)

// WalkFileFunc is called for each file or directory: relative path and whether it is a directory.
type WalkFileFunc func(rel string, isDir bool)

// WalkTreeBatched processes one directory: opens root/relDir, reads entries in batches via File.ReadDir(batchSize) (the only ReadDir in the program), and calls walkFileFunc for every entry discovered. Does not recurse; does not know about queues or workers. Caller must pass a positive batchSize.
func walkTreeBatched(root, relDir string, batchSize int, walkFileFunc WalkFileFunc) {
	absPath := filepath.Join(root, relDir)
	dirFile, err := os.Open(absPath)
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

// WalkTree lists the root directory only (root/relDir with relDir empty) and calls walkFileFunc for each entry. Does not recurse; the caller recurses if needed. batchSize must be positive (e.g. from CLI --dir-batch-size).
func WalkTree(root string, batchSize int, walkFileFunc WalkFileFunc) {
	walkTreeBatched(root, "", batchSize, walkFileFunc)
}
