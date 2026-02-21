package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashFile_xxhash(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(f, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	h, err := hashFile(f, "xxhash", 10*1024*1024)
	if err != nil {
		t.Fatalf("hashFile: %v", err)
	}
	if h == "" || len(h) < 8 {
		t.Errorf("hash = %q", h)
	}
}

func TestHashFile_sha256(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f")
	os.WriteFile(f, []byte("hello"), 0644)
	h, err := hashFile(f, "sha256", 10*1024*1024)
	if err != nil {
		t.Fatalf("hashFile: %v", err)
	}
	// SHA256 hex is 64 chars
	if len(h) != 64 {
		t.Errorf("sha256 hex len = %d", len(h))
	}
}

func TestHashFile_sameContentSameHash(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(dir, "b"), []byte("x"), 0644)
	ha, _ := hashFile(filepath.Join(dir, "a"), "xxhash", 10<<20)
	hb, _ := hashFile(filepath.Join(dir, "b"), "xxhash", 10<<20)
	if ha != hb {
		t.Errorf("same content should have same hash: %q vs %q", ha, hb)
	}
}
