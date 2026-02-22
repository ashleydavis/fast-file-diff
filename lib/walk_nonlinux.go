//go:build !linux

package lib

import (
	"io/fs"
	"path/filepath"
	"time"
)

// walkTreePortable walks root with filepath.WalkDir, calls walkFileFunc for each file and dir with relative path and metadata; skips symlinks and non-regular files. Used on non-Linux so behavior is consistent everywhere.
func walkTreePortable(root string, walkFileFunc WalkFileFunc) {
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
			walkFileFunc(rel, true, 0, time.Time{})
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
		walkFileFunc(rel, false, info.Size(), info.ModTime())
		return nil
	})
}

// On non-Linux, batch size is ignored and we use walkTreePortable so behavior is consistent without Linux-specific Readdir batching.
func walkTreeWithBatch(root string, _ int, walkFileFunc WalkFileFunc) {
	walkTreePortable(root, walkFileFunc)
}
