package lib

import (
	"os"
	"path/filepath"
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
	diff, _, _ := comparePair(left, right, "f", "xxhash", 10<<20)
	if diff {
		t.Error("comparePair(same file) = true; want same")
	}
}

// Size comparison is done in main before enqueueing; comparePair only hashes and compares hashes.

func TestComparePair_sameSizeDifferentMtime(t *testing.T) {
	root := t.TempDir()
	left := filepath.Join(root, "left")
	right := filepath.Join(root, "right")
	os.MkdirAll(left, 0755)
	os.MkdirAll(right, 0755)
	os.WriteFile(filepath.Join(left, "f"), []byte("aa"), 0644)
	time.Sleep(1 * time.Second)
	os.WriteFile(filepath.Join(right, "f"), []byte("bb"), 0644)
	diff, reason, hashStr := comparePair(left, right, "f", "xxhash", 10<<20)
	if !diff {
		t.Error("comparePair = false; want different")
	}
	if reason != "content differs" {
		t.Errorf("reason = %q", reason)
	}
	if hashStr == "" {
		t.Error("expected hash")
	}
}
