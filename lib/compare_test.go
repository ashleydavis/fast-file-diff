package lib

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestComparePair_sameFile(t *testing.T) {
	root := t.TempDir()
	left := filepath.Join(root, "left")
	right := filepath.Join(root, "right")
	os.MkdirAll(left, 0755)
	os.MkdirAll(right, 0755)
	content := []byte("same")
	os.WriteFile(filepath.Join(left, "f"), content, 0644)
	os.WriteFile(filepath.Join(right, "f"), content, 0644)
	diff, _, _, _, _, _, _, _ := comparePair(left, right, "f", "xxhash", 10<<20)
	if diff {
		t.Error("comparePair(same file) = true; want same")
	}
}

// comparePair stats both files; if size or mtime differ it may skip hash or report size changed.

func TestComparePair_sameSizeDifferentMtime(t *testing.T) {
	root := t.TempDir()
	left := filepath.Join(root, "left")
	right := filepath.Join(root, "right")
	os.MkdirAll(left, 0755)
	os.MkdirAll(right, 0755)
	os.WriteFile(filepath.Join(left, "f"), []byte("aa"), 0644)
	time.Sleep(1 * time.Second)
	os.WriteFile(filepath.Join(right, "f"), []byte("bb"), 0644)
	diff, reason, lHash, rHash, _, _, _, _ := comparePair(left, right, "f", "xxhash", 10<<20)
	if !diff {
		t.Error("comparePair = false; want different")
	}
	if reason != "content differs" {
		t.Errorf("reason = %q", reason)
	}
	if lHash == "" || rHash == "" {
		t.Error("expected both hashes")
	}
}

// TestCompare_onePair verifies that Compare processes one pair path and sends one CompareResult to resultCh.
func TestCompare_onePair(t *testing.T) {
	root := t.TempDir()
	left := filepath.Join(root, "left")
	right := filepath.Join(root, "right")
	os.MkdirAll(left, 0755)
	os.MkdirAll(right, 0755)
	os.WriteFile(filepath.Join(left, "f"), []byte("same"), 0644)
	os.WriteFile(filepath.Join(right, "f"), []byte("same"), 0644)

	resultCh := make(chan CompareResult, 1)
	progress := &ProgressCounts{}
	util := NewWorkerUtilization(1, 10)
	go Compare(left, right, []string{"f"}, 1, "xxhash", 10<<20, resultCh, progress, util)

	var result CompareResult
	select {
	case result = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for CompareResult")
	}
	if result.RelativePath != "f" {
		t.Errorf("RelativePath = %q, want f", result.RelativePath)
	}
	if result.Diff != nil {
		t.Errorf("expected identical pair, got Diff: %v", result.Diff)
	}
	// Drain remaining results until channel closed
	for range resultCh {
	}
	if atomic.LoadInt32(&progress.Processed) != 1 {
		t.Errorf("Processed = %d, want 1", atomic.LoadInt32(&progress.Processed))
	}
}
