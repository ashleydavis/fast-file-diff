package lib

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestWalkTree_collectsRelativePaths(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a", "b"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a", "f1"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a", "b", "f2"), []byte("y"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".hidden"), []byte("z"), 0644); err != nil {
		t.Fatal(err)
	}
	var dirs, files []string
	var walk func(baseRoot, baseRel string)
	walk = func(baseRoot, baseRel string) {
		WalkTree(baseRoot, 4096, func(rel string, isDir bool) {
			fullRel := rel
			if baseRel != "" {
				fullRel = filepath.Join(baseRel, rel)
			}
			if isDir {
				dirs = append(dirs, fullRel)
				walk(filepath.Join(baseRoot, rel), fullRel)
			} else {
				files = append(files, fullRel)
			}
		})
	}
	walk(root, "")
	sort.Strings(dirs)
	sort.Strings(files)
	if len(dirs) < 2 {
		t.Errorf("expected at least 2 dirs (a, a/b), got %v", dirs)
	}
	if len(files) != 3 {
		t.Errorf("expected 3 files, got %v", files)
	}
	hasHidden := false
	for _, fileName := range files {
		if filepath.Base(fileName) == ".hidden" {
			hasHidden = true
			break
		}
	}
	if !hasHidden {
		t.Errorf("expected .hidden file, got files %v", files)
	}
}

func TestWalkTree_regularFilesOnly(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("f", filepath.Join(root, "link")); err != nil {
		t.Skip("symlink not supported")
	}
	var files []string
	var walk func(baseRoot, baseRel string)
	walk = func(baseRoot, baseRel string) {
		WalkTree(baseRoot, 4096, func(rel string, isDir bool) {
			fullRel := rel
			if baseRel != "" {
				fullRel = filepath.Join(baseRel, rel)
			}
			if !isDir {
				files = append(files, fullRel)
			} else {
				walk(filepath.Join(baseRoot, rel), fullRel)
			}
		})
	}
	walk(root, "")
	if len(files) != 1 || files[0] != "f" {
		t.Errorf("expected only regular file f, got %v", files)
	}
}
