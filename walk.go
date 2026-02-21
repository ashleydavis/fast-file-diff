package main

import (
	"io/fs"
	"path/filepath"
)

// walkTree walks root recursively and calls fn for each directory and regular file
// (relative path, isDir). Includes hidden/dotfiles. Regular files only for files.
func walkTree(root string, fn func(rel string, isDir bool)) {
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
		if d.IsDir() {
			fn(rel, true)
			return nil
		}
		if d.Type() == fs.ModeSymlink {
			return nil
		}
		info, err := d.Info()
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
