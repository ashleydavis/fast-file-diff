package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashFile_xxhash(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	hash, err := hashFile(filePath, "xxhash", 10*1024*1024)
	if err != nil {
		t.Fatalf("hashFile: %v", err)
	}
	if hash == "" || len(hash) < 8 {
		t.Errorf("hash = %q", hash)
	}
}

func TestHashFile_sha256(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "f")
	os.WriteFile(filePath, []byte("hello"), 0644)
	hash, err := hashFile(filePath, "sha256", 10*1024*1024)
	if err != nil {
		t.Fatalf("hashFile: %v", err)
	}
	if len(hash) != 64 {
		t.Errorf("sha256 hex len = %d", len(hash))
	}
}

func TestHashFile_sameContentSameHash(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(dir, "b"), []byte("x"), 0644)
	hashA, _ := hashFile(filepath.Join(dir, "a"), "xxhash", 10<<20)
	hashB, _ := hashFile(filepath.Join(dir, "b"), "xxhash", 10<<20)
	if hashA != hashB {
		t.Errorf("same content should have same hash: %q vs %q", hashA, hashB)
	}
}

func TestHashBytes_xxhash(t *testing.T) {
	data := []byte("hello")
	hash, err := hashBytes(data, "xxhash")
	if err != nil {
		t.Fatal(err)
	}
	if len(hash) != 16 {
		t.Errorf("xxhash hex len = %d", len(hash))
	}
	duplicateHash, _ := hashBytes(data, "xxhash")
	if hash != duplicateHash {
		t.Error("hashBytes same input should give same output")
	}
}

func TestHashBytes_md5(t *testing.T) {
	data := []byte("hello")
	hash, err := hashBytes(data, "md5")
	if err != nil {
		t.Fatal(err)
	}
	if len(hash) != 32 {
		t.Errorf("md5 hex len = %d", len(hash))
	}
}
