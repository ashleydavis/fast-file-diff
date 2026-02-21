package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePath_underRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a", "b"), 0755); err != nil {
		t.Fatal(err)
	}
	got, err := resolvePath(root, filepath.Join("a", "b"))
	if err != nil {
		t.Fatalf("resolvePath(%q, \"a/b\") err = %v", root, err)
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
		t.Fatalf("resolvePath(%q, \"\") err = %v", root, err)
	}
	if got != root {
		t.Errorf("resolvePath(empty rel) = %q, want %q", got, root)
	}
}

func TestResolvePath_rejectsEscapingRoot(t *testing.T) {
	root := t.TempDir()
	_, err := resolvePath(root, "..")
	if err == nil {
		t.Error("resolvePath(..) wanted error (path escapes root)")
	}
	_, err = resolvePath(root, filepath.Join("a", "..", ".."))
	if err == nil {
		t.Error("resolvePath(a/../..) wanted error")
	}
}

func TestPathUnder_underRoot(t *testing.T) {
	root := filepath.Clean(t.TempDir())
	sub := filepath.Join(root, "a", "b")
	if !pathUnder(sub, root) {
		t.Error("pathUnder(sub, root) want true")
	}
	if !pathUnder(root, root) {
		t.Error("pathUnder(root, root) want true")
	}
}

func TestPathUnder_escapesRoot(t *testing.T) {
	root := filepath.Clean(t.TempDir())
	escape := filepath.Join(root, "..", "other")
	if pathUnder(escape, root) {
		t.Error("pathUnder(escape, root) want false")
	}
}

func TestPathPool_Intern_dedupe(t *testing.T) {
	pool := newPathPool()
	a := pool.Intern("foo/bar")
	b := pool.Intern("foo/bar")
	if a != b {
		t.Errorf("Intern should return same string for same input")
	}
	if a != "foo/bar" {
		t.Errorf("Intern = %q", a)
	}
}
