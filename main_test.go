package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDir_validDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := ensureDir(dir); err != nil {
		t.Errorf("ensureDir(%q) = %v, want nil", dir, err)
	}
}

func TestEnsureDir_emptyPath(t *testing.T) {
	if err := ensureDir(""); err == nil {
		t.Error("ensureDir(\"\") = nil, want error")
	}
}

func TestEnsureDir_nonexistent(t *testing.T) {
	if err := ensureDir(filepath.Join(t.TempDir(), "nonexistent")); err == nil {
		t.Error("ensureDir(nonexistent) = nil, want error")
	}
}

func TestEnsureDir_fileNotDir(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file")
	if err := os.WriteFile(f, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ensureDir(f); err == nil {
		t.Error("ensureDir(file) = nil, want error")
	}
}
