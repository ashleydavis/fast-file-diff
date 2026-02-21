package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePath_underRoot(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "a", "b"), 0755)
	got, err := resolvePath(root, filepath.Join("a", "b"))
	if err != nil {
		t.Fatalf("resolvePath err = %v", err)
	}
	want := filepath.Join(root, "a", "b")
	if got != want {
		t.Errorf("resolvePath = %q, want %q", got, want)
	}
}

func TestResolvePath_emptyRel(t *testing.T) {
	root := t.TempDir()
	got, err := resolvePath(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Errorf("resolvePath(empty rel) = %q, want %q", got, root)
	}
}

func TestResolvePath_rejectsEscapingRoot(t *testing.T) {
	root := t.TempDir()
	if _, err := resolvePath(root, ".."); err == nil {
		t.Error("resolvePath(..) wanted error")
	}
}

func TestPathUnder_underRoot(t *testing.T) {
	root := filepath.Clean(t.TempDir())
	sub := filepath.Join(root, "a", "b")
	if !pathUnder(sub, root) {
		t.Error("pathUnder want true")
	}
}

func TestPathPool_Intern_dedupe(t *testing.T) {
	pool := NewPathPool()
	a := pool.Intern("foo/bar")
	b := pool.Intern("foo/bar")
	if a != b {
		t.Error("Intern should return same string for same input")
	}
}

func TestEnsureDir_validDirectory(t *testing.T) {
	if err := EnsureDir(t.TempDir()); err != nil {
		t.Errorf("EnsureDir = %v", err)
	}
}

func TestEnsureDir_emptyPath(t *testing.T) {
	if err := EnsureDir(""); err == nil {
		t.Error("EnsureDir(\"\") want error")
	}
}
