package main

import (
	"os"
	"path/filepath"
	"testing"
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
	diff, reason := comparePair(left, right, "f")
	if diff {
		t.Errorf("comparePair(same file) = true, reason %q; want same", reason)
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
	diff, reason := comparePair(left, right, "f")
	if !diff {
		t.Error("comparePair(different size) = false; want different")
	}
	if reason != "size changed" {
		t.Errorf("reason = %q, want size changed", reason)
	}
}
