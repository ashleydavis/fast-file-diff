package main

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
	if err := os.MkdirAll(left, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(right, 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte("same")
	if err := os.WriteFile(filepath.Join(left, "f"), content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(right, "f"), content, 0644); err != nil {
		t.Fatal(err)
	}
	diff, _, _, _, _ := comparePair(left, right, "f", "xxhash", 10<<20)
	if diff {
		t.Error("comparePair(same file) = true; want same")
	}
}

func TestComparePair_differentSize(t *testing.T) {
	root := t.TempDir()
	left := filepath.Join(root, "left")
	right := filepath.Join(root, "right")
	os.MkdirAll(left, 0755)
	os.MkdirAll(right, 0755)
	os.WriteFile(filepath.Join(left, "f"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(right, "f"), []byte("ab"), 0644)
	diff, reason, _, _, _ := comparePair(left, right, "f", "xxhash", 10<<20)
	if !diff {
		t.Error("comparePair(different size) = false; want different")
	}
	if reason != "size changed" {
		t.Errorf("reason = %q, want size changed", reason)
	}
}

func TestComparePair_sameSizeDifferentMtime(t *testing.T) {
	root := t.TempDir()
	left := filepath.Join(root, "left")
	right := filepath.Join(root, "right")
	os.MkdirAll(left, 0755)
	os.MkdirAll(right, 0755)
	os.WriteFile(filepath.Join(left, "f"), []byte("aa"), 0644)
	time.Sleep(1 * time.Second)
	os.WriteFile(filepath.Join(right, "f"), []byte("bb"), 0644) // same size, different content
	diff, reason, hashStr, _, _ := comparePair(left, right, "f", "xxhash", 10<<20)
	if !diff {
		t.Error("comparePair(same size, different mtime) = false; want different")
	}
	if reason != "content differs" {
		t.Errorf("reason = %q, want content differs", reason)
	}
	if hashStr == "" {
		t.Error("expected hash for content differs")
	}
}
