package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestEstimateRemainingFromElapsed(t *testing.T) {
	elapsed := 10 * time.Second
	// 10 processed in 10s => 1s per pair; 5 pending => 5s remaining
	got := estimateRemainingFromElapsed(elapsed, 10, 5)
	if got != 5*time.Second {
		t.Errorf("estimateRemainingFromElapsed(10s, 10, 5) = %v, want 5s", got)
	}
	// processed 0 => no estimate
	got = estimateRemainingFromElapsed(elapsed, 0, 5)
	if got != 0 {
		t.Errorf("estimateRemainingFromElapsed(10s, 0, 5) = %v, want 0", got)
	}
	// pending 0 => 0 remaining
	got = estimateRemainingFromElapsed(elapsed, 10, 0)
	if got != 0 {
		t.Errorf("estimateRemainingFromElapsed(10s, 10, 0) = %v, want 0", got)
	}
}
