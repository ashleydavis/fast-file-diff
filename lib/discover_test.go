package lib

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestDiscover_oneDir verifies that Discover with a single DirJob walks the tree and adds all files to the set on that side.
func TestDiscover_oneDir(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(root, "sub")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "c.txt"), []byte("c"), 0644); err != nil {
		t.Fatal(err)
	}

	pool := NewPathPool()
	set := NewDiscoveredSet(pool)
	fileCh := make(chan DiscoveredFile, 64)
	doneCh := make(chan struct{})
	go func() {
		for file := range fileCh {
			set.Add(file.Rel, file.Side)
		}
		close(doneCh)
	}()
	util := NewWorkerUtilization(2, 10)
	go Discover([]DirJob{{Root: root, RelDir: "", Side: SideLeft}}, fileCh, 10, 2, util)
	<-doneCh

	left := set.LeftOnlyPaths()
	sort.Strings(left)
	want := []string{"a.txt", "b.txt", filepath.Join("sub", "c.txt")}
	if len(left) != len(want) {
		t.Fatalf("LeftOnlyPaths: got %d paths, want %d: %v", len(left), len(want), left)
	}
	for i := range want {
		if left[i] != want[i] {
			t.Errorf("LeftOnlyPaths[%d]: got %q, want %q", i, left[i], want[i])
		}
	}
}

// TestDiscover_twoDirs verifies that Discover with two DirJobs (left and right roots) populates the set with pairs and left/right-only paths.
func TestDiscover_twoDirs(t *testing.T) {
	leftRoot := t.TempDir()
	rightRoot := t.TempDir()
	// Left: a.txt, b.txt, sub/c.txt
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(leftRoot, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(leftRoot, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(leftRoot, "sub", "c.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	// Right: a.txt (pair), sub/c.txt (pair), right-only.txt
	if err := os.WriteFile(filepath.Join(rightRoot, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(rightRoot, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rightRoot, "sub", "c.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rightRoot, "right-only.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	pool := NewPathPool()
	set := NewDiscoveredSet(pool)
	fileCh := make(chan DiscoveredFile, 64)
	doneCh := make(chan struct{})
	go func() {
		for file := range fileCh {
			set.Add(file.Rel, file.Side)
		}
		close(doneCh)
	}()
	util := NewWorkerUtilization(2, 10)
	go Discover(
		[]DirJob{
			{Root: leftRoot, RelDir: "", Side: SideLeft},
			{Root: rightRoot, RelDir: "", Side: SideRight},
		},
		fileCh, 10, 2, util)
	<-doneCh

	if got := set.PairsCount(); got != 2 {
		t.Errorf("PairsCount: got %d, want 2", got)
	}
	leftOnly := set.LeftOnlyPaths()
	if len(leftOnly) != 1 || leftOnly[0] != "b.txt" {
		t.Errorf("LeftOnlyPaths: got %v, want [b.txt]", leftOnly)
	}
	rightOnly := set.RightOnlyPaths()
	if len(rightOnly) != 1 || rightOnly[0] != "right-only.txt" {
		t.Errorf("RightOnlyPaths: got %v, want [right-only.txt]", rightOnly)
	}
}
